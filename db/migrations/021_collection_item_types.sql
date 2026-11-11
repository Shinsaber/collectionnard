-- P7-1b — Types d'item acceptés et tri par défaut d'une collection (docs/phases/exemplaires.md).
--
-- Le propriétaire resserre les types d'item que sa collection accepte, dans la limite de son type de collection
-- (`collection_type_item_types`) : aucune ligne = tout ce que permet le type. Une collection qui n'accepte qu'un type
-- d'item est « spécialisée » et peut se trier par les attributs de ce type. `default_sort` : ordre des lots à
-- l'ouverture (vide = ajout le plus récent), valeurs du paramètre `sort` de GET /collections/{id}/items.
CREATE TABLE collection_item_types (
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    item_type_id  UUID NOT NULL REFERENCES item_types(id) ON DELETE CASCADE,
    PRIMARY KEY (collection_id, item_type_id)
);

ALTER TABLE collections ADD COLUMN default_sort TEXT NOT NULL DEFAULT '';

---- create above / drop below ----

ALTER TABLE collections DROP COLUMN IF EXISTS default_sort;
DROP TABLE IF EXISTS collection_item_types;
