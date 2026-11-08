-- P7-2 — Amis (docs/phases/exemplaires-prets.md).
--
-- Une amitié est réciproque : un membre fait une demande, l'autre l'accepte. Une seule ligne par paire, quel
-- que soit le sens ; refuser, annuler ou rompre supprime la ligne. Bloquer un membre rompt l'amitié et retire
-- les demandes entre eux.
CREATE TABLE friendships (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status        VARCHAR(10) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    accepted_at   TIMESTAMPTZ,
    CHECK (requester_id <> addressee_id),
    CHECK ((status = 'accepted') = (accepted_at IS NOT NULL))
);
CREATE UNIQUE INDEX friendships_pair ON friendships (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));
CREATE INDEX friendships_addressee ON friendships (addressee_id);

-- Nouveau niveau de visibilité entre `private` et `members` : les amis du propriétaire.
ALTER TABLE collections DROP CONSTRAINT collections_visibility_check,
    ADD CONSTRAINT collections_visibility_check CHECK (visibility IN ('private', 'friends', 'members', 'public'));
ALTER TABLE users DROP CONSTRAINT users_profile_visibility_check,
    ADD CONSTRAINT users_profile_visibility_check CHECK (profile_visibility IN ('private', 'friends', 'members', 'public'));
ALTER TABLE reviews DROP CONSTRAINT reviews_visibility_check,
    ADD CONSTRAINT reviews_visibility_check CHECK (visibility IN ('private', 'friends', 'members', 'anonymous', 'public'));

---- create above / drop below ----

UPDATE collections SET visibility = 'private' WHERE visibility = 'friends';
UPDATE users SET profile_visibility = 'private' WHERE profile_visibility = 'friends';
UPDATE reviews SET visibility = 'private' WHERE visibility = 'friends';
ALTER TABLE collections DROP CONSTRAINT collections_visibility_check,
    ADD CONSTRAINT collections_visibility_check CHECK (visibility IN ('private', 'members', 'public'));
ALTER TABLE users DROP CONSTRAINT users_profile_visibility_check,
    ADD CONSTRAINT users_profile_visibility_check CHECK (profile_visibility IN ('private', 'members', 'public'));
ALTER TABLE reviews DROP CONSTRAINT reviews_visibility_check,
    ADD CONSTRAINT reviews_visibility_check CHECK (visibility IN ('private', 'members', 'anonymous', 'public'));
DROP TABLE IF EXISTS friendships;
