-- I-2 / I-3 — Traces dans un fil de commentaires (docs/phases/interactions.md).
--
-- Un fil est à plat : retirer un message sans trace change le sens des réponses. Un commentaire supprimé par son
-- auteur garde sa place (`deleted_at`, texte effacé) sauf s'il était le dernier du fil ; un commentaire masqué par
-- la modération apparaît aux autres comme une ligne vide. `edited_at` marque un commentaire modifié après coup.
ALTER TABLE comments ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE comments ADD COLUMN edited_at TIMESTAMPTZ;

---- create above / drop below ----

ALTER TABLE comments DROP COLUMN IF EXISTS edited_at;
ALTER TABLE comments DROP COLUMN IF EXISTS deleted_at;
