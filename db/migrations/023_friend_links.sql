-- P7-2 — Liens d'ami (docs/phases/exemplaires.md).
--
-- L'invitation par nom d'utilisateur est réservée aux profils visibles de l'inviteur (sinon les noms se
-- devineraient). Pour un profil privé, le membre crée un lien secret, à usage unique et limité dans le temps,
-- qu'il transmet lui-même ; celui qui l'ouvre devient son ami en l'acceptant. Seul le hash SHA-256 du jeton est
-- stocké, comme pour les invitations de comptes. Un lien utilisé est supprimé.
CREATE TABLE friend_links (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX friend_links_user ON friend_links (user_id);

---- create above / drop below ----

DROP TABLE IF EXISTS friend_links;
