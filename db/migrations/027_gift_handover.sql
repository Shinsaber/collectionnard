-- P7-4b — Remise d'un cadeau (docs/phases/exemplaires-prets.md).
--
-- `given` change de sens : le cadeau est **remis**, la surprise est levée. Le propriétaire est prévenu et voit la
-- remise sur son lot (given_at), puis confirme (`received`, received_at : les exemplaires ont quitté sa liste, la
-- réservation n'est plus que l'historique de l'offrant, lot_id NULL) ou la refuse (réservation supprimée).
-- Seul `reserved` reste secret, et limité dans le temps.
ALTER TABLE gift_reservations DROP CONSTRAINT gift_reservations_status_check,
    ADD CONSTRAINT gift_reservations_status_check CHECK (status IN ('reserved', 'given', 'received')),
    ADD COLUMN given_at TIMESTAMPTZ,
    ADD COLUMN received_at TIMESTAMPTZ;
UPDATE gift_reservations SET given_at = updated_at WHERE status = 'given';

-- Un lot supprimé (retiré de la liste, collection supprimée, réception confirmée) vaut réception de ses cadeaux
-- remis : sans lot, le propriétaire n'aurait plus rien à confirmer et la remise resterait en attente.
CREATE FUNCTION gift_lot_deleted() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE gift_reservations SET status = 'received', received_at = now(), updated_at = now()
    WHERE lot_id = OLD.id AND status = 'given';
    RETURN OLD;
END $$;
CREATE TRIGGER gift_lot_deleted BEFORE DELETE ON collection_items
    FOR EACH ROW EXECUTE FUNCTION gift_lot_deleted();

---- create above / drop below ----

DROP TRIGGER IF EXISTS gift_lot_deleted ON collection_items;
DROP FUNCTION IF EXISTS gift_lot_deleted();

UPDATE gift_reservations SET status = 'given' WHERE status = 'received';
ALTER TABLE gift_reservations DROP COLUMN IF EXISTS received_at, DROP COLUMN IF EXISTS given_at,
    DROP CONSTRAINT gift_reservations_status_check,
    ADD CONSTRAINT gift_reservations_status_check CHECK (status IN ('reserved', 'given'));
