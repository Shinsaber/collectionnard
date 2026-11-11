-- P7-3 — Prêts (docs/phases/exemplaires.md).
--
-- Un prêt porte une partie (ou la totalité) d'un lot d'une collection dont le type est marqué `x-lendable`.
-- L'emprunteur est un ami (borrower_id) ou un simple nom ; borrower_name garde toujours le nom affiché au moment du
-- prêt, figé si le compte de l'emprunteur disparaît. La somme des prêts en cours d'un lot ne dépasse pas sa quantité
-- (contrôlée dans la transaction). Un prêt rendu (returned_at) reste comme historique. reminded_at : rappel de retard
-- déjà envoyé pour l'échéance en cours (remis à zéro quand elle change).
CREATE TABLE loans (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lot_id       UUID NOT NULL REFERENCES collection_items(id) ON DELETE CASCADE,
    quantity     INTEGER NOT NULL DEFAULT 1 CHECK (quantity BETWEEN 1 AND 100000),
    borrower_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    borrower_name TEXT NOT NULL CHECK (length(borrower_name) BETWEEN 1 AND 100),
    lent_at      DATE NOT NULL DEFAULT current_date,
    due_at       DATE,
    returned_at  DATE,
    note         TEXT NOT NULL DEFAULT '' CHECK (length(note) <= 500),
    reminded_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (due_at IS NULL OR due_at >= lent_at),
    CHECK (returned_at IS NULL OR returned_at >= lent_at)
);
CREATE INDEX loans_lot ON loans (lot_id);
CREATE INDEX loans_borrower ON loans (borrower_id) WHERE borrower_id IS NOT NULL;
CREATE INDEX loans_overdue ON loans (due_at) WHERE returned_at IS NULL AND due_at IS NOT NULL;

---- create above / drop below ----

DROP TABLE IF EXISTS loans;
