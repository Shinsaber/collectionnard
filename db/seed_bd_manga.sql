-- =============================================================================
-- Collectionnard — Jeu de données BD / Manga (lot 13, modèle : les.collection.md)
-- =============================================================================
-- Prérequis : migrations appliquées, paquets de domaine installés (books, tv, collections) et
-- db/seed.sql chargé (utilisateur shinsaber a…02).
--
-- Données réelles, reprises de sources ouvertes (voir docs/sources-externes.md) :
--   - Wikidata (CC0) : univers, œuvres, personnes, éditeurs — `external_ids.wikidata` ;
--   - BnF, catalogue général (Licence Ouverte Etalab : source + date de récupération) :
--     éditions et tomes — `external_ids.bnf_ark`, `sources[].retrieved_at`.
--   Récupération : 2026-10-11. Les synopsis / biographies reprennent les descriptions
--   courtes Wikidata (CC0), jamais le texte de Wikipédia (CC BY-SA). Pas d'images (droits).
--
-- Convention : clés JSON, noms de types, RelationTypes et valeurs d'enum en anglais
-- snake_case ASCII (« shōnen » → `shonen`, « tankōbon » → `tankobon`, « OAV » → `ova`) ;
-- l'affichage passe par x-title, x-labels et x-enum-labels (fr / en).
--
-- UUID : e…0100+ items, f…0010 collection (plages libres de seed.sql).
-- =============================================================================

-- =============================================================================
-- 1. TYPES
-- =============================================================================
-- Les types du domaine (ItemTypes b…0010-0017, RelationTypes c…0010-0016) sont le paquet de domaine
-- « books » (server/internal/domains/packages/books.json), installé avant ce seed avec « tv » et
-- « collections ». Attributs communs (les.collection.md § « Sources de référence ») :
--   external_ids : { wikidata: "Q…", bnf_ark: "ark:/12148/…" }
--   sources      : [{ source: wikidata|bnf|manual, retrieved_at: date }]

-- =============================================================================
-- 4. ITEMS — Dragon Ball
-- =============================================================================

INSERT INTO items (id, item_type_id, attributes) VALUES

-- Univers : wd:Q2020 (franchise médiatique), BnF 14534411x
('e0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000010', '{
    "title": { "fr": "Dragon Ball", "en": "Dragon Ball" },
    "description": { "fr": "franchise médiatique japonaise", "en": "Japanese media franchise" },
    "external_ids": { "wikidata": "Q2020", "bnf_ark": "ark:/12148/cb14534411x" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Sagas (découpage éditorial du document de modèle, sans équivalent Wikidata)
('e0000000-0000-0000-0000-000000000101', 'b0000000-0000-0000-0000-000000000011', '{
    "title": { "fr": "Saga de l''Enfance", "en": "Childhood saga" },
    "order": 1,
    "sources": [{ "source": "manual" }]
}'),
('e0000000-0000-0000-0000-000000000102', 'b0000000-0000-0000-0000-000000000011', '{
    "title": { "fr": "Dragon Ball Super", "en": "Dragon Ball Super" },
    "order": 2,
    "sources": [{ "source": "manual" }]
}'),

-- Manga Dragon Ball : wd:Q15051307 (1984-11-20 → 1995-05-23, 42 tomes, Jump Comics)
('e0000000-0000-0000-0000-000000000103', 'b0000000-0000-0000-0000-000000000012', '{
    "title": { "fr": "Dragon Ball", "en": "Dragon Ball" },
    "original_title": "ドラゴンボール",
    "description": { "fr": "série de manga", "en": "Japanese manga series" },
    "sub_type": "manga",
    "demographic": "shonen",
    "status": "completed",
    "start_year": 1984,
    "end_year": 1995,
    "volumes_vo": 42,
    "external_ids": { "wikidata": "Q15051307" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Manga Dragon Ball Super : wd:Q64230248 (depuis 2015-06-20, 24 tomes)
('e0000000-0000-0000-0000-000000000104', 'b0000000-0000-0000-0000-000000000012', '{
    "title": { "fr": "Dragon Ball Super", "en": "Dragon Ball Super" },
    "original_title": "ドラゴンボール超",
    "description": { "fr": "série de manga adaptée de la série animée éponyme", "en": "manga series within the larger Dragon Ball manga series" },
    "sub_type": "manga",
    "demographic": "shonen",
    "start_year": 2015,
    "volumes_vo": 24,
    "external_ids": { "wikidata": "Q64230248" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Personnes : Akira Toriyama wd:Q208582 (BnF 12336978f), Toyotarō wd:Q24838002
('e0000000-0000-0000-0000-000000000105', 'b0000000-0000-0000-0000-000000000016', '{
    "last_name": "Toriyama",
    "first_name": "Akira",
    "native_name": "鳥山 明",
    "nationality": "JP",
    "birth_date": "1955-04-05",
    "biography": { "fr": "auteur de manga et character designer japonais (1955-2024)", "en": "Japanese manga artist and character designer (1955-2024)" },
    "external_ids": { "wikidata": "Q208582", "bnf_ark": "ark:/12148/cb12336978f" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),
('e0000000-0000-0000-0000-000000000106', 'b0000000-0000-0000-0000-000000000016', '{
    "last_name": "Toyotarō",
    "native_name": "とよたろう",
    "nationality": "JP",
    "birth_date": "1978-05-17",
    "biography": { "en": "Japanese mangaka and illustrator" },
    "external_ids": { "wikidata": "Q24838002" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Éditeur : Glénat Éditions wd:Q1454296 (BnF 12925926s)
('e0000000-0000-0000-0000-000000000107', 'b0000000-0000-0000-0000-000000000017', '{
    "name": "Glénat",
    "country": "FR",
    "website": "https://glenat.com",
    "description": { "fr": "maison d''édition française", "en": "French publishing company" },
    "external_ids": { "wikidata": "Q1454296", "bnf_ark": "ark:/12148/cb12925926s" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Édition française (Glénat, 1993) : notice d'ensemble BnF cb34338597k
('e0000000-0000-0000-0000-000000000108', 'b0000000-0000-0000-0000-000000000014', '{
    "title": { "fr": "Dragon Ball - édition française (1993)", "en": "Dragon Ball - French edition (1993)" },
    "language": "fr",
    "format": "tankobon",
    "start_year": 1993,
    "external_ids": { "bnf_ark": "ark:/12148/cb34338597k" },
    "sources": [{ "source": "bnf", "retrieved_at": "2026-10-11" }]
}'),

-- Tomes 1 et 2 (Glénat, 1993) : notices BnF
('e0000000-0000-0000-0000-000000000109', 'b0000000-0000-0000-0000-000000000015', '{
    "number": 1,
    "title": { "fr": "Sangoku" },
    "isbn": "9782876952058",
    "external_ids": { "bnf_ark": "ark:/12148/cb356004006" },
    "sources": [{ "source": "bnf", "retrieved_at": "2026-10-11" }]
}'),
('e0000000-0000-0000-0000-000000000110', 'b0000000-0000-0000-0000-000000000015', '{
    "number": 2,
    "title": { "fr": "Kamehameha" },
    "isbn": "9782876952065",
    "external_ids": { "bnf_ark": "ark:/12148/cb356004068" },
    "sources": [{ "source": "bnf", "retrieved_at": "2026-10-11" }]
}'),

-- Adaptation : l'anime Dragon Ball wd:Q239506 (1986-1989), réutilise l'ItemType Series de seed.sql
-- (son schéma, partagé avec Breaking Bad, n'a pas d'external_ids).
('e0000000-0000-0000-0000-000000000111', 'b0000000-0000-0000-0000-000000000001', '{
    "title": "Dragon Ball",
    "synopsis": { "fr": "série télévisée d''animation japonaise", "en": "Japanese anime television series" },
    "start_year": 1986,
    "end_year": 1989
}');

-- =============================================================================
-- 5. ITEMS — Astérix (BD franco-belge, sans saga)
-- =============================================================================

INSERT INTO items (id, item_type_id, attributes) VALUES

-- Univers : wd:Q105030970 (franchise)
('e0000000-0000-0000-0000-000000000120', 'b0000000-0000-0000-0000-000000000010', '{
    "title": { "fr": "Astérix", "en": "Asterix" },
    "description": { "fr": "franchise médiatique", "en": "media franchise" },
    "external_ids": { "wikidata": "Q105030970" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Série : wd:Q42490 (depuis 1959-10-29 dans Pilote), BnF 44353554m
('e0000000-0000-0000-0000-000000000121', 'b0000000-0000-0000-0000-000000000013', '{
    "title": { "fr": "Astérix", "en": "Asterix" },
    "description": { "fr": "série française de bandes dessinées", "en": "series of French comic books" },
    "sub_type": "franco_belge",
    "status": "ongoing",
    "start_year": 1959,
    "external_ids": { "wikidata": "Q42490", "bnf_ark": "ark:/12148/cb44353554m" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Personnes : René Goscinny wd:Q192214 (BnF 11905455f), Albert Uderzo wd:Q206685 (BnF 11927309w)
('e0000000-0000-0000-0000-000000000122', 'b0000000-0000-0000-0000-000000000016', '{
    "last_name": "Goscinny",
    "first_name": "René",
    "nationality": "FR",
    "birth_date": "1926-08-14",
    "biography": { "fr": "scénariste français de bande dessinée", "en": "French comic book artist and author (1926-1977)" },
    "external_ids": { "wikidata": "Q192214", "bnf_ark": "ark:/12148/cb11905455f" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),
('e0000000-0000-0000-0000-000000000123', 'b0000000-0000-0000-0000-000000000016', '{
    "last_name": "Uderzo",
    "first_name": "Albert",
    "nationality": "FR",
    "birth_date": "1927-04-25",
    "biography": { "fr": "dessinateur français de bande dessinée (1927–2020)", "en": "French comic book artist (1927–2020)" },
    "external_ids": { "wikidata": "Q206685", "bnf_ark": "ark:/12148/cb11927309w" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Éditeurs : Dargaud wd:Q1165843 (BnF 13922158m), Les Éditions Albert René wd:Q3236518
('e0000000-0000-0000-0000-000000000124', 'b0000000-0000-0000-0000-000000000017', '{
    "name": "Dargaud",
    "country": "FR",
    "website": "http://www.dargaud.com",
    "description": { "fr": "maison d''édition française", "en": "publisher of comics" },
    "external_ids": { "wikidata": "Q1165843", "bnf_ark": "ark:/12148/cb13922158m" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),
('e0000000-0000-0000-0000-000000000125', 'b0000000-0000-0000-0000-000000000017', '{
    "name": "Les Éditions Albert René",
    "country": "FR",
    "website": "https://www.asterix.com/",
    "description": { "fr": "maison d''édition française", "en": "French publishing house" },
    "external_ids": { "wikidata": "Q3236518" },
    "sources": [{ "source": "wikidata", "retrieved_at": "2026-10-11" }]
}'),

-- Édition : collection « Une aventure d'Astérix le Gaulois » (Dargaud puis Albert René), BnF cb371381846
('e0000000-0000-0000-0000-000000000126', 'b0000000-0000-0000-0000-000000000014', '{
    "title": { "fr": "Une aventure d''Astérix le Gaulois", "en": "Asterix the Gaul (French albums)" },
    "language": "fr",
    "format": "album",
    "external_ids": { "bnf_ark": "ark:/12148/cb371381846" },
    "sources": [{ "source": "bnf", "retrieved_at": "2026-10-11" }]
}'),

-- Albums 24 (Dargaud, 1979, dernier de Goscinny) et 26 (Albert René, 1981, premier d'Uderzo seul)
('e0000000-0000-0000-0000-000000000127', 'b0000000-0000-0000-0000-000000000015', '{
    "number": 24,
    "title": { "fr": "Astérix chez les Belges", "en": "Asterix in Belgium" },
    "isbn": "9782205011500",
    "page_count": 48,
    "external_ids": { "bnf_ark": "ark:/12148/cb34628381z" },
    "sources": [{ "source": "bnf", "retrieved_at": "2026-10-11" }]
}'),
('e0000000-0000-0000-0000-000000000128', 'b0000000-0000-0000-0000-000000000015', '{
    "number": 26,
    "title": { "fr": "L''Odyssée d''Astérix", "en": "Asterix and the Black Gold" },
    "isbn": "9782864970040",
    "page_count": 48,
    "external_ids": { "bnf_ark": "ark:/12148/cb34722387r" },
    "sources": [{ "source": "bnf", "retrieved_at": "2026-10-11" }]
}');

-- =============================================================================
-- 6. RELATIONS
-- =============================================================================

INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES
-- Dragon Ball : univers → sagas → œuvres
('c0000000-0000-0000-0000-000000000010', 'e0000000-0000-0000-0000-000000000100', 'e0000000-0000-0000-0000-000000000101', '{"order": 1}'),
('c0000000-0000-0000-0000-000000000010', 'e0000000-0000-0000-0000-000000000100', 'e0000000-0000-0000-0000-000000000102', '{"order": 2}'),
('c0000000-0000-0000-0000-000000000011', 'e0000000-0000-0000-0000-000000000101', 'e0000000-0000-0000-0000-000000000103', '{"order": 1}'),
('c0000000-0000-0000-0000-000000000011', 'e0000000-0000-0000-0000-000000000102', 'e0000000-0000-0000-0000-000000000104', '{"order": 1}'),
-- Créateurs : Toriyama auteur complet de Dragon Ball ; sur Super, la MÊME personne
-- est scénariste et Toyotarō dessinateur (Wikidata P50 sur les deux œuvres)
('c0000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000103', 'e0000000-0000-0000-0000-000000000105', '{"roles": ["author"]}'),
('c0000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000104', 'e0000000-0000-0000-0000-000000000105', '{"roles": ["writer"]}'),
('c0000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000104', 'e0000000-0000-0000-0000-000000000106', '{"roles": ["artist"]}'),
-- Édition française, éditeur, tomes
('c0000000-0000-0000-0000-000000000013', 'e0000000-0000-0000-0000-000000000103', 'e0000000-0000-0000-0000-000000000108', '{}'),
('c0000000-0000-0000-0000-000000000014', 'e0000000-0000-0000-0000-000000000108', 'e0000000-0000-0000-0000-000000000107', '{}'),
('c0000000-0000-0000-0000-000000000015', 'e0000000-0000-0000-0000-000000000108', 'e0000000-0000-0000-0000-000000000109', '{"order": 1}'),
('c0000000-0000-0000-0000-000000000015', 'e0000000-0000-0000-0000-000000000108', 'e0000000-0000-0000-0000-000000000110', '{"order": 2}'),
-- Adaptation anime (wd:Q239506 « basé sur » wd:Q15051307)
('c0000000-0000-0000-0000-000000000016', 'e0000000-0000-0000-0000-000000000103', 'e0000000-0000-0000-0000-000000000111', '{"adaptation_type": "anime"}'),

-- Astérix : univers → œuvre directement (sans saga)
('c0000000-0000-0000-0000-000000000011', 'e0000000-0000-0000-0000-000000000120', 'e0000000-0000-0000-0000-000000000121', '{"order": 1}'),
-- Rôles multiples : Uderzo dessinateur puis aussi scénariste après 1977 (une seule relation)
('c0000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000121', 'e0000000-0000-0000-0000-000000000122', '{"roles": ["writer"]}'),
('c0000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000121', 'e0000000-0000-0000-0000-000000000123', '{"roles": ["writer", "artist"]}'),
-- Édition, éditeurs successifs, albums
('c0000000-0000-0000-0000-000000000013', 'e0000000-0000-0000-0000-000000000121', 'e0000000-0000-0000-0000-000000000126', '{}'),
('c0000000-0000-0000-0000-000000000014', 'e0000000-0000-0000-0000-000000000126', 'e0000000-0000-0000-0000-000000000124', '{}'),
('c0000000-0000-0000-0000-000000000014', 'e0000000-0000-0000-0000-000000000126', 'e0000000-0000-0000-0000-000000000125', '{}'),
('c0000000-0000-0000-0000-000000000015', 'e0000000-0000-0000-0000-000000000126', 'e0000000-0000-0000-0000-000000000127', '{"order": 24}'),
('c0000000-0000-0000-0000-000000000015', 'e0000000-0000-0000-0000-000000000126', 'e0000000-0000-0000-0000-000000000128', '{"order": 26}');

-- =============================================================================
-- 7. COLLECTION D'EXEMPLE (shinsaber, type Ownership d…01)
-- =============================================================================

INSERT INTO collections (id, user_id, collection_type_id, name, description) VALUES
('f0000000-0000-0000-0000-000000000010',
 'a0000000-0000-0000-0000-000000000002',
 'd0000000-0000-0000-0000-000000000001',
 'Mes mangas', 'Tomes possédés');

INSERT INTO collection_items (collection_id, item_id, attributes) VALUES
('f0000000-0000-0000-0000-000000000010', 'e0000000-0000-0000-0000-000000000109',
 '{"condition": "very_good", "purchase_price": 6.50, "acquisition_date": "2021-03-10"}'),
('f0000000-0000-0000-0000-000000000010', 'e0000000-0000-0000-0000-000000000110',
 '{"condition": "new", "purchase_price": 6.50, "acquisition_date": "2021-03-10"}');

-- =============================================================================
-- 8. ROMANS (P3-3) — générés par le connecteur « novels » (Wikidata CC0 + BnF Licence Ouverte,
--    récupérés le 2026-10-17) : L'Étranger (une édition Folio = le livre, cycle de l'absurde)
--    et Les Misérables (édition Pocket 2019 en trois tomes). UUID e…0200-0211.
-- =============================================================================

INSERT INTO items (id, item_type_id, attributes) VALUES
-- work : wikidata:Q163297
('e0000000-0000-0000-0000-000000000200', 'b0000000-0000-0000-0000-000000000018', '{"description":{"en":"1942 novel by Albert Camus","fr":"roman d’Albert Camus"},"external_ids":{"bnf_ark":"ark:/12148/cb11958514w","wikidata":"Q163297"},"first_publication_year":1942,"form":"novel","original_language":"fr","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}],"title":{"en":"The Stranger","fr":"L''Étranger"}}'),
-- universe : wikidata:Q3008298
('e0000000-0000-0000-0000-000000000201', 'b0000000-0000-0000-0000-000000000011', '{"description":{"en":"literary tetralogy by Albert Camus","fr":"absurde camusien"},"external_ids":{"wikidata":"Q3008298"},"sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}],"title":{"en":"The strange writer","fr":"philosophie d''Albert Camus"}}'),
-- person : wikidata:Q34670
('e0000000-0000-0000-0000-000000000202', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French philosopher, author, and journalist (1913–1960)","fr":"écrivain, philosophe et journaliste français"},"birth_date":"1913-11-07","external_ids":{"bnf_ark":"ark:/12148/cb118949856","wikidata":"Q34670"},"first_name":"Albert","last_name":"Camus","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- edition : bnf:cb436623939
('e0000000-0000-0000-0000-000000000203', 'b0000000-0000-0000-0000-000000000014', '{"collection":"Collection Folio","external_ids":{"bnf_ark":"ark:/12148/cb436623939"},"isbn":"9782070453177","language":"fr","retail_price":8.7,"sources":[{"retrieved_at":"2026-10-17","source":"bnf"}],"start_year":2013,"title":{"fr":"L''étranger"}}'),
-- publisher : publisher:gallimard
('e0000000-0000-0000-0000-000000000204', 'b0000000-0000-0000-0000-000000000017', '{"name":"Gallimard","sources":[{"retrieved_at":"2026-10-17","source":"bnf"}]}'),
-- work : wikidata:Q180736
('e0000000-0000-0000-0000-000000000205', 'b0000000-0000-0000-0000-000000000018', '{"description":{"en":"1862 novel by Victor Hugo","fr":"roman de Victor Hugo"},"external_ids":{"bnf_ark":"ark:/12148/cb13516296h","wikidata":"Q180736"},"first_publication_year":1862,"form":"novel","genres":["historical"],"original_language":"fr","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}],"title":{"en":"Les Misérables","fr":"Les Misérables"}}'),
-- person : wikidata:Q535
('e0000000-0000-0000-0000-000000000206', 'b0000000-0000-0000-0000-000000000016', '{"biography":{"en":"French novelist, poet, dramatist and politician (1802–1885)","fr":"romancier, poète, dramaturge, dessinateur et politicien français (1802–1885)"},"birth_date":"1802-02-26","external_ids":{"bnf_ark":"ark:/12148/cb11907966z","wikidata":"Q535"},"first_name":"Victor","last_name":"Hugo","nationality":"FR","sources":[{"retrieved_at":"2026-10-17","source":"wikidata"}]}'),
-- edition : bnf:cb45767941c
('e0000000-0000-0000-0000-000000000207', 'b0000000-0000-0000-0000-000000000014', '{"collection":"Pocket","external_ids":{"bnf_ark":"ark:/12148/cb45767941c"},"language":"fr","sources":[{"retrieved_at":"2026-10-17","source":"bnf"}],"start_year":2019,"title":{"fr":"Les misérables"},"volume_count":3}'),
-- publisher : publisher:pocket
('e0000000-0000-0000-0000-000000000208', 'b0000000-0000-0000-0000-000000000017', '{"name":"Pocket","sources":[{"retrieved_at":"2026-10-17","source":"bnf"}]}'),
-- volume : bnf:cb45718977k
('e0000000-0000-0000-0000-000000000209', 'b0000000-0000-0000-0000-000000000015', '{"external_ids":{"bnf_ark":"ark:/12148/cb45718977k"},"isbn":"9782266295901","number":1,"page_count":859,"retail_price":6.9,"sources":[{"retrieved_at":"2026-10-17","source":"bnf"}],"title":{"fr":"Les misérables"}}'),
-- volume : bnf:cb47420905r
('e0000000-0000-0000-0000-000000000210', 'b0000000-0000-0000-0000-000000000015', '{"external_ids":{"bnf_ark":"ark:/12148/cb47420905r"},"isbn":"9782266296007","number":2,"page_count":891,"retail_price":6.9,"sources":[{"retrieved_at":"2026-10-17","source":"bnf"}],"title":{"fr":"Les misérables"}}'),
-- volume : bnf:cb45840019s
('e0000000-0000-0000-0000-000000000211', 'b0000000-0000-0000-0000-000000000015', '{"external_ids":{"bnf_ark":"ark:/12148/cb45840019s"},"isbn":"9782266296137","number":3,"page_count":439,"retail_price":6.9,"sources":[{"retrieved_at":"2026-10-17","source":"bnf"}],"title":{"fr":"Les misérables"}}');

INSERT INTO item_relations (relation_type_id, source_item_id, target_item_id, attributes) VALUES
('c0000000-0000-0000-0000-000000000011', 'e0000000-0000-0000-0000-000000000201', 'e0000000-0000-0000-0000-000000000200', '{"order":1}'),
('c0000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000200', 'e0000000-0000-0000-0000-000000000202', '{"roles":["author"]}'),
('c0000000-0000-0000-0000-000000000012', 'e0000000-0000-0000-0000-000000000205', 'e0000000-0000-0000-0000-000000000206', '{"roles":["author"]}'),
('c0000000-0000-0000-0000-000000000013', 'e0000000-0000-0000-0000-000000000200', 'e0000000-0000-0000-0000-000000000203', '{}'),
('c0000000-0000-0000-0000-000000000013', 'e0000000-0000-0000-0000-000000000205', 'e0000000-0000-0000-0000-000000000207', '{}'),
('c0000000-0000-0000-0000-000000000014', 'e0000000-0000-0000-0000-000000000203', 'e0000000-0000-0000-0000-000000000204', '{}'),
('c0000000-0000-0000-0000-000000000014', 'e0000000-0000-0000-0000-000000000207', 'e0000000-0000-0000-0000-000000000208', '{}'),
('c0000000-0000-0000-0000-000000000015', 'e0000000-0000-0000-0000-000000000207', 'e0000000-0000-0000-0000-000000000209', '{"order":1}'),
('c0000000-0000-0000-0000-000000000015', 'e0000000-0000-0000-0000-000000000207', 'e0000000-0000-0000-0000-000000000211', '{"order":3}'),
('c0000000-0000-0000-0000-000000000015', 'e0000000-0000-0000-0000-000000000207', 'e0000000-0000-0000-0000-000000000210', '{"order":2}');
