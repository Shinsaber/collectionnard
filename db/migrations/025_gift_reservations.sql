-- P7-4 — Cadeaux réservés (docs/phases/exemplaires.md).
--
-- Un ami du propriétaire d'une liste de souhaits (type marqué `x-reservable`) réserve une part d'un lot pour
-- l'offrir. Les réservations ne sont JAMAIS renvoyées au propriétaire : aucune requête de ses listes, exports,
-- statistiques ou copie hors ligne ne lit cette table. Quand le lot quitte la liste (lot supprimé), la réservation
-- reste (lot_id NULL) pour que l'offrant en soit prévenu (notified_at) et décide. Annuler une réservation la supprime.
CREATE TABLE gift_reservations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lot_id       UUID REFERENCES collection_items(id) ON DELETE SET NULL,
    item_id      UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    owner_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    giver_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quantity     INTEGER NOT NULL DEFAULT 1 CHECK (quantity BETWEEN 1 AND 100000),
    status       VARCHAR(10) NOT NULL DEFAULT 'reserved' CHECK (status IN ('reserved', 'given')),
    note         TEXT NOT NULL DEFAULT '' CHECK (length(note) <= 500),
    notified_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (owner_id <> giver_id)
);
CREATE INDEX gift_reservations_lot ON gift_reservations (lot_id) WHERE lot_id IS NOT NULL;
CREATE INDEX gift_reservations_giver ON gift_reservations (giver_id);
CREATE INDEX gift_reservations_left ON gift_reservations (giver_id) WHERE lot_id IS NULL AND notified_at IS NULL;

---- create above / drop below ----

DROP TABLE IF EXISTS gift_reservations;
