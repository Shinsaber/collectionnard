-- P7-1 — Exemplaires multiples (docs/phases/exemplaires.md).
--
-- Une ligne de collection devient un **lot** : un ou plusieurs exemplaires identiques d'un item, avec une
-- quantité. Un même item peut former plusieurs lots d'une collection (un état ou une finition différents) :
-- l'unicité (collection_id, item_id) est levée. Les lignes existantes deviennent des lots de 1, exactement le
-- comportement précédent.
ALTER TABLE collection_items DROP CONSTRAINT collection_items_collection_id_item_id_key;
ALTER TABLE collection_items
    ADD COLUMN quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity BETWEEN 1 AND 100000);
-- Remplace l'index porté par la contrainte levée (lecture d'une collection, recherche d'un item dans elle).
CREATE INDEX idx_collection_items_collection ON collection_items (collection_id, item_id);

---- create above / drop below ----

-- Échoue tant qu'un item forme plusieurs lots d'une même collection : les fusionner d'abord.
DROP INDEX IF EXISTS idx_collection_items_collection;
ALTER TABLE collection_items DROP COLUMN IF EXISTS quantity;
ALTER TABLE collection_items ADD CONSTRAINT collection_items_collection_id_item_id_key UNIQUE (collection_id, item_id);
