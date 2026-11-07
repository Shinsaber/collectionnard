-- I-3 — Articles sur la fiche d'un item (docs/phases/interactions.md).
--
-- Relayer une nouvelle d'une œuvre (sortie, annonce, critique) : un titre, un lien http(s), un mot. Ouverts sur
-- tous les types d'item, mais l'article d'un membre attend d'être validé (`status` pending → published, ou
-- rejected avec un motif), comme une proposition de fiche ; les admins et les modérateurs publient directement.
-- Un article publié se signale et se masque comme un avis, et porte un fil de commentaires (`comments`).
CREATE TABLE item_articles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id             UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    author_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title               VARCHAR(200) NOT NULL,
    url                 VARCHAR(2000) NOT NULL CHECK (url ~* '^https?://'),
    message             TEXT NOT NULL DEFAULT '',
    lang                VARCHAR(10) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'published', 'rejected')),
    reviewed_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at         TIMESTAMPTZ,
    reject_reason       TEXT,
    hidden_at           TIMESTAMPTZ,
    hidden_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    hidden_reason       TEXT,
    comments_closed_at  TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX item_articles_item ON item_articles (item_id, created_at DESC);
CREATE INDEX item_articles_pending ON item_articles (created_at) WHERE status = 'pending';
CREATE TRIGGER trg_item_articles_updated_at
    BEFORE UPDATE ON item_articles FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Le fil part avec son article.
CREATE FUNCTION delete_article_comments() RETURNS trigger AS $$
BEGIN
    DELETE FROM comments WHERE target_kind = 'article' AND target_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_item_articles_delete_comments
    AFTER DELETE ON item_articles FOR EACH ROW EXECUTE FUNCTION delete_article_comments();

---- create above / drop below ----

DROP TRIGGER IF EXISTS trg_item_articles_delete_comments ON item_articles;
DROP FUNCTION IF EXISTS delete_article_comments();
DROP TABLE IF EXISTS item_articles;
