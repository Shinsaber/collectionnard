-- P4-2 — Visibilité et partage (social.md).
--
-- Une collection et un profil sont privés (propriétaire seul), visibles des membres
-- (comptes connectés) ou publics (tout le monde). Privé par défaut : rien ne s'ouvre
-- sans un choix explicite du propriétaire.
ALTER TABLE collections
    ADD COLUMN visibility VARCHAR(10) NOT NULL DEFAULT 'private'
        CHECK (visibility IN ('private', 'members', 'public')),
    -- Champs du CollectionType cachés à tout autre lecteur que le propriétaire.
    -- NULL : les propriétés `x-private` du schéma ; sinon, la liste complète (remplace le défaut).
    ADD COLUMN hidden_fields TEXT[];

ALTER TABLE users
    ADD COLUMN profile_visibility VARCHAR(10) NOT NULL DEFAULT 'private'
        CHECK (profile_visibility IN ('private', 'members', 'public'));

-- Liens de partage secrets d'une collection : lecture seule, valables quelle que soit sa
-- visibilité, jusqu'à leur révocation. Seul le hash SHA-256 du jeton est stocké.
CREATE TABLE share_links (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    token_hash    VARCHAR(64) NOT NULL UNIQUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at    TIMESTAMPTZ
);
CREATE INDEX share_links_collection_idx ON share_links (collection_id);

CREATE INDEX collections_visibility_idx ON collections (user_id, visibility);

---- create above / drop below ----

DROP INDEX IF EXISTS collections_visibility_idx;
DROP TABLE IF EXISTS share_links;
ALTER TABLE users DROP COLUMN IF EXISTS profile_visibility;
ALTER TABLE collections DROP COLUMN IF EXISTS hidden_fields, DROP COLUMN IF EXISTS visibility;
