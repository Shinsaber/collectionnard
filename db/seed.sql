-- =============================================================================
-- Collectionnard — Jeu de données de test MVP : Séries TV
-- =============================================================================
-- Prérequis : migrations appliquées (colec-server migrate). Appliqué automatiquement par
-- `colec-server migrate` si SEED_ON_EMPTY=true et la table users est vide (pg-start, compose).
-- Scénario : Breaking Bad avec multi-chemin (Broadcast + DVD)
--
-- Convention : clés JSON, noms de RelationTypes / ItemTypes / CollectionTypes et valeurs d'enum
-- en anglais snake_case ; les libellés affichés viennent de x-labels (fr / en) ; le libellé d'un
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
-- 2. ITEM TYPES avec JSON Schema
-- =============================================================================

INSERT INTO item_types (id, name, description, attributes_schema) VALUES

-- Série
('b0000000-0000-0000-0000-000000000001', 'Series', 'Série télévisée', '{
    "type": "object",
    "required": ["title"],
    "x-label": "{title}",
    "x-order": ["title", "synopsis", "start_year", "end_year", "images"],
    "x-labels": {
        "fr": { "title": "Titre", "synopsis": "Synopsis", "start_year": "Année de début", "end_year": "Année de fin", "images": "Images" },
        "en": { "title": "Title", "synopsis": "Synopsis", "start_year": "Start Year", "end_year": "End Year", "images": "Images" }
    },
    "properties": {
        "title":      { "type": "string",  "minLength": 1 },
        "synopsis":   { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "additionalProperties": false },
        "start_year": { "type": "integer", "minimum": 1900 },
        "end_year":   { "type": "integer", "minimum": 1900 },
        "images":     { "type": "object", "x-widget": "image-gallery", "x-image-categories": {"poster": {"ratio": "2/3"}, "screenshot": {"ratio": "16/9"}, "banner": {"ratio": "3/1"}}, "properties": { "default_image": { "type": "string" }, "poster": { "type": "array", "items": { "type": "string" } }, "screenshot": { "type": "array", "items": { "type": "string" } }, "banner": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false }
    },
    "additionalProperties": false
}'),

-- Diffusion (chaîne TV, plateforme de streaming…)
('b0000000-0000-0000-0000-000000000002', 'Broadcast', 'Mode de diffusion (TV, streaming…)', '{
    "type": "object",
    "required": ["name"],
    "x-label": "{name}",
    "x-order": ["name", "type", "channel"],
    "x-labels": {
        "fr": { "name": "Nom", "type": "Type", "channel": "Support" },
        "en": { "name": "Name", "type": "Type", "channel": "Support" }
    },
    "properties": {
        "name":    { "type": "string", "minLength": 1 },
        "type":    { "type": "string", "enum": ["tv", "streaming", "cable", "web"] },
        "channel": { "type": "string" }
    },
    "additionalProperties": false
}'),

-- Saison
('b0000000-0000-0000-0000-000000000003', 'Season', 'Saison d''une série', '{
    "type": "object",
    "required": ["number"],
    "x-label": "{name}",
    "x-order": ["name", "number", "year"],
    "x-labels": {
        "fr": { "name": "Nom", "number": "Numéro", "year": "Année" },
        "en": { "name": "Name", "number": "Number", "year": "Year" }
    },
    "properties": {
        "name":   { "type": "string", "minLength": 1 },
        "number": { "type": "integer", "minimum": 1 },
        "year":   { "type": "integer", "minimum": 1900 }
    },
    "additionalProperties": false
}'),

-- Épisode
('b0000000-0000-0000-0000-000000000004', 'Episode', 'Épisode d''une série', '{
    "type": "object",
    "required": ["number", "title"],
    "x-label": "{number}. {title}",
    "x-order": ["number", "title", "duration", "synopsis"],
    "x-labels": {
        "fr": { "number": "Numéro", "title": "Titre", "duration": "Durée (min)", "synopsis": "Synopsis" },
        "en": { "number": "Number", "title": "Title", "duration": "Duration (min)", "synopsis": "Synopsis" }
    },
    "properties": {
        "number":   { "type": "integer", "minimum": 1 },
        "title":    { "type": "string",  "minLength": 1 },
        "duration": { "type": "integer", "minimum": 1, "description": "Durée en minutes" },
        "synopsis": { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "additionalProperties": false }
    },
    "additionalProperties": false
}'),

-- DVD
('b0000000-0000-0000-0000-000000000005', 'DVD', 'Édition DVD/Blu-ray', '{
    "type": "object",
    "required": ["title"],
    "x-label": "{title}",
    "x-order": ["title", "publisher", "release_year", "disc_count"],
    "x-labels": {
        "fr": { "title": "Titre", "publisher": "Éditeur", "release_year": "Année de sortie", "disc_count": "Nombre de disques" },
        "en": { "title": "Title", "publisher": "Publisher", "release_year": "Release Year", "disc_count": "Number of Discs" }
    },
    "properties": {
        "title":        { "type": "string",  "minLength": 1 },
        "publisher":    { "type": "string" },
        "release_year": { "type": "integer", "minimum": 1900 },
        "disc_count":   { "type": "integer", "minimum": 1 }
    },
    "additionalProperties": false
}');

-- =============================================================================
-- 3. RELATION TYPES avec JSON Schema
-- =============================================================================

INSERT INTO relation_types (id, name, description, attributes_schema) VALUES

-- Série → Diffusion
('c0000000-0000-0000-0000-000000000001', 'broadcast_on', 'Série diffusée via un mode de diffusion', '{
    "type": "object",
    "x-order": ["broadcast_start_year"],
    "x-labels": {
        "fr": { "broadcast_start_year": "Année de début de diffusion" },
        "en": { "broadcast_start_year": "Broadcast Start Year" }
    },
    "properties": {
        "broadcast_start_year": { "type": "integer", "minimum": 1900 }
    },
    "additionalProperties": false
}'),

-- Diffusion → Saison
('c0000000-0000-0000-0000-000000000002', 'contains_season', 'Diffusion contient une saison', '{
    "type": "object",
    "properties": {},
    "additionalProperties": false
}'),

-- Saison → Épisode (attribut `order` : convention de tri des relations dans le front)
('c0000000-0000-0000-0000-000000000003', 'contains_episode', 'Saison contient un épisode', '{
    "type": "object",
    "x-order": ["order"],
    "x-labels": {
        "fr": { "order": "Ordre" },
        "en": { "order": "Order" }
    },
    "properties": {
        "order": { "type": "integer", "minimum": 1 }
    },
    "additionalProperties": false
}'),

-- Série → DVD
('c0000000-0000-0000-0000-000000000004', 'released_as', 'Série éditée en DVD/Blu-ray', '{
    "type": "object",
    "properties": {},
    "additionalProperties": false
}'),

-- DVD → Épisode
('c0000000-0000-0000-0000-000000000005', 'includes_episode', 'DVD inclut un épisode', '{
    "type": "object",
    "x-order": ["disc", "track"],
    "x-labels": {
        "fr": { "disc": "Disque", "track": "Piste" },
        "en": { "disc": "Disc", "track": "Track" }
    },
    "properties": {
        "disc":  { "type": "integer", "minimum": 1 },
        "track": { "type": "integer", "minimum": 1 }
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

('d0000000-0000-0000-0000-000000000001', 'Ownership', 'Items possédés physiquement', '{
    "type": "object",
    "x-order": ["condition", "purchase_price", "acquisition_date"],
    "x-labels": {
        "fr": { "condition": "État physique", "purchase_price": "Prix d''achat", "acquisition_date": "Date d''acquisition" },
        "en": { "condition": "Physical Condition", "purchase_price": "Purchase Price", "acquisition_date": "Acquisition Date" }
    },
    "properties": {
        "condition":        { "type": "string", "enum": ["new", "very_good", "good", "fair", "poor"] },
        "purchase_price":   { "type": "number", "minimum": 0 },
        "acquisition_date": { "type": "string", "format": "date" }
    },
    "additionalProperties": false
}'),

('d0000000-0000-0000-0000-000000000002', 'Watching', 'Séries/films vus', '{
    "type": "object",
    "x-order": ["watched_at", "rating", "progress"],
    "x-labels": {
        "fr": { "watched_at": "Date de visionnage", "rating": "Note", "progress": "Progression" },
        "en": { "watched_at": "Watch Date", "rating": "Rating", "progress": "Progress" }
    },
    "properties": {
        "watched_at": { "type": "string", "format": "date" },
        "rating":     { "type": "integer", "minimum": 0, "maximum": 10 },
        "progress":   { "type": "integer", "minimum": 0, "maximum": 100, "description": "Pourcentage" }
    },
    "additionalProperties": false
}'),

('d0000000-0000-0000-0000-000000000003', 'Wishlist', 'Items souhaités', '{
    "type": "object",
    "x-order": ["priority", "max_budget"],
    "x-labels": {
        "fr": { "priority": "Priorité", "max_budget": "Budget maximum" },
        "en": { "priority": "Priority", "max_budget": "Maximum Budget" }
    },
    "properties": {
        "priority":   { "type": "integer", "minimum": 1, "maximum": 5 },
        "max_budget": { "type": "number",  "minimum": 0 }
    },
    "additionalProperties": false
}');

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
