-- I-0 — Édition utilisée pour donner un avis (docs/phases/interactions.md, décision du 2026-11-07).
--
-- Un avis sur une œuvre peut préciser l'édition jouée, vue, lue ou écoutée (`context_item_id`) : une cible de
-- l'œuvre par une relation que son type d'avis déclare (`review_type_context_relations` : Game → GameRelease par
-- game_released_as…), avec le libellé de ce choix (`context_label` : « Joué sur »). L'édition n'est jamais montrée
-- sur un avis non signé pour le lecteur : une édition rare pourrait désigner son propriétaire.
ALTER TABLE review_types ADD COLUMN context_label JSONB;

CREATE TABLE review_type_context_relations (
    review_type_id    UUID NOT NULL REFERENCES review_types(id) ON DELETE CASCADE,
    relation_type_id  UUID NOT NULL REFERENCES relation_types(id) ON DELETE CASCADE,
    PRIMARY KEY (review_type_id, relation_type_id)
);

ALTER TABLE reviews ADD COLUMN context_item_id UUID REFERENCES items(id) ON DELETE SET NULL;
CREATE INDEX idx_reviews_context ON reviews (context_item_id) WHERE context_item_id IS NOT NULL;

---- create above / drop below ----

DROP INDEX IF EXISTS idx_reviews_context;
ALTER TABLE reviews DROP COLUMN IF EXISTS context_item_id;
DROP TABLE IF EXISTS review_type_context_relations;
ALTER TABLE review_types DROP COLUMN IF EXISTS context_label;
