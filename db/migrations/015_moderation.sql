-- I-1 — Signalements, modération et réactions (docs/phases/interactions.md).
--
-- Rôle `moderator` : traite les signalements avec les admins, sans écrire dans le catalogue (le rôle reste
-- unique par compte). Un avis signalé peut être masqué (`hidden_*`) : il sort des listes et des moyennes,
-- son auteur le voit encore avec le motif. Le journal (`moderation_log`) garde toute action, en ajout seul
-- (le code n'y modifie ni n'y supprime rien), avec une copie du contenu au moment de l'action. Les réactions
-- donnent une intention ou une émotion ; elles n'envoient jamais un avis en modération.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'contributor', 'moderator', 'user'));
ALTER TABLE invitations DROP CONSTRAINT invitations_role_check;
ALTER TABLE invitations ADD CONSTRAINT invitations_role_check CHECK (role IN ('admin', 'contributor', 'moderator', 'user'));

ALTER TABLE reviews ADD COLUMN hidden_at TIMESTAMPTZ;
ALTER TABLE reviews ADD COLUMN hidden_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE reviews ADD COLUMN hidden_reason TEXT;

CREATE TABLE reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Commentaires (I-2) et articles (I-3) viendront s'ajouter aux avis.
    target_kind  VARCHAR(20) NOT NULL CHECK (target_kind IN ('review', 'comment', 'article')),
    target_id    UUID NOT NULL,
    reporter_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason       VARCHAR(20) NOT NULL
                 CHECK (reason IN ('abusive', 'harassment', 'spoiler', 'spam', 'off_topic', 'other')),
    details      TEXT NOT NULL DEFAULT '',
    status       VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'actioned', 'dismissed')),
    handled_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    handled_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Un seul signalement ouvert par (signaleur, cible).
CREATE UNIQUE INDEX reports_open_once ON reports (reporter_id, target_kind, target_id) WHERE status = 'open';
CREATE INDEX reports_open ON reports (target_kind, target_id) WHERE status = 'open';
CREATE INDEX reports_reporter ON reports (reporter_id, status, handled_at);

CREATE TABLE moderation_log (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action            VARCHAR(20) NOT NULL CHECK (action IN ('hide', 'restore', 'dismiss')),
    target_kind       VARCHAR(20) NOT NULL,
    target_id         UUID NOT NULL,
    target_author_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    moderator_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    reason            TEXT NOT NULL DEFAULT '',
    -- Copie du contenu au moment de l'action : l'entrée reste lisible si l'auteur supprime son avis.
    excerpt           TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX moderation_log_author ON moderation_log (target_author_id, created_at DESC);
CREATE INDEX moderation_log_recent ON moderation_log (created_at DESC);

CREATE TABLE review_reactions (
    review_id   UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        VARCHAR(20) NOT NULL CHECK (kind IN ('interesting', 'tempting', 'funny', 'disapprove')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (review_id, user_id, kind)
);
CREATE INDEX review_reactions_kind ON review_reactions (review_id, kind);

---- create above / drop below ----

DROP TABLE IF EXISTS review_reactions;
DROP TABLE IF EXISTS moderation_log;
DROP TABLE IF EXISTS reports;
ALTER TABLE reviews DROP COLUMN IF EXISTS hidden_reason;
ALTER TABLE reviews DROP COLUMN IF EXISTS hidden_by;
ALTER TABLE reviews DROP COLUMN IF EXISTS hidden_at;
UPDATE users SET role = 'user' WHERE role = 'moderator';
DELETE FROM invitations WHERE role = 'moderator';
ALTER TABLE invitations DROP CONSTRAINT invitations_role_check;
ALTER TABLE invitations ADD CONSTRAINT invitations_role_check CHECK (role IN ('admin', 'contributor', 'user'));
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'contributor', 'user'));
