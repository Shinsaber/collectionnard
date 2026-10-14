-- Migration 002 : item_relations.updated_at (les relations sont modifiables via PUT /relations/{id},
-- comme les autres instances, qui ont toutes created_at + updated_at).
ALTER TABLE item_relations
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE item_relations SET updated_at = created_at;

CREATE TRIGGER trg_item_relations_updated_at
    BEFORE UPDATE ON item_relations FOR EACH ROW EXECUTE FUNCTION update_updated_at();

---- create above / drop below ----

DROP TRIGGER IF EXISTS trg_item_relations_updated_at ON item_relations;
ALTER TABLE item_relations DROP COLUMN IF EXISTS updated_at;
