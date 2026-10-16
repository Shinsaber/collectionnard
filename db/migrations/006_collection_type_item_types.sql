-- Migration 006 : types d'item permis dans les collections d'un type de collection.
--
-- Aucune ligne pour un type de collection = tous les types d'item sont permis (comportement
-- antérieur). Le contrôle porte sur les ajouts (POST /collections/{id}/items, import de
-- collection) : les items déjà présents et devenus non permis sont conservés.
-- Pas de cascade côté item_types (comme relation_type_sources) : supprimer un type d'item encore
-- permis renvoie 409, sans quoi une liste vidée permettrait silencieusement tous les types.
CREATE TABLE collection_type_item_types (
    collection_type_id  UUID NOT NULL REFERENCES collection_types(id) ON DELETE CASCADE,
    item_type_id        UUID NOT NULL REFERENCES item_types(id),
    PRIMARY KEY (collection_type_id, item_type_id)
);
CREATE INDEX idx_collection_type_item_types_item_type ON collection_type_item_types (item_type_id);

---- create above / drop below ----

DROP TABLE collection_type_item_types;
