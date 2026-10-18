-- P4-3a — Historique du catalogue et rôle contributeur (social.md).
--
-- Rôle `contributor` : écrit directement dans le catalogue (sans supprimer) et modère les propositions.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'contributor', 'user'));
ALTER TABLE invitations DROP CONSTRAINT IF EXISTS invitations_role_check;
ALTER TABLE invitations ADD CONSTRAINT invitations_role_check CHECK (role IN ('admin', 'contributor', 'user'));

-- Historique : une ligne par écriture d'un item ou d'une relation, écrite par un déclencheur
-- (aucune écriture ne peut l'oublier). Instantanés complets des attributs avant / après
-- (NULL : création / suppression). Pas de clé étrangère vers l'entité : l'historique d'une
-- entité supprimée est gardé. L'auteur et l'origine viennent des réglages de transaction
-- `colec.user_id`, `colec.origin`, `colec.origin_ref` (store.begin) ; sans eux : `system`
-- (seeds, interventions en SQL).
CREATE TABLE item_history (
    id            BIGSERIAL PRIMARY KEY,
    item_id       UUID NOT NULL,
    version       INTEGER NOT NULL,
    item_type_id  UUID NOT NULL,
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    origin        VARCHAR(20) NOT NULL,
    origin_ref    TEXT,
    before        JSONB,
    after         JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (item_id, version)
);
CREATE INDEX item_history_user_idx ON item_history (user_id, created_at);

CREATE TABLE relation_history (
    id                BIGSERIAL PRIMARY KEY,
    relation_id       UUID NOT NULL,
    version           INTEGER NOT NULL,
    relation_type_id  UUID NOT NULL,
    source_item_id    UUID NOT NULL,
    target_item_id    UUID NOT NULL,
    user_id           UUID REFERENCES users(id) ON DELETE SET NULL,
    origin            VARCHAR(20) NOT NULL,
    origin_ref        TEXT,
    before            JSONB,
    after             JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (relation_id, version)
);
CREATE INDEX relation_history_source_idx ON relation_history (source_item_id, created_at);
CREATE INDEX relation_history_target_idx ON relation_history (target_item_id, created_at);

CREATE FUNCTION colec_actor_user() RETURNS UUID LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('colec.user_id', true), '')::uuid
$$;
CREATE FUNCTION colec_actor_origin() RETURNS TEXT LANGUAGE sql STABLE AS $$
    SELECT COALESCE(NULLIF(current_setting('colec.origin', true), ''), 'system')
$$;
CREATE FUNCTION colec_actor_ref() RETURNS TEXT LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('colec.origin_ref', true), '')
$$;

CREATE FUNCTION colec_item_history() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    rec items%ROWTYPE;
BEGIN
    IF TG_OP = 'DELETE' THEN rec := OLD; ELSE rec := NEW; END IF;
    IF TG_OP = 'UPDATE' AND OLD.attributes IS NOT DISTINCT FROM NEW.attributes THEN
        RETURN NULL;
    END IF;
    INSERT INTO item_history (item_id, version, item_type_id, user_id, origin, origin_ref, before, after)
    VALUES (
        rec.id,
        COALESCE((SELECT max(version) FROM item_history WHERE item_id = rec.id), 0) + 1,
        rec.item_type_id, colec_actor_user(), colec_actor_origin(), colec_actor_ref(),
        CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE OLD.attributes END,
        CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE NEW.attributes END
    );
    RETURN NULL;
END
$$;
CREATE TRIGGER items_history AFTER INSERT OR UPDATE OR DELETE ON items
    FOR EACH ROW EXECUTE FUNCTION colec_item_history();

CREATE FUNCTION colec_relation_history() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    rec item_relations%ROWTYPE;
BEGIN
    IF TG_OP = 'DELETE' THEN rec := OLD; ELSE rec := NEW; END IF;
    IF TG_OP = 'UPDATE' AND OLD.attributes IS NOT DISTINCT FROM NEW.attributes
       AND OLD.source_item_id = NEW.source_item_id AND OLD.target_item_id = NEW.target_item_id THEN
        RETURN NULL;
    END IF;
    INSERT INTO relation_history (relation_id, version, relation_type_id, source_item_id, target_item_id,
                                  user_id, origin, origin_ref, before, after)
    VALUES (
        rec.id,
        COALESCE((SELECT max(version) FROM relation_history WHERE relation_id = rec.id), 0) + 1,
        rec.relation_type_id, rec.source_item_id, rec.target_item_id,
        colec_actor_user(), colec_actor_origin(), colec_actor_ref(),
        CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE OLD.attributes END,
        CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE NEW.attributes END
    );
    RETURN NULL;
END
$$;
CREATE TRIGGER item_relations_history AFTER INSERT OR UPDATE OR DELETE ON item_relations
    FOR EACH ROW EXECUTE FUNCTION colec_relation_history();

---- create above / drop below ----

DROP TRIGGER IF EXISTS item_relations_history ON item_relations;
DROP TRIGGER IF EXISTS items_history ON items;
DROP FUNCTION IF EXISTS colec_relation_history();
DROP FUNCTION IF EXISTS colec_item_history();
DROP FUNCTION IF EXISTS colec_actor_ref();
DROP FUNCTION IF EXISTS colec_actor_origin();
DROP FUNCTION IF EXISTS colec_actor_user();
DROP TABLE IF EXISTS relation_history;
DROP TABLE IF EXISTS item_history;
UPDATE invitations SET role = 'user' WHERE role = 'contributor';
UPDATE users SET role = 'user' WHERE role = 'contributor';
ALTER TABLE invitations DROP CONSTRAINT IF EXISTS invitations_role_check;
ALTER TABLE invitations ADD CONSTRAINT invitations_role_check CHECK (role IN ('admin', 'user'));
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'user'));
