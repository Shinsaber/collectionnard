-- I-2 — Commentaires (docs/phases/interactions.md).
--
-- Un seul fil par contenu : les avis maintenant, les articles en I-3 (`target_kind`). Fil à plat, en
-- Markdown restreint comme les avis. Le fil suit la visibilité de son avis ; une note seule n'en a pas.
-- L'auteur de l'avis peut fermer son fil (`reviews.comments_closed_at`). Un commentaire se signale et se
-- masque comme un avis (I-1).
CREATE TABLE comments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_kind    VARCHAR(20) NOT NULL CHECK (target_kind IN ('review', 'article')),
    target_id      UUID NOT NULL,
    author_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body           TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    hidden_at      TIMESTAMPTZ,
    hidden_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    hidden_reason  TEXT
);
CREATE INDEX comments_target ON comments (target_kind, target_id, created_at);
CREATE TRIGGER trg_comments_updated_at
    BEFORE UPDATE ON comments FOR EACH ROW EXECUTE FUNCTION update_updated_at();

ALTER TABLE reviews ADD COLUMN comments_closed_at TIMESTAMPTZ;

-- Le fil part avec son avis (suppression par l'auteur, ou de l'item en cascade).
CREATE FUNCTION delete_review_comments() RETURNS trigger AS $$
BEGIN
    DELETE FROM comments WHERE target_kind = 'review' AND target_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_reviews_delete_comments
    AFTER DELETE ON reviews FOR EACH ROW EXECUTE FUNCTION delete_review_comments();

---- create above / drop below ----

DROP TRIGGER IF EXISTS trg_reviews_delete_comments ON reviews;
DROP FUNCTION IF EXISTS delete_review_comments();
ALTER TABLE reviews DROP COLUMN IF EXISTS comments_closed_at;
DROP TABLE IF EXISTS comments;
