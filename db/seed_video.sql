-- =============================================================================
-- Collectionnard — Domaine Vidéo (P3-5, video.md)
-- Chargé après seed.sql (utilisateurs) ; les types viennent des paquets de domaine
-- « video » et « people » (server/internal/domains/packages), installés avant les seeds.
--
-- Généré par le connecteur « films » (Wikidata CC0 + BnF Licence Ouverte, récupérés le
-- 2026-10-17) : Le Fabuleux Destin d'Amélie Poulain, son générique (réalisation, scénario,
-- musique, image, dix interprètes) et son Blu-ray « édition collector » TF1 vidéo 2010.
-- UUID e…0400-0416.
-- =============================================================================

INSERT INTO items (id, item_type_id, attributes) VALUES
-- work : wikidata:Q484048
('e0000000-0000-0000-0000-000000000400', 'b0000000-0000-0000-0000-000000000030', '{"countries":["FR","DE"],"description":{"en":"2001 film directed by Jean-Pierre Jeunet","fr":"film français de 2001"},"duration":122,"external_ids":{"bnf_ark":"ark:/12148/cb14404106q","imdb":"tt0211915","wikidata":"Q484048"},"genres":["romantic_comedy","comedy","drama"],"kind":"feature","original_language":"fr","release_year":2001,"sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}],"title":{"en":"Amélie","fr":"Le Fabuleux Destin d''Amélie Poulain"}}'),
-- person : wikidata:Q217324
('e0000000-0000-0000-0000-000000000401', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French film director","fr":"cinéaste français"},"birth_date":"1953-09-03","external_ids":{"bnf_ark":"ark:/12148/cb126149782","wikidata":"Q217324"},"first_name":"Jean-Pierre","last_name":"Jeunet","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q1915951
('e0000000-0000-0000-0000-000000000402', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French screenwriter (born 1961)","fr":"scénariste français"},"birth_date":"1961-11-22","external_ids":{"bnf_ark":"ark:/12148/cb14113424x","wikidata":"Q1915951"},"first_name":"Guillaume","last_name":"Laurant","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q157256
('e0000000-0000-0000-0000-000000000403', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French composer (born 1970)","fr":"auteur-compositeur et musicien français"},"birth_date":"1970-06-23","external_ids":{"bnf_ark":"ark:/12148/cb14023970h","wikidata":"Q157256"},"first_name":"Yann","last_name":"Tiersen","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q993210
('e0000000-0000-0000-0000-000000000404', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French cinematographer","fr":"directeur de la photographie et réalisateur français"},"external_ids":{"wikidata":"Q993210"},"first_name":"Bruno","last_name":"Delbonnel","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q304123
('e0000000-0000-0000-0000-000000000405', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French actress (born 1976)","fr":"actrice, mannequin et photographe française"},"birth_date":"1976-08-09","external_ids":{"bnf_ark":"ark:/12148/cb14053885w","wikidata":"Q304123"},"first_name":"Audrey","last_name":"Tautou","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q491766
('e0000000-0000-0000-0000-000000000406', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French actor, film director, film producer and screenwriter (born 1967)","fr":"acteur, producteur, réalisateur et scénariste français"},"birth_date":"1967-08-03","external_ids":{"bnf_ark":"ark:/12148/cb12495720r","wikidata":"Q491766"},"first_name":"Mathieu","last_name":"Kassovitz","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q312661
('e0000000-0000-0000-0000-000000000407', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"Moroccan Amazigh -French comedian and actor","fr":"acteur et producteur franco-marocain"},"birth_date":"1975-06-18","external_ids":{"bnf_ark":"ark:/12148/cb14021335v","wikidata":"Q312661"},"first_name":"Jamel","last_name":"Debbouze","nationality":"FR","native_name":"جما","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q38554
('e0000000-0000-0000-0000-000000000408', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French actor","fr":"acteur français"},"birth_date":"1955-03-04","external_ids":{"bnf_ark":"ark:/12148/cb140338217","wikidata":"Q38554"},"first_name":"Dominique","last_name":"Pinon","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q241437
('e0000000-0000-0000-0000-000000000409', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"Belgian actress, film director and screenwriter","fr":"actrice, réalisatrice et scénariste belge"},"birth_date":"1953-02-27","external_ids":{"bnf_ark":"ark:/12148/cb14208567n","wikidata":"Q241437"},"first_name":"Yolande","last_name":"Moreau","nationality":"BE","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q518457
('e0000000-0000-0000-0000-000000000410', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French actor","fr":"acteur français"},"birth_date":"1946-02-17","external_ids":{"bnf_ark":"ark:/12148/cb138935436","wikidata":"Q518457"},"first_name":"André","last_name":"Dussollier","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q3141565
('e0000000-0000-0000-0000-000000000411', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French actress, film director, theatre director and screenwriter","fr":"actrice, réalisatrice, scénariste et chanteuse française"},"birth_date":"1962-01-21","external_ids":{"bnf_ark":"ark:/12148/cb140141187","wikidata":"Q3141565"},"first_name":"Isabelle","last_name":"Nanty","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q715111
('e0000000-0000-0000-0000-000000000412', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French actor and comedian","fr":"acteur et humoriste français"},"birth_date":"1942-12-19","external_ids":{"bnf_ark":"ark:/12148/cb11923113m","wikidata":"Q715111"},"first_name":"Jacques","last_name":"Narcy","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q2121322
('e0000000-0000-0000-0000-000000000413', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French actor (1944-2004)","fr":"acteur français"},"birth_date":"1944-06-24","external_ids":{"bnf_ark":"ark:/12148/cb14014107k","wikidata":"Q2121322"},"first_name":"Joseph","last_name":"Holgado","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- person : wikidata:Q2848938
('e0000000-0000-0000-0000-000000000414', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French actress (1929–2022)","fr":"actrice française"},"birth_date":"1929-09-20","external_ids":{"bnf_ark":"ark:/12148/cb14168787w","wikidata":"Q2848938"},"first_name":"Andrée","last_name":"Damant","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- edition : bnf:cb42284132m
('e0000000-0000-0000-0000-000000000415', 'b0000000-0000-0000-0000-000000000005', '{"audio_languages":["fr"],"disc_count":1,"ean":"3384442196406","edition":"édition collector","external_ids":{"bnf_ark":"ark:/12148/cb42284132m"},"format":"blu_ray","notes":"Contient aussi : commentaire audio du réalisateur, interview du réalisateur (21 min), making of (13 min), bande-annonce, photos, extrait du storyboard, essais des comédiens (6 min), projets d''affiches, etc.","publisher":"TF1 vidéo","release_year":2010,"runtime":120,"sources":[{"retrieved_at":"2026-10-17","source":"bnf"}],"title":"Le fabuleux destin d''Amélie Poulain"}'),
-- publisher : publisher:tf1 vidéo
('e0000000-0000-0000-0000-000000000416', 'b0000000-0000-0000-0000-000000000017', '{"name":"TF1 vidéo","sources":[{"retrieved_at":"2026-10-17","source":"bnf"}]}');

INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES
('c0000000-0000-0000-0000-000000000004', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000415', '{}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000403', '{"roles":["composer"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000402', '{"roles":["screenwriter"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000413', '{"order":9,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000401', '{"roles":["director","screenwriter"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000409', '{"order":5,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000414', '{"order":10,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000405', '{"order":1,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000407', '{"order":3,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000411', '{"order":7,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000408', '{"order":4,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000406', '{"order":2,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000410', '{"order":6,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000412', '{"order":8,"roles":["actor"]}'),
('c0000000-0000-0000-0000-000000000030', 'e0000000-0000-0000-0000-000000000400', 'e0000000-0000-0000-0000-000000000404', '{"roles":["cinematographer"]}'),
('c0000000-0000-0000-0000-000000000031', 'e0000000-0000-0000-0000-000000000415', 'e0000000-0000-0000-0000-000000000416', '{"roles":["publisher"]}');
