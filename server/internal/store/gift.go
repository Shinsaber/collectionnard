package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shinsaber/colec/internal/model"
)

// Gift reservations (P7-4). A reservation not handed over is never read for
// the owner of the list: every function answers a giver or a friend of the
// owner, except GivenGifts, ReceiveGift and DeclineGift, which only touch
// gifts handed over (P7-4b: the surprise is over).

// reservationActive: a reservation that counts, handed over or not expired.
const reservationActive = `(r.status <> 'reserved' OR r.expires_at >= current_date)`

const reservationSelect = `SELECT r.id, r.lot_id, ci.collection_id, r.item_id, r.quantity, r.status, r.expires_at::text,
	NOT ` + reservationActive + `, r.given_at, r.received_at, r.note, r.created_at, r.updated_at,
	r.giver_id, g.username, g.display_name, g.avatar,
	r.owner_id, o.username, o.display_name, o.avatar,
	i.item_type_id, i.attributes, it.name, it.attributes_schema
	FROM gift_reservations r
	JOIN users g ON g.id = r.giver_id
	JOIN users o ON o.id = r.owner_id
	JOIN items i ON i.id = r.item_id
	JOIN item_types it ON it.id = i.item_type_id
	LEFT JOIN collection_items ci ON ci.id = r.lot_id`

func scanReservation(row pgx.Row) (*model.GiftReservation, error) {
	var r model.GiftReservation
	var g, o model.ReviewAuthor
	var i model.Item
	var t model.ItemType
	if err := row.Scan(&r.ID, &r.LotID, &r.CollectionID, &r.ItemID, &r.Quantity, &r.Status, &r.ExpiresAt, &r.Expired,
		&r.GivenAt, &r.ReceivedAt, &r.Note, &r.CreatedAt, &r.UpdatedAt,
		&r.GiverID, &g.Username, &g.DisplayName, &g.Avatar,
		&r.OwnerID, &o.Username, &o.DisplayName, &o.Avatar,
		&i.ItemTypeID, &i.Attributes, &t.Name, &t.AttributesSchema); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	r.LeftList = r.LotID == nil
	r.Giver, r.Owner = &g, &o
	i.ID, t.ID = r.ItemID, i.ItemTypeID
	i.ItemType = &t
	r.Item = &i
	return &r, nil
}

func collectReservations(rows pgx.Rows, err error) ([]model.GiftReservation, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.GiftReservation{}
	for rows.Next() {
		r, err := scanReservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// reservedCopies sums the reservations that count of a lot, except the reservation except.
func reservedCopies(ctx context.Context, q rowQuerier, lotID uuid.UUID, except *uuid.UUID) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT COALESCE(sum(quantity), 0) FROM gift_reservations r
		WHERE r.lot_id = $1 AND ($2::uuid IS NULL OR r.id <> $2) AND `+reservationActive, lotID, except).Scan(&n)
	return n, err
}

// CreateReservation reserves quantity copies of a lot for giverID until
// expiresAt (ErrInvalidQuantity beyond the copies not reserved yet).
func (s *PgStore) CreateReservation(ctx context.Context, lotID, giverID uuid.UUID, quantity int, expiresAt, note string) (*model.GiftReservation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var lotQuantity int
	var itemID, ownerID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT ci.quantity, ci.item_id, c.user_id FROM collection_items ci
		JOIN collections c ON c.id = ci.collection_id WHERE ci.id = $1 FOR UPDATE OF ci`, lotID).Scan(&lotQuantity, &itemID, &ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	reserved, err := reservedCopies(ctx, tx, lotID, nil)
	if err != nil {
		return nil, err
	}
	if reserved+quantity > lotQuantity {
		return nil, ErrInvalidQuantity
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO gift_reservations (lot_id, item_id, owner_id, giver_id, quantity, expires_at, note)
		VALUES ($1, $2, $3, $4, $5, $6::date, $7) RETURNING id`, lotID, itemID, ownerID, giverID, quantity, expiresAt, note).Scan(&id); err != nil {
		return nil, err
	}
	r, err := scanReservation(tx.QueryRow(ctx, reservationSelect+` WHERE r.id = $1`, id))
	if err != nil {
		return nil, err
	}
	return r, tx.Commit(ctx)
}

// GetReservation returns a reservation (the handler checks it is the reader's).
func (s *PgStore) GetReservation(ctx context.Context, id uuid.UUID) (*model.GiftReservation, error) {
	return scanReservation(s.pool.QueryRow(ctx, reservationSelect+` WHERE r.id = $1`, id))
}

// UpdateReservation replaces the quantity, due date and note of a reservation
// not handed over (the handler checks); once it counts again, its quantity
// still fits in its lot (a lot gone accepts anything). A new due date allows a
// new reminder.
func (s *PgStore) UpdateReservation(ctx context.Context, id uuid.UUID, in model.ReservationUpdate) (*model.GiftReservation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var lotID *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT lot_id FROM gift_reservations WHERE id = $1`, id).Scan(&lotID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var counts bool
	if err := tx.QueryRow(ctx, `SELECT $1::date >= current_date`, in.ExpiresAt).Scan(&counts); err != nil {
		return nil, err
	}
	if lotID != nil && counts {
		var lotQuantity int
		if err := tx.QueryRow(ctx, `SELECT quantity FROM collection_items WHERE id = $1 FOR UPDATE`, *lotID).Scan(&lotQuantity); err != nil {
			return nil, err
		}
		reserved, err := reservedCopies(ctx, tx, *lotID, &id)
		if err != nil {
			return nil, err
		}
		if reserved+in.Quantity > lotQuantity {
			return nil, ErrInvalidQuantity
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE gift_reservations SET quantity = $2, note = $3, expires_at = $4::date,
			expiry_reminded_at = CASE WHEN expires_at = $4::date THEN expiry_reminded_at END, updated_at = now()
		WHERE id = $1`, id, in.Quantity, in.Note, in.ExpiresAt); err != nil {
		return nil, err
	}
	r, err := scanReservation(tx.QueryRow(ctx, reservationSelect+` WHERE r.id = $1`, id))
	if err != nil {
		return nil, err
	}
	return r, tx.Commit(ctx)
}

// DeleteReservation cancels a reservation.
func (s *PgStore) DeleteReservation(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM gift_reservations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// LotReservations returns the reservations that count of the given lots, by
// lot, givers of disabled accounts excepted. For a friend of the owner only.
func (s *PgStore) LotReservations(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error) {
	out := map[uuid.UUID][]model.GiftReservation{}
	if len(lotIDs) == 0 {
		return out, nil
	}
	list, err := collectReservations(s.pool.Query(ctx, reservationSelect+` WHERE r.lot_id = ANY($1) AND g.disabled_at IS NULL AND `+reservationActive+`
		ORDER BY r.created_at, r.id`, lotIDs))
	if err != nil {
		return nil, err
	}
	for _, r := range list {
		out[*r.LotID] = append(out[*r.LotID], r)
	}
	return out, nil
}

// giftStatus: "open" (to give, or handed over), "received", or "" (all).
func giftStatus(status string) string {
	switch status {
	case "open":
		return ` AND r.status IN ('reserved', 'given')`
	case "received":
		return ` AND r.status = 'received'`
	}
	return ``
}

// ListGifts lists the reservations of giverID: those whose lot left the list
// first, then the expired ones, the ones to give, the ones handed over, and
// the received ones; newest first.
func (s *PgStore) ListGifts(ctx context.Context, giverID uuid.UUID, status string, pg model.Pagination) ([]model.GiftReservation, int, error) {
	where := ` WHERE r.giver_id = $1` + giftStatus(status)
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM gift_reservations r`+where, giverID).Scan(&total); err != nil {
		return nil, 0, err
	}
	list, err := collectReservations(s.pool.Query(ctx, reservationSelect+where+`
		ORDER BY CASE WHEN r.status = 'reserved' AND r.lot_id IS NULL THEN 0
			WHEN r.status = 'reserved' AND r.expires_at < current_date THEN 1
			WHEN r.status = 'reserved' THEN 2 WHEN r.status = 'given' THEN 3 ELSE 4 END,
			r.created_at DESC, r.id LIMIT $2 OFFSET $3`,
		giverID, pg.PerPage, pg.Offset()))
	return list, total, err
}

// LeftReservations lists the reservations still to give whose lot left the
// list, their giver not told yet.
func (s *PgStore) LeftReservations(ctx context.Context) ([]model.GiftReservation, error) {
	return collectReservations(s.pool.Query(ctx, reservationSelect+` WHERE r.lot_id IS NULL AND r.notified_at IS NULL
		AND r.status = 'reserved' AND g.disabled_at IS NULL ORDER BY r.updated_at`))
}

// MarkReservationsNotified records that their giver was told.
func (s *PgStore) MarkReservationsNotified(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE gift_reservations SET notified_at = now() WHERE id = ANY($1)`, ids)
	return err
}

// ExpiringReservations lists the reservations not given whose lot is still on
// the list and whose due date comes within the notice, their giver not told yet.
func (s *PgStore) ExpiringReservations(ctx context.Context) ([]model.GiftReservation, error) {
	return collectReservations(s.pool.Query(ctx, reservationSelect+` WHERE r.status = 'reserved' AND r.lot_id IS NOT NULL
		AND r.expiry_reminded_at IS NULL AND r.expires_at BETWEEN current_date AND current_date + $1::int
		AND g.disabled_at IS NULL ORDER BY r.expires_at`, model.GiftExpiryNoticeDays))
}

// MarkReservationsExpiryReminded records the reminder of the current due date.
func (s *PgStore) MarkReservationsExpiryReminded(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE gift_reservations SET expiry_reminded_at = now() WHERE id = ANY($1)`, ids)
	return err
}

// HandOverReservation marks a reservation handed over (ErrConflict unless it
// is reserved). Its lot gone, there is nothing left for the owner to confirm:
// it is received at once.
func (s *PgStore) HandOverReservation(ctx context.Context, id uuid.UUID) (*model.GiftReservation, error) {
	ct, err := s.pool.Exec(ctx, `UPDATE gift_reservations
		SET status = CASE WHEN lot_id IS NULL THEN 'received' ELSE 'given' END, given_at = now(),
			received_at = CASE WHEN lot_id IS NULL THEN now() END, updated_at = now()
		WHERE id = $1 AND status = 'reserved'`, id)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrConflict
	}
	return s.GetReservation(ctx, id)
}

// GivenGifts returns the gifts handed over on the given lots, waiting for the
// owner to confirm them: the only reservations the owner ever reads.
func (s *PgStore) GivenGifts(ctx context.Context, lotIDs []uuid.UUID) (map[uuid.UUID][]model.GiftReservation, error) {
	out := map[uuid.UUID][]model.GiftReservation{}
	if len(lotIDs) == 0 {
		return out, nil
	}
	list, err := collectReservations(s.pool.Query(ctx, reservationSelect+` WHERE r.lot_id = ANY($1) AND r.status = 'given'
		ORDER BY r.given_at, r.id`, lotIDs))
	if err != nil {
		return nil, err
	}
	for _, r := range list {
		out[*r.LotID] = append(out[*r.LotID], r)
	}
	return out, nil
}

// ReceiveGift confirms a gift handed over: its copies leave the list (the lot
// loses them, or goes), into another collection when into is set (a new lot
// with attrs), and the reservation is only the giver's history from then on.
// ErrNotFound unless the gift is handed over.
func (s *PgStore) ReceiveGift(ctx context.Context, id uuid.UUID, into *uuid.UUID, attrs json.RawMessage) (*model.GiftReservation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var lotID *uuid.UUID
	var itemID uuid.UUID
	var quantity int
	err = tx.QueryRow(ctx, `SELECT lot_id, item_id, quantity FROM gift_reservations
		WHERE id = $1 AND status = 'given' FOR UPDATE`, id).Scan(&lotID, &itemID, &quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lotID != nil {
		var lotQuantity int
		if err := tx.QueryRow(ctx, `SELECT quantity FROM collection_items WHERE id = $1 FOR UPDATE`, *lotID).Scan(&lotQuantity); err != nil {
			return nil, err
		}
		if lotQuantity <= quantity {
			// Les autres réservations du lot le voient quitter la liste (lot_id NULL) : leur offrant sera prévenu.
			_, err = tx.Exec(ctx, `DELETE FROM collection_items WHERE id = $1`, *lotID)
		} else {
			_, err = tx.Exec(ctx, `UPDATE collection_items SET quantity = quantity - $2 WHERE id = $1`, *lotID, quantity)
		}
		if err != nil {
			return nil, err
		}
	}
	if into != nil {
		if len(attrs) == 0 {
			attrs = json.RawMessage(`{}`)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO collection_items (collection_id, item_id, attributes, quantity)
			VALUES ($1, $2, $3, $4)`, *into, itemID, attrs, quantity); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE gift_reservations SET status = 'received', received_at = now(), lot_id = NULL,
		updated_at = now() WHERE id = $1`, id); err != nil {
		return nil, err
	}
	r, err := scanReservation(tx.QueryRow(ctx, reservationSelect+` WHERE r.id = $1`, id))
	if err != nil {
		return nil, err
	}
	return r, tx.Commit(ctx)
}

// DeclineGift invalidates a gift handed over that the owner did not receive.
func (s *PgStore) DeclineGift(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM gift_reservations WHERE id = $1 AND status = 'given'`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
