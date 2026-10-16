-- Migration 004 : noms d'utilisateur insensibles à la casse (« Admin » et « admin » = un seul compte).
--
-- Le nom reste stocké et affiché tel que saisi ; l'unicité et la connexion portent sur lower(username).
-- Contrôle préalable : si deux comptes ne diffèrent que par la casse, la migration échoue en les
-- nommant (à renommer à la main avant de relancer).
DO $$
DECLARE
    dups text;
BEGIN
    SELECT string_agg(names, '; ') INTO dups
    FROM (SELECT string_agg(username, ', ' ORDER BY username) AS names
          FROM users GROUP BY lower(username) HAVING count(*) > 1) d;
    IF dups IS NOT NULL THEN
        RAISE EXCEPTION 'usernames differing only by case, rename them first: %', dups;
    END IF;
END $$;

ALTER TABLE users DROP CONSTRAINT users_username_key;
CREATE UNIQUE INDEX users_username_lower_key ON users (lower(username));

---- create above / drop below ----

DROP INDEX users_username_lower_key;
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);
