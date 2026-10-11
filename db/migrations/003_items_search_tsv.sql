-- Migration 003 : recherche plein texte des items (paramètre `search` de GET /items).
--
-- `search_tsv` indexe toutes les chaînes et tous les nombres des attributs, à toute profondeur
-- (objets i18n-text compris, donc toutes les langues), hors URLs et chemins (galeries d'images).
-- Dictionnaire `simple` (pas de racinisation : neutre vis-à-vis de la langue) et `unaccent` :
-- « serie » trouve « Série ». Le serveur construit une requête préfixe (`break:* & bad:*`).
--
-- `unaccent()` n'est que STABLE (il dépend du dictionnaire) : une colonne générée exige une
-- expression IMMUTABLE, d'où l'enveloppe `f_unaccent` qui fige le dictionnaire.
-- CREATE EXTENSION exige un rôle propriétaire de la base (cas de pg-start et du compose).
CREATE EXTENSION IF NOT EXISTS unaccent;

CREATE OR REPLACE FUNCTION f_unaccent(text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
    AS $$ SELECT public.unaccent('public.unaccent'::regdictionary, $1) $$;

ALTER TABLE items ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
    to_tsvector('simple', f_unaccent(jsonb_path_query_array(
        attributes,
        'strict $.** ? ((@.type() == "string" && !(@ like_regex "^(/|https?://)")) || @.type() == "number")'
    )::text))
) STORED;

CREATE INDEX idx_items_search ON items USING GIN (search_tsv);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_items_search;
ALTER TABLE items DROP COLUMN IF EXISTS search_tsv;
DROP FUNCTION IF EXISTS f_unaccent(text);
DROP EXTENSION IF EXISTS unaccent;
