-- Migration 008 : comptes, invitations, profils (Phase 4, P4-0 — social.md).
--
-- Profil : nom affiché, présentation, avatar (nom d'un envoi de /uploads). Un compte désactivé ne peut plus se
-- connecter ni rafraîchir son jeton (ses jetons de rafraîchissement sont supprimés à la désactivation).
-- Invitations et liens de réinitialisation : seul le hash SHA-256 du jeton est stocké (comme refresh_tokens) ;
-- le lien n'est montré qu'une fois, à sa création, et sert une seule fois.
ALTER TABLE users
    ADD COLUMN display_name VARCHAR(100),
    ADD COLUMN bio          TEXT,
    ADD COLUMN avatar       VARCHAR(255),
    ADD COLUMN disabled_at  TIMESTAMPTZ;

CREATE TABLE invitations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash  VARCHAR(64)  NOT NULL UNIQUE,
    created_by  UUID         REFERENCES users(id) ON DELETE SET NULL,
    role        VARCHAR(20)  NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    note        VARCHAR(255),
    expires_at  TIMESTAMPTZ  NOT NULL,
    used_at     TIMESTAMPTZ,
    used_by     UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_invitations_created_by ON invitations (created_by);

CREATE TABLE password_resets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64)  NOT NULL UNIQUE,
    created_by  UUID         REFERENCES users(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ  NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_password_resets_user ON password_resets (user_id);

---- create above / drop below ----

DROP TABLE password_resets;
DROP TABLE invitations;
ALTER TABLE users
    DROP COLUMN disabled_at,
    DROP COLUMN avatar,
    DROP COLUMN bio,
    DROP COLUMN display_name;
