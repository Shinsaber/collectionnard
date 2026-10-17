-- =============================================================================
-- Collectionnard — Domaine Jeux vidéo (P3-7, jeux-video.md)
-- Chargé après seed.sql (utilisateurs) ; les types viennent des paquets de domaine
-- « games » et « people » (server/internal/domains/packages), installés avant les seeds.
--
-- Généré par le connecteur « games » (Wikidata CC0 + BnF Licence Ouverte, récupérés le
-- 2026-10-17) : Mario Kart 8 Deluxe, la Nintendo Switch rattachée à Nintendo, ses
-- sociétés, la cartouche Switch (notice complète) et une notice brève avec EAN.
-- UUID e…0500-0507.
-- =============================================================================

INSERT INTO items (id, item_type_id, attributes) VALUES
-- work : wikidata:Q28321447
('e0000000-0000-0000-0000-000000000500', 'b0000000-0000-0000-0000-000000000040', '{"description":{"en":"2017 re-release of the Mario Kart 8 video game","fr":"réédition améliorée de Mario Kart 8 sur Nintendo Switch de 2017"},"external_ids":{"wikidata":"Q28321447"},"game_modes":["single_player","multiplayer","pvp"],"original_title":"マリオカート8 デラックス","pegi":"3","release_year":2017,"sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}],"title":{"en":"Mario Kart 8 Deluxe","fr":"Mario Kart 8 Deluxe"}}'),
-- publisher : wikidata:Q20972825
('e0000000-0000-0000-0000-000000000501', 'b0000000-0000-0000-0000-000000000017', '{"country":"JP","description":{"en":"Nintendo''s largest internal video game studio","fr":"studio japonais de développement de jeux vidéo"},"external_ids":{"wikidata":"Q20972825"},"name":"Nintendo Entertainment Planning & Development","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- publisher : wikidata:Q8093
('e0000000-0000-0000-0000-000000000502', 'b0000000-0000-0000-0000-000000000017', '{"country":"JP","description":{"en":"Japanese multinational video game and consumer electronics company","fr":"entreprise multinationale japonaise de jeu vidéo"},"external_ids":{"wikidata":"Q8093"},"name":"Nintendo","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}],"website":"https://www.nintendo.com/jp/index.html"}'),
-- universe : wikidata:Q19610114
('e0000000-0000-0000-0000-000000000503', 'b0000000-0000-0000-0000-000000000042', '{"external_ids":{"wikidata":"Q19610114"},"name":"Nintendo Switch","release_year":2017,"sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- publisher : wikidata:Q21190892
('e0000000-0000-0000-0000-000000000504', 'b0000000-0000-0000-0000-000000000017', '{"country":"JP","description":{"en":"division of Nintendo","fr":"branche interne à la société Nintendo"},"external_ids":{"wikidata":"Q21190892"},"name":"Nintendo Platform Technology Development","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- edition : bnf:cb453824912
('e0000000-0000-0000-0000-000000000505', 'b0000000-0000-0000-0000-000000000041', '{"external_ids":{"bnf_ark":"ark:/12148/cb453824912"},"format":"cartridge","languages":["en","de","fr","nl","rus","es","pt","it","ja"],"notes":"Avec : 1 f.","release_year":2017,"sources":[{"retrieved_at":"2026-10-17","source":"bnf"}],"title":"Mario kart 8 : deluxe"}'),
-- edition : bnf:cb488158895
('e0000000-0000-0000-0000-000000000506', 'b0000000-0000-0000-0000-000000000041', '{"ean":"045496420246","external_ids":{"bnf_ark":"ark:/12148/cb488158895"},"format":"cartridge","languages":["fr"],"sources":[{"retrieved_at":"2026-10-17","source":"bnf"}],"title":"Mario Kart 8 Deluxe"}'),
-- publisher : publisher:nintendo france
('e0000000-0000-0000-0000-000000000507', 'b0000000-0000-0000-0000-000000000017', '{"name":"NINTENDO FRANCE","sources":[{"retrieved_at":"2026-10-17","source":"bnf"}]}');

INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES
('c0000000-0000-0000-0000-000000000040', 'e0000000-0000-0000-0000-000000000500', 'e0000000-0000-0000-0000-000000000505', '{}'),
('c0000000-0000-0000-0000-000000000040', 'e0000000-0000-0000-0000-000000000500', 'e0000000-0000-0000-0000-000000000506', '{}'),
('c0000000-0000-0000-0000-000000000041', 'e0000000-0000-0000-0000-000000000505', 'e0000000-0000-0000-0000-000000000503', '{}'),
('c0000000-0000-0000-0000-000000000041', 'e0000000-0000-0000-0000-000000000506', 'e0000000-0000-0000-0000-000000000503', '{}'),
('c0000000-0000-0000-0000-000000000041', 'e0000000-0000-0000-0000-000000000500', 'e0000000-0000-0000-0000-000000000503', '{}'),
('c0000000-0000-0000-0000-000000000042', 'e0000000-0000-0000-0000-000000000505', 'e0000000-0000-0000-0000-000000000502', '{"roles":["developer"]}'),
('c0000000-0000-0000-0000-000000000042', 'e0000000-0000-0000-0000-000000000506', 'e0000000-0000-0000-0000-000000000507', '{"roles":["publisher"]}'),
('c0000000-0000-0000-0000-000000000042', 'e0000000-0000-0000-0000-000000000503', 'e0000000-0000-0000-0000-000000000504', '{"roles":["manufacturer"]}'),
('c0000000-0000-0000-0000-000000000042', 'e0000000-0000-0000-0000-000000000503', 'e0000000-0000-0000-0000-000000000502', '{"roles":["manufacturer"]}'),
('c0000000-0000-0000-0000-000000000042', 'e0000000-0000-0000-0000-000000000500', 'e0000000-0000-0000-0000-000000000501', '{"roles":["developer"]}'),
('c0000000-0000-0000-0000-000000000042', 'e0000000-0000-0000-0000-000000000500', 'e0000000-0000-0000-0000-000000000502', '{"roles":["publisher"]}');
