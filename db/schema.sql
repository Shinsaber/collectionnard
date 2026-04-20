-- =============================================================================
-- Collectionnard — Schéma de Base de Données PostgreSQL
-- =============================================================================
-- Ce schéma implémente le modèle conceptuel décrit dans Collectionnard.md
-- Pattern clé : JSONB + JSON Schema pour les attributs dynamiques
-- =============================================================================

-- Extensions requises
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()

-- =============================================================================
-- 1. USERS
-- =============================================================================

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username    VARCHAR(100) NOT NULL UNIQUE,
    email       VARCHAR(255) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,           -- hash bcrypt/argon2
    role        VARCHAR(20)  NOT NULL DEFAULT 'user'
                CHECK (role IN ('admin', 'user')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- =============================================================================
-- 2. META-TYPES (schémas dynamiques)
-- =============================================================================
-- Chaque méta-type stocke un JSON Schema dans `attributes_schema`.
-- Ce JSON Schema décrit et valide les attributs des instances associées.
--
-- Exemple de attributes_schema pour un ItemType "Série" :
-- {
--   "type": "object",
--   "required": ["titre"],
--   "properties": {
--     "titre":      { "type": "string", "minLength": 1 },
--     "synopsis":   { "type": "string" },
--     "annee_debut": { "type": "integer", "minimum": 1900 },
--     "annee_fin":   { "type": "integer", "minimum": 1900 },
--     "poster":     { "type": "string", "format": "uri" }
--   }
-- }
-- =============================================================================

-- 2a. Item Types
CREATE TABLE item_types (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL UNIQUE,
    description         TEXT,
    attributes_schema   JSONB NOT NULL DEFAULT '{}',   -- JSON Schema
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2b. Relation Types
CREATE TABLE relation_types (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL UNIQUE,
    description         TEXT,
    attributes_schema   JSONB NOT NULL DEFAULT '{}',   -- JSON Schema
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Junction tables for multiple source/target item types per relation type
CREATE TABLE relation_type_sources (
    relation_type_id UUID NOT NULL REFERENCES relation_types(id) ON DELETE CASCADE,
    item_type_id     UUID NOT NULL REFERENCES item_types(id),
    PRIMARY KEY (relation_type_id, item_type_id)
);

CREATE TABLE relation_type_targets (
    relation_type_id UUID NOT NULL REFERENCES relation_types(id) ON DELETE CASCADE,
    item_type_id     UUID NOT NULL REFERENCES item_types(id),
    PRIMARY KEY (relation_type_id, item_type_id)
);

-- 2c. Collection Types
CREATE TABLE collection_types (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL UNIQUE,
    description         TEXT,
    attributes_schema   JSONB NOT NULL DEFAULT '{}',   -- JSON Schema
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =============================================================================
-- 3. INSTANCES
-- =============================================================================

-- 3a. Items
CREATE TABLE items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_type_id    UUID NOT NULL REFERENCES item_types(id),
    attributes      JSONB NOT NULL DEFAULT '{}',    -- validé par item_types.attributes_schema
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 3b. Item Relations (graphe)
CREATE TABLE item_relations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    relation_type_id    UUID NOT NULL REFERENCES relation_types(id),
    source_item_id      UUID NOT NULL REFERENCES items(id),
    target_item_id      UUID NOT NULL REFERENCES items(id),
    attributes          JSONB NOT NULL DEFAULT '{}',    -- validé par relation_types.attributes_schema
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Empêcher les doublons exacts (même type, même source, même target)
    UNIQUE (relation_type_id, source_item_id, target_item_id),

    -- Un item ne peut pas être relié à lui-même
    CHECK (source_item_id != target_item_id)
);

-- 3c. Collections
CREATE TABLE collections (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id),
    collection_type_id  UUID NOT NULL REFERENCES collection_types(id),
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Un user ne peut pas avoir deux collections avec le même nom
    UNIQUE (user_id, name)
);

-- 3d. Collection Items (lien collection ↔ item)
CREATE TABLE collection_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_id   UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    item_id         UUID NOT NULL REFERENCES items(id),
    attributes      JSONB NOT NULL DEFAULT '{}',    -- validé par collection_types.attributes_schema
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Un item ne peut être qu'une fois dans une collection
    UNIQUE (collection_id, item_id)
);

-- =============================================================================
-- 4. TAGS
-- =============================================================================

CREATE TABLE tags (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name    VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE item_tags (
    item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, tag_id)
);

-- =============================================================================
-- 5. INDEX
-- =============================================================================

-- Items : recherche par type + attributs
CREATE INDEX idx_items_type       ON items (item_type_id);
CREATE INDEX idx_items_attributes ON items USING GIN (attributes);

-- Relations : traversée de graphe
CREATE INDEX idx_relations_source ON item_relations (source_item_id);
CREATE INDEX idx_relations_target ON item_relations (target_item_id);
CREATE INDEX idx_relations_type   ON item_relations (relation_type_id);
CREATE INDEX idx_relations_attrs  ON item_relations USING GIN (attributes);

-- Collections : recherche par user + type
CREATE INDEX idx_collections_user ON collections (user_id);
CREATE INDEX idx_collections_type ON collections (collection_type_id);

-- Collection items : recherche par item (dans quelles collections est cet item ?)
CREATE INDEX idx_collection_items_item ON collection_items (item_id);
CREATE INDEX idx_collection_items_attrs ON collection_items USING GIN (attributes);

-- Tags
CREATE INDEX idx_item_tags_tag ON item_tags (tag_id);

-- =============================================================================
-- 6. TRIGGERS — updated_at automatique
-- =============================================================================

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_item_types_updated_at
    BEFORE UPDATE ON item_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_items_updated_at
    BEFORE UPDATE ON items FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_collection_types_updated_at
    BEFORE UPDATE ON collection_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_relation_types_updated_at
    BEFORE UPDATE ON relation_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_collections_updated_at
    BEFORE UPDATE ON collections FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_collection_items_updated_at
    BEFORE UPDATE ON collection_items FOR EACH ROW EXECUTE FUNCTION update_updated_at();
