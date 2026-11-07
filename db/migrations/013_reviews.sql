-- I-0 — Avis et critères (docs/phases/interactions.md).
--
-- `review_types` : quatrième méta-type, son JSON Schema liste les critères notés de 0 à 10. Un type
-- d'item sans `review_type_id` n'ouvre pas les avis (Person, par exemple).
--
-- `reviews` : un avis par utilisateur et par item, sur l'œuvre et non sur un exemplaire. Note globale
-- (`criteria` NULL) ou note par critère (`rating` = moyenne calculée par le serveur).
CREATE TABLE review_types (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name               VARCHAR(100) NOT NULL UNIQUE,
    description        TEXT,
    attributes_schema  JSONB NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_review_types_updated_at
    BEFORE UPDATE ON review_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();

ALTER TABLE item_types ADD COLUMN review_type_id UUID REFERENCES review_types(id) ON DELETE RESTRICT;
CREATE INDEX idx_item_types_review_type ON item_types (review_type_id);

CREATE TABLE reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id         UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    -- Type dont viennent les critères ; NULL en note globale.
    review_type_id  UUID REFERENCES review_types(id) ON DELETE RESTRICT,
    criteria        JSONB,
    rating          NUMERIC(3,1) NOT NULL CHECK (rating >= 0 AND rating <= 10),
    body            TEXT NOT NULL DEFAULT '',
    spoiler         BOOLEAN NOT NULL DEFAULT false,
    lang            VARCHAR(10) NOT NULL,
    -- private : note seule pour tous ; members : signé pour les membres, note seule pour les visiteurs ;
    -- anonymous : texte sans auteur ; public : signé pour tous. Toute note compte dans la moyenne.
    visibility      VARCHAR(10) NOT NULL
                    CHECK (visibility IN ('private', 'members', 'anonymous', 'public')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, item_id),
    CHECK ((criteria IS NULL) = (review_type_id IS NULL))
);
CREATE INDEX idx_reviews_item ON reviews (item_id, created_at DESC);
CREATE INDEX idx_reviews_review_type ON reviews (review_type_id);
CREATE TRIGGER trg_reviews_updated_at
    BEFORE UPDATE ON reviews FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Les paquets de domaine portent aussi des types d'avis.
ALTER TABLE meta_type_origins DROP CONSTRAINT meta_type_origins_kind_check;
ALTER TABLE meta_type_origins ADD CONSTRAINT meta_type_origins_kind_check
    CHECK (kind IN ('item_types', 'relation_types', 'collection_types', 'review_types'));

-- La note quitte le type de collection « Watching », sans reprise des valeurs (instance en bêta). La
-- mise à jour d'un paquet ne retire rien, et le schéma interdit les propriétés inconnues : une valeur
-- laissée en place ferait échouer la prochaine écriture de l'exemplaire.
UPDATE collection_items SET attributes = attributes - 'rating'
WHERE attributes ? 'rating' AND collection_id IN (
    SELECT id FROM collections WHERE collection_type_id = 'd0000000-0000-0000-0000-000000000002');

UPDATE collection_types SET attributes_schema = jsonb_set(
    attributes_schema #- '{properties,rating}' #- '{x-labels,fr,rating}' #- '{x-labels,en,rating}',
    '{x-order}', COALESCE(attributes_schema -> 'x-order', '[]'::jsonb) - 'rating')
WHERE id = 'd0000000-0000-0000-0000-000000000002' AND attributes_schema -> 'properties' ? 'rating';

UPDATE meta_type_origins SET base = jsonb_set(
    base #- '{attributes_schema,properties,rating}'
         #- '{attributes_schema,x-labels,fr,rating}' #- '{attributes_schema,x-labels,en,rating}',
    '{attributes_schema,x-order}', COALESCE(base #> '{attributes_schema,x-order}', '[]'::jsonb) - 'rating')
WHERE meta_type_id = 'd0000000-0000-0000-0000-000000000002' AND base #> '{attributes_schema,properties}' ? 'rating';

---- create above / drop below ----

ALTER TABLE meta_type_origins DROP CONSTRAINT meta_type_origins_kind_check;
DELETE FROM meta_type_origins WHERE kind = 'review_types';
ALTER TABLE meta_type_origins ADD CONSTRAINT meta_type_origins_kind_check
    CHECK (kind IN ('item_types', 'relation_types', 'collection_types'));
DROP TABLE IF EXISTS reviews;
ALTER TABLE item_types DROP COLUMN IF EXISTS review_type_id;
DROP TABLE IF EXISTS review_types;
