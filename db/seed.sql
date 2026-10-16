-- =============================================================================
-- Collectionnard — Jeu de données de test MVP : Séries TV
-- =============================================================================
-- Prérequis : migrations appliquées (colec-server migrate). Appliqué automatiquement par
-- `colec-server migrate` si SEED_ON_EMPTY=true et la table users est vide (pg-start, compose).
-- Scénario : Breaking Bad avec multi-chemin (Broadcast + DVD)
--
-- Convention : clés JSON, noms de RelationTypes / ItemTypes / CollectionTypes et valeurs d'enum
-- en anglais snake_case ; les libellés affichés viennent de x-labels (fr / en), les noms affichés
-- des types de x-title / x-description et ceux des valeurs d'enum de x-enum-labels ; le libellé d'un
-- item vient du gabarit x-label de son ItemType.
-- =============================================================================

-- =============================================================================
-- 1. USERS
-- =============================================================================

-- Mots de passe de dev : admin / admin, shinsaber / shinsaber (hash bcrypt : `colec-server hash-password <mdp>`)
INSERT INTO users (id, username, email, password, role) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'admin',     'admin@colec.local',     '$2a$10$59V0nqZ3Sb.G99iWddEB4ecQY2TRU20UZzgWcM/ZNsggk.UEI5lGK', 'admin'),
    ('a0000000-0000-0000-0000-000000000002', 'shinsaber', 'shinsaber@colec.local', '$2a$10$5d6MyNdOvZD/Ac31Dmi6TOR6YWPydNX3m/n2tYouPMeEOPTf2ndrG',  'user');

-- =============================================================================
-- 2. TYPES
-- =============================================================================
-- Les types (ItemTypes, RelationTypes, CollectionTypes) ne sont plus dans ce fichier : ce sont les
-- paquets de domaine « collections » et « tv » (server/internal/domains/packages/*.json), installés
-- avant ce seed (colec-server migrate avec SEED_ON_EMPTY=true, colec-server seed).

-- =============================================================================
-- 5. ITEMS — Breaking Bad
-- =============================================================================

-- Série
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001',
 '{"title": "Breaking Bad", "synopsis": {"fr": "Un professeur de chimie se tourne vers la fabrication de méthamphétamine.", "en": "A chemistry teacher turns to methamphetamine manufacturing."}, "start_year": 2008, "end_year": 2013, "images": {"poster": [], "screenshot": [], "banner": [], "default_image": ""}}');

-- Diffusion AMC
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002',
 '{"name": "AMC 2008-2013", "type": "cable", "channel": "AMC"}');

-- Saison 1
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000003',
 '{"name": "Saison 1", "number": 1, "year": 2008}');

-- Saison 2
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000003',
 '{"name": "Saison 2", "number": 2, "year": 2009}');

-- Épisodes Saison 1
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000004',
 '{"number": 1, "title": "Pilot", "duration": 58, "synopsis": {"fr": "Walter White reçoit un diagnostic de cancer.", "en": "Walter White receives a cancer diagnosis."}}'),
('e0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000004',
 '{"number": 2, "title": "Cat''s in the Bag...", "duration": 48}'),
('e0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000004',
 '{"number": 3, "title": "...And the Bag''s in the River", "duration": 48}');

-- DVD Intégrale
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000020', 'b0000000-0000-0000-0000-000000000005',
 '{"title": "Breaking Bad - Coffret Intégrale", "publisher": "Sony Pictures", "release_year": 2014, "disc_count": 16}');

-- =============================================================================
-- 6. ITEM RELATIONS — Le graphe multi-chemin
-- =============================================================================

-- Chemin 1 : Série → Diffusion → Saison → Épisode
INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES

-- Breaking Bad → broadcast_on → AMC
('c0000000-0000-0000-0000-000000000001',
 'e0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000002',
 '{"broadcast_start_year": 2008}'),

-- AMC → contains_season → Saison 1
('c0000000-0000-0000-0000-000000000002',
 'e0000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000003',
 '{}'),

-- AMC → contains_season → Saison 2
('c0000000-0000-0000-0000-000000000002',
 'e0000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000004',
 '{}'),

-- Saison 1 → contains_episode → Ep1, Ep2, Ep3
('c0000000-0000-0000-0000-000000000003',
 'e0000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000010',
 '{"order": 1}'),
('c0000000-0000-0000-0000-000000000003',
 'e0000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000011',
 '{"order": 2}'),
('c0000000-0000-0000-0000-000000000003',
 'e0000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000012',
 '{"order": 3}');

-- Chemin 2 : Série → DVD → Épisode (mêmes épisodes !)
INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES

-- Breaking Bad → released_as → DVD Intégrale
('c0000000-0000-0000-0000-000000000004',
 'e0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000020',
 '{}'),

-- DVD Intégrale → includes_episode → Ep1, Ep2, Ep3
('c0000000-0000-0000-0000-000000000005',
 'e0000000-0000-0000-0000-000000000020', 'e0000000-0000-0000-0000-000000000010',
 '{"disc": 1, "track": 1}'),
('c0000000-0000-0000-0000-000000000005',
 'e0000000-0000-0000-0000-000000000020', 'e0000000-0000-0000-0000-000000000011',
 '{"disc": 1, "track": 2}'),
('c0000000-0000-0000-0000-000000000005',
 'e0000000-0000-0000-0000-000000000020', 'e0000000-0000-0000-0000-000000000012',
 '{"disc": 1, "track": 3}');

-- =============================================================================
-- 7. COLLECTIONS et COLLECTION ITEMS
-- =============================================================================

-- Collection "Mes DVDs" (type Ownership) pour shinsaber
INSERT INTO collections (id, user_id, collection_type_id, name, description) VALUES
('f0000000-0000-0000-0000-000000000001',
 'a0000000-0000-0000-0000-000000000002',
 'd0000000-0000-0000-0000-000000000001',
 'Mes DVDs', 'Ma collection de DVDs');

-- Ajout du coffret DVD Intégrale dans "Mes DVDs"
INSERT INTO collection_items (collection_id, item_id, attributes) VALUES
('f0000000-0000-0000-0000-000000000001',
 'e0000000-0000-0000-0000-000000000020',
 '{"condition": "new", "purchase_price": 45.00, "acquisition_date": "2020-12-25"}');

-- Collection "Séries vues" (type Watching) pour shinsaber
INSERT INTO collections (id, user_id, collection_type_id, name, description) VALUES
('f0000000-0000-0000-0000-000000000002',
 'a0000000-0000-0000-0000-000000000002',
 'd0000000-0000-0000-0000-000000000002',
 'Séries vues', 'Les séries que j''ai regardées');

-- Breaking Bad dans "Séries vues"
INSERT INTO collection_items (collection_id, item_id, attributes) VALUES
('f0000000-0000-0000-0000-000000000002',
 'e0000000-0000-0000-0000-000000000001',
 '{"watched_at": "2023-03-15", "rating": 10, "progress": 100}');

-- =============================================================================
-- 8. TAGS
-- =============================================================================

INSERT INTO tags (id, name) VALUES
('aa000000-0000-0000-0000-000000000001', 'drama'),
('aa000000-0000-0000-0000-000000000002', 'thriller'),
('aa000000-0000-0000-0000-000000000003', 'crime');

INSERT INTO item_tags (item_id, tag_id) VALUES
('e0000000-0000-0000-0000-000000000001', 'aa000000-0000-0000-0000-000000000001'),
('e0000000-0000-0000-0000-000000000001', 'aa000000-0000-0000-0000-000000000002'),
('e0000000-0000-0000-0000-000000000001', 'aa000000-0000-0000-0000-000000000003');
