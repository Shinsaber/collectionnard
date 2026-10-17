-- =============================================================================
-- Collectionnard — Domaine Musique (P3-4, musique.md)
-- Chargé après seed.sql (utilisateurs) ; les types viennent du paquet de domaine
-- « music » (server/internal/domains/packages/music.json), installé avant les seeds.
--
-- Généré par le connecteur « music » (MusicBrainz, données de base CC0, récupérées le
-- 2026-10-17) : l'album Discovery de Daft Punk, son édition française en double vinyle
-- (Virgin 2001), son label et ses 14 pistes. UUID e…0300-0317.
-- =============================================================================

INSERT INTO items (id, item_type_id, attributes) VALUES
-- work : mb:release-group:48117b90-a16e-34ca-a514-19c702df1158
('e0000000-0000-0000-0000-000000000300', 'b0000000-0000-0000-0000-000000000021', '{"external_ids":{"musicbrainz":"48117b90-a16e-34ca-a514-19c702df1158","wikidata":"Q1901313"},"first_release_year":2001,"primary_type":"album","sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Discovery"}'),
-- person : mb:artist:056e4f3e-d505-4dad-8ec1-d04f521cbb56
('e0000000-0000-0000-0000-000000000301', 'b0000000-0000-0000-0000-000000000020', '{"begin_year":1993,"country":"FR","disambiguation":"French electronic duo","end_year":2021,"external_ids":{"musicbrainz":"056e4f3e-d505-4dad-8ec1-d04f521cbb56","wikidata":"Q185828"},"kind":"group","name":"Daft Punk","sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}]}'),
-- edition : mb:release:ac7518c6-b630-4761-99c7-6a94cc35a594
('e0000000-0000-0000-0000-000000000302', 'b0000000-0000-0000-0000-000000000022', '{"barcode":"724384960612","catalog_number":"8496061","country":"FR","external_ids":{"musicbrainz":"ac7518c6-b630-4761-99c7-6a94cc35a594"},"format":"vinyl","media_count":2,"release_date":"2001-10-03","release_year":2001,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"status":"official","title":"Discovery"}'),
-- publisher : mb:label:49b58bdb-3d74-40c6-956a-4c4b46115c9c
('e0000000-0000-0000-0000-000000000303', 'b0000000-0000-0000-0000-000000000024', '{"disambiguation":"worldwide imprint of Virgin Records Ltd. and all its subsidiaries","external_ids":{"musicbrainz":"49b58bdb-3d74-40c6-956a-4c4b46115c9c"},"name":"Virgin","sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}]}'),
-- volume : mb:recording:60fa767a-d85d-4991-82bc-4294e0b11ae7
('e0000000-0000-0000-0000-000000000304', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"60fa767a-d85d-4991-82bc-4294e0b11ae7"},"isrc":"GBAHT1305744","length":321,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"One More Time"}'),
-- volume : mb:recording:7c0e11a3-1c7a-4e6d-a14d-6e86d86dbaad
('e0000000-0000-0000-0000-000000000305', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"7c0e11a3-1c7a-4e6d-a14d-6e86d86dbaad"},"isrc":"GATIW0000057","length":208,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Aerodynamic"}'),
-- volume : mb:recording:15f16efc-8762-4151-95ab-c12d06268640
('e0000000-0000-0000-0000-000000000306', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"15f16efc-8762-4151-95ab-c12d06268640"},"isrc":"GBDUW0000058","length":298,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Digital Love"}'),
-- volume : mb:recording:f1a6a40f-78f5-4918-968d-f64363bae94c
('e0000000-0000-0000-0000-000000000307', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"f1a6a40f-78f5-4918-968d-f64363bae94c"},"isrc":"GBDUW0000059","length":224,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Harder, Better, Faster, Stronger"}'),
-- volume : mb:recording:d63ad586-1124-4df1-9a2b-c014d7b7cb02
('e0000000-0000-0000-0000-000000000308', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"d63ad586-1124-4df1-9a2b-c014d7b7cb02"},"isrc":"GBDUW0000060","length":212,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Crescendolls"}'),
-- volume : mb:recording:712edd8d-b413-4e4e-aca4-9b91fce2e65c
('e0000000-0000-0000-0000-000000000309', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"712edd8d-b413-4e4e-aca4-9b91fce2e65c"},"isrc":"GBDUW0000061","length":104,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Nightvision"}'),
-- volume : mb:recording:8a98614f-0533-44ef-890a-639cab407a2d
('e0000000-0000-0000-0000-000000000310', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"8a98614f-0533-44ef-890a-639cab407a2d"},"isrc":"GBDUW0000062","length":238,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Superheroes"}'),
-- volume : mb:recording:07b720fc-0d6f-4342-a6df-876d1749de0b
('e0000000-0000-0000-0000-000000000311', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"07b720fc-0d6f-4342-a6df-876d1749de0b"},"isrc":"GBDUW0000063","length":202,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"High Life"}'),
-- volume : mb:recording:defbfcb2-7a5c-4456-be28-b7e5ccf92cc2
('e0000000-0000-0000-0000-000000000312', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"defbfcb2-7a5c-4456-be28-b7e5ccf92cc2"},"isrc":"GBDUW0000064","length":231,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Something About Us"}'),
-- volume : mb:recording:e1fe1d12-dac2-4325-ae1b-25e0ef06b998
('e0000000-0000-0000-0000-000000000313', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"e1fe1d12-dac2-4325-ae1b-25e0ef06b998"},"isrc":"GBDUW0000065","length":228,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Voyager"}'),
-- volume : mb:recording:2a220770-1150-4190-b0e0-57d89a0a7469
('e0000000-0000-0000-0000-000000000314', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"2a220770-1150-4190-b0e0-57d89a0a7469"},"isrc":"GBDUW0000066","length":345,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Veridis Quo"}'),
-- volume : mb:recording:3f60ae01-43cc-4db9-86ec-97b79f83764f
('e0000000-0000-0000-0000-000000000315', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"3f60ae01-43cc-4db9-86ec-97b79f83764f"},"isrc":"GBDUW0000067","length":207,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Short Circuit"}'),
-- volume : mb:recording:59038571-03ea-4ca4-965c-a57eae2aa138
('e0000000-0000-0000-0000-000000000316', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"59038571-03ea-4ca4-965c-a57eae2aa138"},"isrc":"GBDUW0000068","length":240,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Face to Face"}'),
-- volume : mb:recording:1a915a2d-e2f4-4300-a230-96fc9c8a7b60
('e0000000-0000-0000-0000-000000000317', 'b0000000-0000-0000-0000-000000000023', '{"external_ids":{"musicbrainz":"1a915a2d-e2f4-4300-a230-96fc9c8a7b60"},"isrc":"GBDUW0000069","length":600,"sources":[{"retrieved_at":"2026-10-17","source":"musicbrainz"}],"title":"Too Long"}');

INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES
('c0000000-0000-0000-0000-000000000020', 'e0000000-0000-0000-0000-000000000300', 'e0000000-0000-0000-0000-000000000301', '{}'),
('c0000000-0000-0000-0000-000000000021', 'e0000000-0000-0000-0000-000000000300', 'e0000000-0000-0000-0000-000000000302', '{}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000311', '{"disc":2,"order":8,"position":1}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000306', '{"disc":1,"order":3,"position":3}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000317', '{"disc":2,"order":14,"position":7}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000314', '{"disc":2,"order":11,"position":4}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000315', '{"disc":2,"order":12,"position":5}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000316', '{"disc":2,"order":13,"position":6}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000304', '{"disc":1,"order":1,"position":1}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000309', '{"disc":1,"order":6,"position":6}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000305', '{"disc":1,"order":2,"position":2}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000310', '{"disc":1,"order":7,"position":7}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000308', '{"disc":1,"order":5,"position":5}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000312', '{"disc":2,"order":9,"position":2}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000313', '{"disc":2,"order":10,"position":3}'),
('c0000000-0000-0000-0000-000000000022', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000307', '{"disc":1,"order":4,"position":4}'),
('c0000000-0000-0000-0000-000000000023', 'e0000000-0000-0000-0000-000000000302', 'e0000000-0000-0000-0000-000000000303', '{"catalog_number":"8496061"}');
