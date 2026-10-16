-- Migration 007 : origine des méta-types installés depuis un paquet de domaine (Phase 3, P3-0).
--
-- Pour chaque type installé : son domaine, la version du paquet et la copie de référence (le type tel
-- que le paquet l'a appliqué). À la mise à jour suivante, la comparaison des trois versions (référence,
-- locale, nouvelle) garde les modifications faites sur l'instance et n'apporte que les ajouts du paquet.
-- Pas de clé étrangère (le type peut être un ItemType, un RelationType ou un CollectionType) : une
-- origine sans type signale un type supprimé localement, que la mise à jour ne recrée pas.
CREATE TABLE meta_type_origins (
    kind             VARCHAR(20) NOT NULL CHECK (kind IN ('item_types', 'relation_types', 'collection_types')),
    meta_type_id     UUID        NOT NULL,
    domain           VARCHAR(50) NOT NULL,
    package_version  INTEGER     NOT NULL,
    base             JSONB       NOT NULL,
    installed_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, meta_type_id)
);
CREATE INDEX idx_meta_type_origins_domain ON meta_type_origins (domain);

---- create above / drop below ----

DROP TABLE meta_type_origins;
