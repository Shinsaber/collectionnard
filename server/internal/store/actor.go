package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Origins of a catalog write, recorded in item_history / relation_history (P4-3a).
const (
	OriginDirect   = "direct"   // saisie dans l'interface
	OriginImport   = "import"   // POST /import
	OriginExternal = "external" // sources externes (ref = connecteur)
	OriginProposal = "proposal" // proposition acceptée (ref = id)
	OriginRevert   = "revert"   // retour à une version (ref = version)
)

// Actor is who writes in the catalog and why; the history triggers read it
// from transaction settings (begin).
type Actor struct {
	UserID uuid.UUID
	Origin string
	Ref    string
}

type actorKey struct{}

// WithActor attaches the writer of the catalog to ctx.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// WithOrigin keeps the user of ctx and changes the origin of the writes.
func WithOrigin(ctx context.Context, origin, ref string) context.Context {
	a := ActorFrom(ctx)
	a.Origin, a.Ref = origin, ref
	return WithActor(ctx, a)
}

// ActorFrom returns the actor of ctx (zero value: `system` for the triggers).
func ActorFrom(ctx context.Context) Actor {
	a, _ := ctx.Value(actorKey{}).(Actor)
	return a
}

// begin opens a write transaction of the catalog with the actor of ctx in its
// local settings (colec.user_id, colec.origin, colec.origin_ref).
func (s *PgStore) begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	a := ActorFrom(ctx)
	user := ""
	if a.UserID != uuid.Nil {
		user = a.UserID.String()
	}
	if _, err := tx.Exec(ctx,
		`SELECT set_config('colec.user_id', $1, true), set_config('colec.origin', $2, true),
		        set_config('colec.origin_ref', $3, true)`, user, a.Origin, a.Ref); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}
