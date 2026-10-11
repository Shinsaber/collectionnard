-- =============================================================================
-- Collectionnard — Requêtes de test
-- =============================================================================
-- Ces requêtes valident que le schéma supporte les cas d'usage du MVP.
-- Exécuter après les migrations (colec-server migrate) et seed.sql
-- =============================================================================


-- =============================================================================
-- Q1. Lister tous les items d'un type donné avec leurs attributs
-- Cas : tous les épisodes
-- =============================================================================
SELECT
    i.id,
    it.name AS type,
    i.attributes->>'titre' AS titre,
    (i.attributes->>'numero')::int AS numero,
    (i.attributes->>'duree')::int AS duree_min
FROM items i
JOIN item_types it ON it.id = i.item_type_id
WHERE it.name = 'Episode'
ORDER BY (i.attributes->>'numero')::int;


-- =============================================================================
-- Q2. Traversée de graphe : Série → Diffusion → Saison → Épisodes
-- Cas : tous les épisodes de Breaking Bad via la diffusion AMC
-- =============================================================================
SELECT
    serie.attributes->>'titre'  AS serie,
    diff.attributes->>'nom'     AS diffusion,
    (saison.attributes->>'numero')::int AS saison_num,
    (ep.attributes->>'numero')::int     AS episode_num,
    ep.attributes->>'titre'     AS episode_titre
FROM items serie
-- Série → Diffusion
JOIN item_relations r1 ON r1.source_item_id = serie.id
JOIN relation_types rt1 ON rt1.id = r1.relation_type_id AND rt1.name = 'diffuse_en'
JOIN items diff ON diff.id = r1.target_item_id
-- Diffusion → Saison
JOIN item_relations r2 ON r2.source_item_id = diff.id
JOIN relation_types rt2 ON rt2.id = r2.relation_type_id AND rt2.name = 'contient_saison'
JOIN items saison ON saison.id = r2.target_item_id
-- Saison → Épisode
JOIN item_relations r3 ON r3.source_item_id = saison.id
JOIN relation_types rt3 ON rt3.id = r3.relation_type_id AND rt3.name = 'contient_episode'
JOIN items ep ON ep.id = r3.target_item_id
WHERE serie.attributes->>'titre' = 'Breaking Bad'
ORDER BY saison_num, episode_num;


-- =============================================================================
-- Q3. Traversée de graphe : Série → DVD → Épisodes
-- Cas : tous les épisodes du coffret DVD Breaking Bad
-- =============================================================================
SELECT
    serie.attributes->>'titre'    AS serie,
    dvd.attributes->>'titre'      AS dvd,
    (r_ep.attributes->>'disque')::int AS disque,
    (r_ep.attributes->>'piste')::int  AS piste,
    ep.attributes->>'titre'       AS episode_titre
FROM items serie
-- Série → DVD
JOIN item_relations r1 ON r1.source_item_id = serie.id
JOIN relation_types rt1 ON rt1.id = r1.relation_type_id AND rt1.name = 'edite_en'
JOIN items dvd ON dvd.id = r1.target_item_id
-- DVD → Épisode
JOIN item_relations r_ep ON r_ep.source_item_id = dvd.id
JOIN relation_types rt2 ON rt2.id = r_ep.relation_type_id AND rt2.name = 'inclut_episode'
JOIN items ep ON ep.id = r_ep.target_item_id
WHERE serie.attributes->>'titre' = 'Breaking Bad'
ORDER BY disque, piste;


-- =============================================================================
-- Q4. Multi-chemin : trouver TOUS les chemins vers un épisode
-- Cas : par quels chemins accède-t-on à l'épisode "Pilot" ?
-- =============================================================================
SELECT
    'Diffusion' AS chemin,
    diff.attributes->>'nom' AS via,
    (saison.attributes->>'numero')::int AS saison,
    (r3.attributes->>'ordre')::int AS ordre
FROM items ep
JOIN item_relations r3 ON r3.target_item_id = ep.id
JOIN relation_types rt3 ON rt3.id = r3.relation_type_id AND rt3.name = 'contient_episode'
JOIN items saison ON saison.id = r3.source_item_id
JOIN item_relations r2 ON r2.target_item_id = saison.id
JOIN relation_types rt2 ON rt2.id = r2.relation_type_id AND rt2.name = 'contient_saison'
JOIN items diff ON diff.id = r2.source_item_id
WHERE ep.attributes->>'titre' = 'Pilot'

UNION ALL

SELECT
    'DVD' AS chemin,
    dvd.attributes->>'titre' AS via,
    NULL AS saison,
    (r_ep.attributes->>'piste')::int AS ordre
FROM items ep
JOIN item_relations r_ep ON r_ep.target_item_id = ep.id
JOIN relation_types rt_ep ON rt_ep.id = r_ep.relation_type_id AND rt_ep.name = 'inclut_episode'
JOIN items dvd ON dvd.id = r_ep.source_item_id
WHERE ep.attributes->>'titre' = 'Pilot';


-- =============================================================================
-- Q5. Collections d'un utilisateur avec le contenu
-- Cas : toutes les collections de shinsaber
-- =============================================================================
SELECT
    c.name AS collection,
    ct.name AS type_collection,
    it.name AS type_item,
    i.attributes->>'titre' AS item_titre,
    ci.attributes AS details_perso
FROM collections c
JOIN collection_types ct ON ct.id = c.collection_type_id
JOIN users u ON u.id = c.user_id
JOIN collection_items ci ON ci.collection_id = c.id
JOIN items i ON i.id = ci.item_id
JOIN item_types it ON it.id = i.item_type_id
WHERE u.username = 'shinsaber'
ORDER BY c.name;


-- =============================================================================
-- Q6. Recherche par attribut JSONB
-- Cas : tous les items dont le titre contient "Break"
-- =============================================================================
SELECT
    it.name AS type,
    i.attributes->>'titre' AS titre
FROM items i
JOIN item_types it ON it.id = i.item_type_id
WHERE i.attributes->>'titre' ILIKE '%Break%';


-- =============================================================================
-- Q7. Filtrer par tag
-- Cas : tous les items taggés "drame"
-- =============================================================================
SELECT
    it.name AS type,
    i.attributes->>'titre' AS titre,
    array_agg(t.name) AS tags
FROM items i
JOIN item_types it ON it.id = i.item_type_id
JOIN item_tags itg ON itg.item_id = i.id
JOIN tags t ON t.id = itg.tag_id
WHERE i.id IN (
    SELECT item_id FROM item_tags
    JOIN tags ON tags.id = item_tags.tag_id
    WHERE tags.name = 'drame'
)
GROUP BY it.name, i.attributes->>'titre';


-- =============================================================================
-- Q8. Relations sortantes d'un item (voisins dans le graphe)
-- Cas : à quoi est relié "Breaking Bad" ?
-- =============================================================================
SELECT
    rt.name AS relation,
    it_target.name AS type_cible,
    target.attributes->>'titre' AS cible_titre,
    COALESCE(target.attributes->>'nom', target.attributes->>'titre') AS cible_label,
    ir.attributes AS relation_attrs
FROM items source
JOIN item_relations ir ON ir.source_item_id = source.id
JOIN relation_types rt ON rt.id = ir.relation_type_id
JOIN items target ON target.id = ir.target_item_id
JOIN item_types it_target ON it_target.id = target.item_type_id
WHERE source.attributes->>'titre' = 'Breaking Bad'
ORDER BY rt.name;


-- =============================================================================
-- Q9. Récupérer le JSON Schema d'un ItemType (pour le frontend / validation)
-- =============================================================================
SELECT
    name,
    attributes_schema
FROM item_types
WHERE name = 'Serie';


-- =============================================================================
-- Q10. Lister les RelationTypes possibles depuis un ItemType donné
-- Cas : quelles relations peut-on créer depuis une "Serie" ?
-- =============================================================================
SELECT
    rt.name AS relation,
    rt.description,
    it_target.name AS type_cible,
    rt.attributes_schema
FROM relation_types rt
JOIN relation_type_sources rts ON rts.relation_type_id = rt.id
JOIN item_types it_source ON it_source.id = rts.item_type_id
JOIN relation_type_targets rtt ON rtt.relation_type_id = rt.id
JOIN item_types it_target ON it_target.id = rtt.item_type_id
WHERE it_source.name = 'Serie';


-- =============================================================================
-- Q11. Statistiques : nombre d'items par type
-- =============================================================================
SELECT
    it.name AS type,
    COUNT(*) AS nb_items
FROM items i
JOIN item_types it ON it.id = i.item_type_id
GROUP BY it.name
ORDER BY nb_items DESC;


-- =============================================================================
-- Q12. Traversée récursive (CTE) : tous les descendants d'un item
-- Cas : tout le sous-graphe à partir de "Breaking Bad"
-- =============================================================================
WITH RECURSIVE graph AS (
    -- Point de départ
    SELECT
        i.id,
        it.name AS type,
        COALESCE(i.attributes->>'titre', i.attributes->>'nom') AS label,
        0 AS depth,
        ARRAY[i.id] AS path
    FROM items i
    JOIN item_types it ON it.id = i.item_type_id
    WHERE i.attributes->>'titre' = 'Breaking Bad'
      AND it.name = 'Serie'

    UNION ALL

    -- Récursion
    SELECT
        target.id,
        it2.name AS type,
        COALESCE(target.attributes->>'titre', target.attributes->>'nom',
                 'S' || (target.attributes->>'numero')::text) AS label,
        g.depth + 1,
        g.path || target.id
    FROM graph g
    JOIN item_relations ir ON ir.source_item_id = g.id
    JOIN items target ON target.id = ir.target_item_id
    JOIN item_types it2 ON it2.id = target.item_type_id
    WHERE NOT target.id = ANY(g.path)  -- éviter les cycles
)
SELECT
    depth,
    type,
    label,
    repeat('  ', depth) || label AS tree
FROM graph
ORDER BY path;
