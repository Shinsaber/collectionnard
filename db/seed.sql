-- =============================================================================
-- Collectionnard — Jeu de données de test MVP : Séries TV
-- =============================================================================
-- Prérequis : migrations appliquées (colec-server migrate). Appliqué automatiquement par
-- `colec-server migrate` si SEED_ON_EMPTY=true et la table users est vide (pg-start, compose).
-- Scénario : Breaking Bad avec multi-chemin (Diffusion + DVD)
-- =============================================================================

-- =============================================================================
-- 1. USERS
-- =============================================================================

INSERT INTO users (id, username, email, password, role) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'admin',     'admin@colec.local',     '$2a$10$59V0nqZ3Sb.G99iWddEB4ecQY2TRU20UZzgWcM/ZNsggk.UEI5lGK', 'admin'),
    ('a0000000-0000-0000-0000-000000000002', 'shinsaber', 'shinsaber@colec.local', '$2a$10$YV9A.lO4gHKbcAlw2mT1W.PkJ6E4oWpqKvM.cvlbHvRX7LyCUDcgC',  'user');

-- =============================================================================
-- 2. ITEM TYPES avec JSON Schema
-- =============================================================================

INSERT INTO item_types (id, name, description, attributes_schema) VALUES

-- Série
('b0000000-0000-0000-0000-000000000001', 'Serie', 'Série télévisée', '{
    "type": "object",
    "required": ["titre"],
    "x-labels": {
        "fr": { "titre": "Titre", "synopsis": "Synopsis", "annee_debut": "Année de début", "annee_fin": "Année de fin", "images": "Images" },
        "en": { "titre": "Title", "synopsis": "Synopsis", "annee_debut": "Start Year", "annee_fin": "End Year", "images": "Images" }
    },
    "properties": {
        "titre":       { "type": "string",  "minLength": 1 },
        "synopsis":    { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "additionalProperties": false },
        "annee_debut": { "type": "integer", "minimum": 1900 },
        "annee_fin":   { "type": "integer", "minimum": 1900 },
        "images":      { "type": "object", "x-widget": "image-gallery", "x-image-categories": {"poster": {"ratio": "2/3"}, "screenshot": {"ratio": "16/9"}, "banner": {"ratio": "3/1"}}, "properties": { "default_image": { "type": "string" }, "poster": { "type": "array", "items": { "type": "string" } }, "screenshot": { "type": "array", "items": { "type": "string" } }, "banner": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false }
    },
    "additionalProperties": false
}'),

-- Diffusion
('b0000000-0000-0000-0000-000000000002', 'Diffusion', 'Mode de diffusion (TV, streaming…)', '{
    "type": "object",
    "required": ["nom"],
    "x-labels": {
        "fr": { "nom": "Nom", "type": "Type", "chaine": "Support" },
        "en": { "nom": "Name", "type": "Type", "chaine": "Support" }
    },
    "properties": {
        "nom":    { "type": "string", "minLength": 1 },
        "type":   { "type": "string", "enum": ["TV", "streaming", "cable", "web"] },
        "chaine": { "type": "string" }
    },
    "additionalProperties": false
}'),

-- Saison
('b0000000-0000-0000-0000-000000000003', 'Saison', 'Saison d''une série', '{
    "type": "object",
    "required": ["numero"],
    "x-labels": {
        "fr": { "nom": "Nom", "numero": "Numéro", "annee": "Année" },
        "en": { "nom": "Name", "numero": "Number", "annee": "Year" }
    },
    "properties": {
        "nom":    { "type": "string", "minLength": 1 },
        "numero": { "type": "integer", "minimum": 1 },
        "annee":  { "type": "integer", "minimum": 1900 }
    },
    "additionalProperties": false
}'),

-- Épisode
('b0000000-0000-0000-0000-000000000004', 'Episode', 'Épisode d''une série', '{
    "type": "object",
    "required": ["numero", "titre"],
    "x-labels": {
        "fr": { "numero": "Numéro", "titre": "Titre", "duree": "Durée (min)", "synopsis": "Synopsis" },
        "en": { "numero": "Number", "titre": "Title", "duree": "Duration (min)", "synopsis": "Synopsis" }
    },
    "properties": {
        "numero":   { "type": "integer", "minimum": 1 },
        "titre":    { "type": "string",  "minLength": 1 },
        "duree":    { "type": "integer", "minimum": 1, "description": "Durée en minutes" },
        "synopsis": { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "additionalProperties": false }
    },
    "additionalProperties": false
}'),

-- DVD
('b0000000-0000-0000-0000-000000000005', 'DVD', 'Édition DVD/Blu-ray', '{
    "type": "object",
    "required": ["titre"],
    "x-labels": {
        "fr": { "titre": "Titre", "editeur": "Éditeur", "annee_sortie": "Année de sortie", "nb_disques": "Nombre de disques" },
        "en": { "titre": "Title", "editeur": "Publisher", "annee_sortie": "Release Year", "nb_disques": "Number of Discs" }
    },
    "properties": {
        "titre":        { "type": "string",  "minLength": 1 },
        "editeur":      { "type": "string" },
        "annee_sortie": { "type": "integer", "minimum": 1900 },
        "nb_disques":   { "type": "integer", "minimum": 1 }
    },
    "additionalProperties": false
}');

-- =============================================================================
-- 3. RELATION TYPES avec JSON Schema
-- =============================================================================

INSERT INTO relation_types (id, name, description, attributes_schema) VALUES

-- Série → Diffusion
('c0000000-0000-0000-0000-000000000001', 'diffuse_en', 'Série diffusée via un mode de diffusion', '{
    "type": "object",
    "x-labels": {
        "fr": { "annee_debut_diff": "Année de début de diffusion" },
        "en": { "annee_debut_diff": "Broadcast Start Year" }
    },
    "properties": {
        "annee_debut_diff": { "type": "integer", "minimum": 1900 }
    },
    "additionalProperties": false
}'),

-- Diffusion → Saison
('c0000000-0000-0000-0000-000000000002', 'contient_saison', 'Diffusion contient une saison', '{
    "type": "object",
    "properties": {},
    "additionalProperties": false
}'),

-- Saison → Épisode
('c0000000-0000-0000-0000-000000000003', 'contient_episode', 'Saison contient un épisode', '{
    "type": "object",
    "x-labels": {
        "fr": { "ordre": "Ordre" },
        "en": { "ordre": "Order" }
    },
    "properties": {
        "ordre": { "type": "integer", "minimum": 1 }
    },
    "additionalProperties": false
}'),

-- Série → DVD
('c0000000-0000-0000-0000-000000000004', 'edite_en', 'Série éditée en DVD/Blu-ray', '{
    "type": "object",
    "properties": {},
    "additionalProperties": false
}'),

-- DVD → Épisode
('c0000000-0000-0000-0000-000000000005', 'inclut_episode', 'DVD inclut un épisode', '{
    "type": "object",
    "x-labels": {
        "fr": { "disque": "Disque", "piste": "Piste" },
        "en": { "disque": "Disc", "piste": "Track" }
    },
    "properties": {
        "disque": { "type": "integer", "minimum": 1 },
        "piste":  { "type": "integer", "minimum": 1 }
    },
    "additionalProperties": false
}');

-- Junction: source item types
INSERT INTO relation_type_sources (relation_type_id, item_type_id) VALUES
('c0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001'),
('c0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002'),
('c0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000003'),
('c0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000001'),
('c0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000005');

-- Junction: target item types
INSERT INTO relation_type_targets (relation_type_id, item_type_id) VALUES
('c0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000002'),
('c0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000003'),
('c0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000004'),
('c0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000005'),
('c0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000004');

-- =============================================================================
-- 4. COLLECTION TYPES avec JSON Schema
-- =============================================================================

INSERT INTO collection_types (id, name, description, attributes_schema) VALUES

('d0000000-0000-0000-0000-000000000001', 'Possession', 'Items possédés physiquement', '{
    "type": "object",
    "x-labels": {
        "fr": { "etat_physique": "État physique", "prix_achat": "Prix d\u0027achat", "date_acquisition": "Date d\u0027acquisition" },
        "en": { "etat_physique": "Physical Condition", "prix_achat": "Purchase Price", "date_acquisition": "Acquisition Date" }
    },
    "properties": {
        "etat_physique":    { "type": "string", "enum": ["neuf", "très bon", "bon", "acceptable", "mauvais"] },
        "prix_achat":       { "type": "number", "minimum": 0 },
        "date_acquisition": { "type": "string", "format": "date" }
    },
    "additionalProperties": false
}'),

('d0000000-0000-0000-0000-000000000002', 'Visionnage', 'Séries/films vus', '{
    "type": "object",
    "x-labels": {
        "fr": { "date_vue": "Date de visionnage", "note": "Note", "progression": "Progression" },
        "en": { "date_vue": "Watch Date", "note": "Rating", "progression": "Progress" }
    },
    "properties": {
        "date_vue":    { "type": "string", "format": "date" },
        "note":        { "type": "integer", "minimum": 0, "maximum": 10 },
        "progression": { "type": "integer", "minimum": 0, "maximum": 100, "description": "Pourcentage" }
    },
    "additionalProperties": false
}'),

('d0000000-0000-0000-0000-000000000003', 'Wishlist', 'Items souhaités', '{
    "type": "object",
    "x-labels": {
        "fr": { "priorite": "Priorité", "budget_max": "Budget maximum" },
        "en": { "priorite": "Priority", "budget_max": "Maximum Budget" }
    },
    "properties": {
        "priorite":   { "type": "integer", "minimum": 1, "maximum": 5 },
        "budget_max": { "type": "number",  "minimum": 0 }
    },
    "additionalProperties": false
}');

-- =============================================================================
-- 5. ITEMS — Breaking Bad
-- =============================================================================

-- Série
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001',
 '{"titre": "Breaking Bad", "synopsis": {"fr": "Un professeur de chimie se tourne vers la fabrication de méthamphétamine.", "en": "A chemistry teacher turns to methamphetamine manufacturing."}, "annee_debut": 2008, "annee_fin": 2013, "images": {"poster": [], "screenshot": [], "banner": [], "default_image": ""}}');

-- Diffusion AMC
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002',
 '{"nom": "AMC 2008-2013", "type": "cable", "chaine": "AMC"}');

-- Saison 1
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000003',
 '{"nom": "Saison 1", "numero": 1, "annee": 2008}');

-- Saison 2
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000003',
 '{"nom": "Saison 2", "numero": 2, "annee": 2009}');

-- Épisodes Saison 1
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000004',
 '{"numero": 1, "titre": "Pilot", "duree": 58, "synopsis": {"fr": "Walter White reçoit un diagnostic de cancer.", "en": "Walter White receives a cancer diagnosis."}}'),
('e0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000004',
 '{"numero": 2, "titre": "Cat''s in the Bag...", "duree": 48}'),
('e0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000004',
 '{"numero": 3, "titre": "...And the Bag''s in the River", "duree": 48}');

-- DVD Intégrale
INSERT INTO items (id, item_type_id, attributes) VALUES
('e0000000-0000-0000-0000-000000000020', 'b0000000-0000-0000-0000-000000000005',
 '{"titre": "Breaking Bad - Coffret Intégrale", "editeur": "Sony Pictures", "annee_sortie": 2014, "nb_disques": 16}');

-- =============================================================================
-- 6. ITEM RELATIONS — Le graphe multi-chemin
-- =============================================================================

-- Chemin 1 : Série → Diffusion → Saison → Épisode
INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES

-- Breaking Bad → diffusé_en → AMC
('c0000000-0000-0000-0000-000000000001',
 'e0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000002',
 '{"annee_debut_diff": 2008}'),

-- AMC → contient_saison → Saison 1
('c0000000-0000-0000-0000-000000000002',
 'e0000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000003',
 '{}'),

-- AMC → contient_saison → Saison 2
('c0000000-0000-0000-0000-000000000002',
 'e0000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000004',
 '{}'),

-- Saison 1 → contient_episode → Ep1, Ep2, Ep3
('c0000000-0000-0000-0000-000000000003',
 'e0000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000010',
 '{"ordre": 1}'),
('c0000000-0000-0000-0000-000000000003',
 'e0000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000011',
 '{"ordre": 2}'),
('c0000000-0000-0000-0000-000000000003',
 'e0000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000012',
 '{"ordre": 3}');

-- Chemin 2 : Série → DVD → Épisode (mêmes épisodes !)
INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES

-- Breaking Bad → édité_en → DVD Intégrale
('c0000000-0000-0000-0000-000000000004',
 'e0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000020',
 '{}'),

-- DVD Intégrale → inclut_episode → Ep1, Ep2, Ep3
('c0000000-0000-0000-0000-000000000005',
 'e0000000-0000-0000-0000-000000000020', 'e0000000-0000-0000-0000-000000000010',
 '{"disque": 1, "piste": 1}'),
('c0000000-0000-0000-0000-000000000005',
 'e0000000-0000-0000-0000-000000000020', 'e0000000-0000-0000-0000-000000000011',
 '{"disque": 1, "piste": 2}'),
('c0000000-0000-0000-0000-000000000005',
 'e0000000-0000-0000-0000-000000000020', 'e0000000-0000-0000-0000-000000000012',
 '{"disque": 1, "piste": 3}');

-- =============================================================================
-- 7. COLLECTIONS et COLLECTION ITEMS
-- =============================================================================

-- Collection "Mes DVDs" (type Possession) pour shinsaber
INSERT INTO collections (id, user_id, collection_type_id, name, description) VALUES
('f0000000-0000-0000-0000-000000000001',
 'a0000000-0000-0000-0000-000000000002',
 'd0000000-0000-0000-0000-000000000001',
 'Mes DVDs', 'Ma collection de DVDs');

-- Ajout du coffret DVD Intégrale dans "Mes DVDs"
INSERT INTO collection_items (collection_id, item_id, attributes) VALUES
('f0000000-0000-0000-0000-000000000001',
 'e0000000-0000-0000-0000-000000000020',
 '{"etat_physique": "neuf", "prix_achat": 45.00, "date_acquisition": "2020-12-25"}');

-- Collection "Séries vues" (type Visionnage) pour shinsaber
INSERT INTO collections (id, user_id, collection_type_id, name, description) VALUES
('f0000000-0000-0000-0000-000000000002',
 'a0000000-0000-0000-0000-000000000002',
 'd0000000-0000-0000-0000-000000000002',
 'Séries vues', 'Les séries que j''ai regardées');

-- Breaking Bad dans "Séries vues"
INSERT INTO collection_items (collection_id, item_id, attributes) VALUES
('f0000000-0000-0000-0000-000000000002',
 'e0000000-0000-0000-0000-000000000001',
 '{"date_vue": "2023-03-15", "note": 10, "progression": 100}');

-- =============================================================================
-- 8. TAGS
-- =============================================================================

INSERT INTO tags (id, name) VALUES
('aa000000-0000-0000-0000-000000000001', 'drame'),
('aa000000-0000-0000-0000-000000000002', 'thriller'),
('aa000000-0000-0000-0000-000000000003', 'crime');

INSERT INTO item_tags (item_id, tag_id) VALUES
('e0000000-0000-0000-0000-000000000001', 'aa000000-0000-0000-0000-000000000001'),
('e0000000-0000-0000-0000-000000000001', 'aa000000-0000-0000-0000-000000000002'),
('e0000000-0000-0000-0000-000000000001', 'aa000000-0000-0000-0000-000000000003');
