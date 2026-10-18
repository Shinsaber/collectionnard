-- P4-3b — Propositions et modération (social.md).
--
-- Une proposition `catalog` porte des items et relations au format de l'import du catalogue
-- (`payload`) et l'instantané des entités existantes qu'elle modifie au moment de sa soumission
-- (`base`, `{"items": {id: attributs}, "relations": {id: attributs}}`) : à l'acceptation, une
-- entité changée depuis est un conflit. `schema_suggestion` et `report` (P4-3c) ne portent qu'un
-- texte (`description`), un signalement visant un item (`item_id`).
CREATE TABLE proposals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind            VARCHAR(20) NOT NULL CHECK (kind IN ('catalog', 'schema_suggestion', 'report')),
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'pending', 'changes_requested', 'accepted', 'rejected', 'withdrawn')),
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    payload         JSONB,
    base            JSONB,
    item_id         UUID REFERENCES items(id) ON DELETE SET NULL,
    reviewer_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    review_comment  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    submitted_at    TIMESTAMPTZ,
    decided_at      TIMESTAMPTZ
);
CREATE INDEX proposals_author_idx ON proposals (author_id, created_at DESC);
CREATE INDEX proposals_status_idx ON proposals (status, submitted_at);

-- Fil de discussion d'une proposition (auteur et modérateurs).
CREATE TABLE proposal_comments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_id  UUID NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    author_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX proposal_comments_proposal_idx ON proposal_comments (proposal_id, created_at);

---- create above / drop below ----

DROP TABLE IF EXISTS proposal_comments;
DROP TABLE IF EXISTS proposals;
