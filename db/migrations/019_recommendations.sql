-- I-4 — Recommandations et blocage (docs/phases/interactions.md).
--
-- Un membre recommande un item à un autre, avec un mot facultatif ; le destinataire est notifié. Une seule
-- recommandation par (expéditeur, destinataire, item) tant qu'elle n'est pas écartée. Un compte peut refuser
-- les recommandations (`users.accepts_recommendations`) et bloquer un membre : le bloqué ne peut plus lui
-- recommander d'item, ni commenter ou réagir à ses contenus signés ; ses contenus sont masqués au bloqueur.
CREATE TABLE recommendations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id       UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    message       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at       TIMESTAMPTZ,
    UNIQUE (sender_id, recipient_id, item_id),
    CHECK (sender_id <> recipient_id)
);
CREATE INDEX recommendations_recipient ON recommendations (recipient_id, created_at DESC);

ALTER TABLE users ADD COLUMN accepts_recommendations BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE user_blocks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- anonymous : bloqué depuis un contenu qui ne le nommait pas (avis anonyme, profil privé) ; son nom n'est
    -- alors jamais montré dans la liste des blocages.
    anonymous   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);
CREATE INDEX user_blocks_blocked ON user_blocks (blocked_id);

---- create above / drop below ----

DROP TABLE IF EXISTS user_blocks;
ALTER TABLE users DROP COLUMN IF EXISTS accepts_recommendations;
DROP TABLE IF EXISTS recommendations;
