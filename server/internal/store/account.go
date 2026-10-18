package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// ErrLastAdmin: the change would leave the instance without an active admin.
var ErrLastAdmin = errors.New("last active admin")

// ── Accounts (admin) ────────────────────────────────────────────────

// ListUsers returns the accounts by username.
func (s *PgStore) ListUsers(ctx context.Context, pg model.Pagination) ([]model.User, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY lower(username) LIMIT $1 OFFSET $2`,
		pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *u)
	}
	return out, total, rows.Err()
}

// UpdateUser changes the role and / or the disabled state of an account. Disabling
// revokes its refresh tokens. ErrLastAdmin when no active admin would remain.
func (s *PgStore) UpdateUser(ctx context.Context, id uuid.UUID, in model.UserUpdate) (*model.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Lock the admins: two concurrent demotions cannot both pass the check.
	if _, err := tx.Exec(ctx, `SELECT id FROM users WHERE role = 'admin' FOR UPDATE`); err != nil {
		return nil, err
	}
	u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1 FOR UPDATE`, id))
	if err != nil {
		return nil, err
	}
	role := u.Role
	if in.Role != nil {
		role = *in.Role
	}
	disabled := u.DisabledAt != nil
	if in.Disabled != nil {
		disabled = *in.Disabled
	}
	if u.Role == model.RoleAdmin && u.DisabledAt == nil && (role != model.RoleAdmin || disabled) {
		var others int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE role = 'admin' AND disabled_at IS NULL AND id <> $1`,
			id).Scan(&others); err != nil {
			return nil, err
		}
		if others == 0 {
			return nil, ErrLastAdmin
		}
	}
	u, err = scanUser(tx.QueryRow(ctx, `
		UPDATE users SET role = $2,
			disabled_at = CASE WHEN $3 THEN COALESCE(disabled_at, now()) ELSE NULL END,
			updated_at = now()
		WHERE id = $1 RETURNING `+userColumns, id, role, disabled))
	if err != nil {
		return nil, err
	}
	if disabled {
		if _, err := tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, id); err != nil {
			return nil, err
		}
	}
	return u, tx.Commit(ctx)
}

// UpdateProfile replaces the profile of a user (empty string = cleared).
func (s *PgStore) UpdateProfile(ctx context.Context, id uuid.UUID, in model.ProfileUpdate) (*model.User, error) {
	return scanUser(s.pool.QueryRow(ctx, `
		UPDATE users SET display_name = NULLIF($2, ''), bio = NULLIF($3, ''), avatar = NULLIF($4, ''),
			profile_visibility = COALESCE(NULLIF($5, ''), 'private'), updated_at = now()
		WHERE id = $1 RETURNING `+userColumns, id, in.DisplayName, in.Bio, in.Avatar, in.ProfileVisibility))
}

// SetPassword changes the password of a user and revokes all its refresh tokens.
func (s *PgStore) SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE users SET password = $2, updated_at = now() WHERE id = $1`, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── Invitations ─────────────────────────────────────────────────────

const invitationColumns = `i.id, i.role, i.note, i.created_by, COALESCE(c.username, ''), i.expires_at, i.used_at,
	COALESCE(u.username, ''), i.created_at`

const invitationFrom = ` FROM invitations i LEFT JOIN users c ON c.id = i.created_by LEFT JOIN users u ON u.id = i.used_by`

func scanInvitation(row pgx.Row) (*model.Invitation, error) {
	var inv model.Invitation
	if err := row.Scan(&inv.ID, &inv.Role, &inv.Note, &inv.CreatedBy, &inv.CreatedByUsername, &inv.ExpiresAt,
		&inv.UsedAt, &inv.UsedByUsername, &inv.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	switch {
	case inv.UsedAt != nil:
		inv.Status = model.InvitationUsed
	case !inv.ExpiresAt.After(time.Now()):
		inv.Status = model.InvitationExpired
	default:
		inv.Status = model.InvitationPending
	}
	return &inv, nil
}

// CreateInvitation stores an invitation (hash of its token only).
func (s *PgStore) CreateInvitation(ctx context.Context, tokenHash string, createdBy uuid.UUID, role, note string, expiresAt time.Time) (*model.Invitation, error) {
	var id uuid.UUID
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO invitations (token_hash, created_by, role, note, expires_at) VALUES ($1, $2, $3, NULLIF($4, ''), $5)
		RETURNING id`, tokenHash, createdBy, role, note, expiresAt).Scan(&id); err != nil {
		return nil, err
	}
	return s.GetInvitation(ctx, id)
}

// GetInvitation returns one invitation.
func (s *PgStore) GetInvitation(ctx context.Context, id uuid.UUID) (*model.Invitation, error) {
	return scanInvitation(s.pool.QueryRow(ctx, `SELECT `+invitationColumns+invitationFrom+` WHERE i.id = $1`, id))
}

// ListInvitations returns the invitations of a creator (all when createdBy is nil), most recent first.
func (s *PgStore) ListInvitations(ctx context.Context, createdBy *uuid.UUID) ([]model.Invitation, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+invitationColumns+invitationFrom+`
		WHERE $1::uuid IS NULL OR i.created_by = $1 ORDER BY i.created_at DESC LIMIT 500`, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Invitation{}
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

// DeleteInvitation withdraws an unused invitation (ErrConflict once used).
func (s *PgStore) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	var used bool
	err := s.pool.QueryRow(ctx, `DELETE FROM invitations WHERE id = $1 AND used_at IS NULL RETURNING false`, id).Scan(&used)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := s.GetInvitation(ctx, id); err != nil {
			return err
		}
		return ErrConflict
	}
	return err
}

// CheckInvitation returns the expiry of a still usable invitation (ErrNotFound otherwise).
func (s *PgStore) CheckInvitation(ctx context.Context, tokenHash string) (time.Time, error) {
	var exp time.Time
	err := s.pool.QueryRow(ctx, `SELECT expires_at FROM invitations
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()`, tokenHash).Scan(&exp)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	return exp, err
}

// RegisterWithInvitation creates an account with the role of a usable
// invitation and consumes it, in one transaction. ErrNotFound: invitation
// unknown, used or expired; ErrConflict: username or email taken.
func (s *PgStore) RegisterWithInvitation(ctx context.Context, tokenHash, username, email, passwordHash string) (*model.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var invID uuid.UUID
	var role string
	err = tx.QueryRow(ctx, `SELECT id, role FROM invitations
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now() FOR UPDATE`, tokenHash).Scan(&invID, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u, err := scanUser(tx.QueryRow(ctx, `INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4)
		RETURNING `+userColumns, username, email, passwordHash, role))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE invitations SET used_at = now(), used_by = $2 WHERE id = $1`, invID, u.ID); err != nil {
		return nil, err
	}
	return u, tx.Commit(ctx)
}

// ── Password resets ─────────────────────────────────────────────────

// CreatePasswordReset stores a reset link for a user; its previous unused links are cancelled.
func (s *PgStore) CreatePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, createdBy uuid.UUID, expiresAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM password_resets WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_resets (user_id, token_hash, created_by, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, tokenHash, createdBy, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ResetPassword consumes a usable reset link of an enabled account, sets the
// password and revokes the refresh tokens (ErrNotFound for any unusable link).
func (s *PgStore) ResetPassword(ctx context.Context, tokenHash, passwordHash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE password_resets r SET used_at = now()
		FROM users u
		WHERE r.token_hash = $1 AND r.used_at IS NULL AND r.expires_at > now() AND u.id = r.user_id AND u.disabled_at IS NULL
		RETURNING r.user_id`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password = $2, updated_at = now() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
