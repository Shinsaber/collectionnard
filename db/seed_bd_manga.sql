-- =============================================================================
-- Collectionnard — Jeu de données BD / Manga (lot 13, modèle : les.collection.md)
-- =============================================================================
-- Prérequis : migrations appliquées et db/seed.sql chargé (réutilise l'ItemType Series
-- b…01, le CollectionType Ownership d…01 et l'utilisateur shinsaber a…02).
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
-- UUID : b…0010-0017 item_types, c…0010-0016 relation_types, e…0100+ items,
--        f…0010 collection (plages libres de seed.sql).
-- =============================================================================

-- =============================================================================
-- 1. ITEM TYPES
-- =============================================================================
-- Attributs communs du domaine (voir les.collection.md § « Sources de référence ») :
--   external_ids : { wikidata: "Q…", bnf_ark: "ark:/12148/…" }
--   sources      : [{ source: wikidata|bnf|manual, retrieved_at: date }]

INSERT INTO item_types (id, name, description, attributes_schema) VALUES

-- Univers (franchise) — optionnel au-dessus des œuvres
('b0000000-0000-0000-0000-000000000010', 'Universe', 'Univers / franchise regroupant des œuvres', '{
    "type": "object",
    "required": ["title"],
    "x-title": { "fr": "Univers", "en": "Universe" },
    "x-description": { "fr": "Franchise regroupant des œuvres et des sagas", "en": "Franchise grouping works and sagas" },
    "x-label": "{title}",
    "x-order": ["title", "description", "images", "external_ids", "sources"],
    "x-labels": {
        "fr": { "title": "Titre", "description": "Description", "images": "Images", "external_ids": "Identifiants externes", "sources": "Sources" },
        "en": { "title": "Title", "description": "Description", "images": "Images", "external_ids": "External IDs", "sources": "Sources" }
    },
    "properties": {
        "title":        { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "description":  { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "images":       { "type": "object", "x-widget": "image-gallery", "x-image-categories": { "poster": { "ratio": "2/3" }, "banner": { "ratio": "3/1" }, "logo": { "ratio": "1/1" } }, "properties": { "default_image": { "type": "string" }, "poster": { "type": "array", "items": { "type": "string" } }, "banner": { "type": "array", "items": { "type": "string" } }, "logo": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false },
        "external_ids": { "type": "object", "properties": { "wikidata": { "type": "string", "pattern": "^Q[1-9][0-9]*$" }, "bnf_ark": { "type": "string", "pattern": "^ark:/12148/[a-z0-9]+$" } }, "additionalProperties": false },
        "sources":      { "type": "array", "items": { "type": "object", "required": ["source"], "properties": { "source": { "type": "string", "enum": ["wikidata", "bnf", "manual"] }, "retrieved_at": { "type": "string", "format": "date" } }, "additionalProperties": false } }
    },
    "additionalProperties": false
}'),

-- Saga (arc narratif d'un univers)
('b0000000-0000-0000-0000-000000000011', 'Saga', 'Arc narratif ou sous-ensemble d''un univers', '{
    "type": "object",
    "required": ["title"],
    "x-title": { "fr": "Saga", "en": "Saga" },
    "x-description": { "fr": "Arc narratif ou sous-ensemble d''un univers", "en": "Story arc or subset of a universe" },
    "x-label": "{title}",
    "x-order": ["title", "description", "order", "images", "external_ids", "sources"],
    "x-labels": {
        "fr": { "title": "Titre", "description": "Description", "order": "Ordre", "images": "Images", "external_ids": "Identifiants externes", "sources": "Sources" },
        "en": { "title": "Title", "description": "Description", "order": "Order", "images": "Images", "external_ids": "External IDs", "sources": "Sources" }
    },
    "properties": {
        "title":        { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "description":  { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "order":        { "type": "integer", "minimum": 1 },
        "images":       { "type": "object", "x-widget": "image-gallery", "x-image-categories": { "illustration": { "ratio": "16/9" } }, "properties": { "default_image": { "type": "string" }, "illustration": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false },
        "external_ids": { "type": "object", "properties": { "wikidata": { "type": "string", "pattern": "^Q[1-9][0-9]*$" }, "bnf_ark": { "type": "string", "pattern": "^ark:/12148/[a-z0-9]+$" } }, "additionalProperties": false },
        "sources":      { "type": "array", "items": { "type": "object", "required": ["source"], "properties": { "source": { "type": "string", "enum": ["wikidata", "bnf", "manual"] }, "retrieved_at": { "type": "string", "format": "date" } }, "additionalProperties": false } }
    },
    "additionalProperties": false
}'),

-- Manga (+ manhwa, manhua, webtoon)
('b0000000-0000-0000-0000-000000000012', 'Manga', 'Œuvre de type manga (manhwa, manhua, webtoon)', '{
    "type": "object",
    "required": ["title"],
    "x-title": { "fr": "Manga", "en": "Manga" },
    "x-description": { "fr": "Œuvre de type manga (manhwa, manhua, webtoon)", "en": "Manga work (manhwa, manhua, webtoon)" },
    "x-label": "{title}",
    "x-order": ["title", "original_title", "description", "sub_type", "demographic", "status", "start_year", "end_year", "volumes_vo", "images", "external_ids", "sources"],
    "x-labels": {
        "fr": { "title": "Titre", "original_title": "Titre original", "description": "Synopsis", "sub_type": "Sous-type", "demographic": "Public", "status": "Statut", "start_year": "Année de début", "end_year": "Année de fin", "volumes_vo": "Tomes (VO)", "images": "Images", "external_ids": "Identifiants externes", "sources": "Sources" },
        "en": { "title": "Title", "original_title": "Original title", "description": "Synopsis", "sub_type": "Sub-type", "demographic": "Demographic", "status": "Status", "start_year": "Start year", "end_year": "End year", "volumes_vo": "Volumes (original)", "images": "Images", "external_ids": "External IDs", "sources": "Sources" }
    },
    "properties": {
        "title":          { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "original_title": { "type": "string" },
        "description":    { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "sub_type":       { "type": "string", "enum": ["manga", "manhwa", "manhua", "webtoon"], "x-enum-labels": { "fr": { "manga": "Manga", "manhwa": "Manhwa", "manhua": "Manhua", "webtoon": "Webtoon" }, "en": { "manga": "Manga", "manhwa": "Manhwa", "manhua": "Manhua", "webtoon": "Webtoon" } } },
        "demographic":    { "type": "string", "enum": ["shonen", "shojo", "seinen", "josei", "kodomo"], "x-enum-labels": { "fr": { "shonen": "Shōnen", "shojo": "Shōjo", "seinen": "Seinen", "josei": "Josei", "kodomo": "Kodomo" }, "en": { "shonen": "Shōnen", "shojo": "Shōjo", "seinen": "Seinen", "josei": "Josei", "kodomo": "Kodomo" } } },
        "status":         { "type": "string", "enum": ["ongoing", "completed", "on_hold", "cancelled"], "x-enum-labels": { "fr": { "ongoing": "En cours", "completed": "Terminée", "on_hold": "En pause", "cancelled": "Abandonnée" }, "en": { "ongoing": "Ongoing", "completed": "Completed", "on_hold": "On hold", "cancelled": "Cancelled" } } },
        "start_year":     { "type": "integer", "minimum": 1800 },
        "end_year":       { "type": "integer", "minimum": 1800 },
        "volumes_vo":     { "type": "integer", "minimum": 1 },
        "images":         { "type": "object", "x-widget": "image-gallery", "x-image-categories": { "poster": { "ratio": "2/3" }, "cover": { "ratio": "2/3" }, "banner": { "ratio": "3/1" } }, "properties": { "default_image": { "type": "string" }, "poster": { "type": "array", "items": { "type": "string" } }, "cover": { "type": "array", "items": { "type": "string" } }, "banner": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false },
        "external_ids":   { "type": "object", "properties": { "wikidata": { "type": "string", "pattern": "^Q[1-9][0-9]*$" }, "bnf_ark": { "type": "string", "pattern": "^ark:/12148/[a-z0-9]+$" } }, "additionalProperties": false },
        "sources":        { "type": "array", "items": { "type": "object", "required": ["source"], "properties": { "source": { "type": "string", "enum": ["wikidata", "bnf", "manual"] }, "retrieved_at": { "type": "string", "format": "date" } }, "additionalProperties": false } }
    },
    "additionalProperties": false
}'),

-- BD (franco-belge, comics, roman graphique)
('b0000000-0000-0000-0000-000000000013', 'Comic', 'Bande dessinée (franco-belge, comics, roman graphique)', '{
    "type": "object",
    "required": ["title"],
    "x-title": { "fr": "BD", "en": "Comic" },
    "x-description": { "fr": "Bande dessinée (franco-belge, comics, roman graphique)", "en": "Comic (Franco-Belgian, comics, graphic novel)" },
    "x-label": "{title}",
    "x-order": ["title", "original_title", "description", "sub_type", "status", "start_year", "end_year", "volumes_vo", "images", "external_ids", "sources"],
    "x-labels": {
        "fr": { "title": "Titre", "original_title": "Titre original", "description": "Synopsis", "sub_type": "Sous-type", "status": "Statut", "start_year": "Année de début", "end_year": "Année de fin", "volumes_vo": "Albums (VO)", "images": "Images", "external_ids": "Identifiants externes", "sources": "Sources" },
        "en": { "title": "Title", "original_title": "Original title", "description": "Synopsis", "sub_type": "Sub-type", "status": "Status", "start_year": "Start year", "end_year": "End year", "volumes_vo": "Albums (original)", "images": "Images", "external_ids": "External IDs", "sources": "Sources" }
    },
    "properties": {
        "title":          { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "original_title": { "type": "string" },
        "description":    { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "sub_type":       { "type": "string", "enum": ["franco_belge", "comic", "graphic_novel", "strip"], "x-enum-labels": { "fr": { "franco_belge": "Franco-belge", "comic": "Comics", "graphic_novel": "Roman graphique", "strip": "Strip" }, "en": { "franco_belge": "Franco-Belgian", "comic": "Comics", "graphic_novel": "Graphic novel", "strip": "Comic strip" } } },
        "status":         { "type": "string", "enum": ["ongoing", "completed", "on_hold", "cancelled"], "x-enum-labels": { "fr": { "ongoing": "En cours", "completed": "Terminée", "on_hold": "En pause", "cancelled": "Abandonnée" }, "en": { "ongoing": "Ongoing", "completed": "Completed", "on_hold": "On hold", "cancelled": "Cancelled" } } },
        "start_year":     { "type": "integer", "minimum": 1800 },
        "end_year":       { "type": "integer", "minimum": 1800 },
        "volumes_vo":     { "type": "integer", "minimum": 1 },
        "images":         { "type": "object", "x-widget": "image-gallery", "x-image-categories": { "poster": { "ratio": "2/3" }, "cover": { "ratio": "3/4" }, "banner": { "ratio": "3/1" } }, "properties": { "default_image": { "type": "string" }, "poster": { "type": "array", "items": { "type": "string" } }, "cover": { "type": "array", "items": { "type": "string" } }, "banner": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false },
        "external_ids":   { "type": "object", "properties": { "wikidata": { "type": "string", "pattern": "^Q[1-9][0-9]*$" }, "bnf_ark": { "type": "string", "pattern": "^ark:/12148/[a-z0-9]+$" } }, "additionalProperties": false },
        "sources":        { "type": "array", "items": { "type": "object", "required": ["source"], "properties": { "source": { "type": "string", "enum": ["wikidata", "bnf", "manual"] }, "retrieved_at": { "type": "string", "format": "date" } }, "additionalProperties": false } }
    },
    "additionalProperties": false
}'),

-- Édition (publication d'une œuvre : originale, perfect, poche…)
('b0000000-0000-0000-0000-000000000014', 'Edition', 'Publication spécifique d''une œuvre', '{
    "type": "object",
    "required": ["title", "language"],
    "x-title": { "fr": "Édition", "en": "Edition" },
    "x-description": { "fr": "Publication spécifique d''une œuvre (originale, perfect, poche…)", "en": "Specific publication of a work (original, perfect, pocket…)" },
    "x-label": "{title}",
    "x-order": ["title", "language", "format", "status", "volume_count", "start_year", "end_year", "images", "external_ids", "sources"],
    "x-labels": {
        "fr": { "title": "Titre", "language": "Langue", "format": "Format", "status": "Statut", "volume_count": "Nombre de tomes", "start_year": "Année de début", "end_year": "Année de fin", "images": "Images", "external_ids": "Identifiants externes", "sources": "Sources" },
        "en": { "title": "Title", "language": "Language", "format": "Format", "status": "Status", "volume_count": "Number of volumes", "start_year": "Start year", "end_year": "End year", "images": "Images", "external_ids": "External IDs", "sources": "Sources" }
    },
    "properties": {
        "title":        { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "language":     { "type": "string", "enum": ["fr", "ja", "en", "ko", "zh", "es", "de", "it"], "x-enum-labels": { "fr": { "fr": "Français", "ja": "Japonais", "en": "Anglais", "ko": "Coréen", "zh": "Chinois", "es": "Espagnol", "de": "Allemand", "it": "Italien" }, "en": { "fr": "French", "ja": "Japanese", "en": "English", "ko": "Korean", "zh": "Chinese", "es": "Spanish", "de": "German", "it": "Italian" } } },
        "format":       { "type": "string", "enum": ["tankobon", "grand_format", "poche", "perfect", "deluxe", "collector", "omnibus", "album"], "x-enum-labels": { "fr": { "tankobon": "Tankōbon", "grand_format": "Grand format", "poche": "Poche", "perfect": "Perfect edition", "deluxe": "Deluxe", "collector": "Collector", "omnibus": "Omnibus", "album": "Album" }, "en": { "tankobon": "Tankōbon", "grand_format": "Large format", "poche": "Pocket", "perfect": "Perfect edition", "deluxe": "Deluxe", "collector": "Collector", "omnibus": "Omnibus", "album": "Album" } } },
        "status":       { "type": "string", "enum": ["ongoing", "completed", "on_hold", "out_of_print"], "x-enum-labels": { "fr": { "ongoing": "En cours", "completed": "Terminée", "on_hold": "En pause", "out_of_print": "Épuisée" }, "en": { "ongoing": "Ongoing", "completed": "Completed", "on_hold": "On hold", "out_of_print": "Out of print" } } },
        "volume_count": { "type": "integer", "minimum": 1 },
        "start_year":   { "type": "integer", "minimum": 1800 },
        "end_year":     { "type": "integer", "minimum": 1800 },
        "images":       { "type": "object", "x-widget": "image-gallery", "x-image-categories": { "cover": { "ratio": "2/3" }, "promo": { "ratio": "16/9" } }, "properties": { "default_image": { "type": "string" }, "cover": { "type": "array", "items": { "type": "string" } }, "promo": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false },
        "external_ids": { "type": "object", "properties": { "wikidata": { "type": "string", "pattern": "^Q[1-9][0-9]*$" }, "bnf_ark": { "type": "string", "pattern": "^ark:/12148/[a-z0-9]+$" } }, "additionalProperties": false },
        "sources":      { "type": "array", "items": { "type": "object", "required": ["source"], "properties": { "source": { "type": "string", "enum": ["wikidata", "bnf", "manual"] }, "retrieved_at": { "type": "string", "format": "date" } }, "additionalProperties": false } }
    },
    "additionalProperties": false
}'),

-- Tome (volume physique : l'item le plus souvent mis en collection)
('b0000000-0000-0000-0000-000000000015', 'Volume', 'Volume physique individuel (tome, album)', '{
    "type": "object",
    "required": ["number", "title"],
    "x-title": { "fr": "Tome", "en": "Volume" },
    "x-description": { "fr": "Volume physique individuel (tome, album)", "en": "Individual physical volume" },
    "x-label": "{number}. {title}",
    "x-order": ["number", "title", "subtitle", "isbn", "publication_date", "page_count", "retail_price", "images", "external_ids", "sources"],
    "x-labels": {
        "fr": { "number": "Numéro", "title": "Titre", "subtitle": "Sous-titre", "isbn": "ISBN", "publication_date": "Date de publication", "page_count": "Nombre de pages", "retail_price": "Prix catalogue (€)", "images": "Images", "external_ids": "Identifiants externes", "sources": "Sources" },
        "en": { "number": "Number", "title": "Title", "subtitle": "Subtitle", "isbn": "ISBN", "publication_date": "Publication date", "page_count": "Page count", "retail_price": "Retail price (€)", "images": "Images", "external_ids": "External IDs", "sources": "Sources" }
    },
    "properties": {
        "number":           { "type": "integer", "minimum": 0 },
        "title":            { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "subtitle":         { "type": "string" },
        "isbn":             { "type": "string", "pattern": "^97[89][0-9]{10}$", "description": "ISBN-13, chiffres seuls" },
        "publication_date": { "type": "string", "format": "date" },
        "page_count":       { "type": "integer", "minimum": 1 },
        "retail_price":     { "type": "number", "minimum": 0 },
        "images":           { "type": "object", "x-widget": "image-gallery", "x-image-categories": { "front": { "ratio": "2/3" }, "back": { "ratio": "2/3" }, "spine": { "ratio": "1/8" } }, "properties": { "default_image": { "type": "string" }, "front": { "type": "array", "items": { "type": "string" } }, "back": { "type": "array", "items": { "type": "string" } }, "spine": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false },
        "external_ids":     { "type": "object", "properties": { "wikidata": { "type": "string", "pattern": "^Q[1-9][0-9]*$" }, "bnf_ark": { "type": "string", "pattern": "^ark:/12148/[a-z0-9]+$" } }, "additionalProperties": false },
        "sources":          { "type": "array", "items": { "type": "object", "required": ["source"], "properties": { "source": { "type": "string", "enum": ["wikidata", "bnf", "manual"] }, "retrieved_at": { "type": "string", "format": "date" } }, "additionalProperties": false } }
    },
    "additionalProperties": false
}'),

-- Personne (auteur, scénariste, dessinateur…)
('b0000000-0000-0000-0000-000000000016', 'Person', 'Créateur : auteur, scénariste, dessinateur, coloriste, traducteur…', '{
    "type": "object",
    "required": ["last_name"],
    "x-title": { "fr": "Personne", "en": "Person" },
    "x-description": { "fr": "Créateur : auteur, scénariste, dessinateur, coloriste, traducteur…", "en": "Creator: author, writer, artist, colorist, translator…" },
    "x-label": "{first_name} {last_name}",
    "x-order": ["last_name", "first_name", "native_name", "nationality", "birth_date", "biography", "images", "external_ids", "sources"],
    "x-labels": {
        "fr": { "last_name": "Nom", "first_name": "Prénom", "native_name": "Nom natif", "nationality": "Nationalité (code pays)", "birth_date": "Date de naissance", "biography": "Biographie", "images": "Images", "external_ids": "Identifiants externes", "sources": "Sources" },
        "en": { "last_name": "Last name", "first_name": "First name", "native_name": "Native name", "nationality": "Nationality (country code)", "birth_date": "Birth date", "biography": "Biography", "images": "Images", "external_ids": "External IDs", "sources": "Sources" }
    },
    "properties": {
        "last_name":    { "type": "string", "minLength": 1 },
        "first_name":   { "type": "string" },
        "native_name":  { "type": "string" },
        "nationality":  { "type": "string", "pattern": "^[A-Z]{2}$", "description": "ISO 3166-1 alpha-2" },
        "birth_date":   { "type": "string", "format": "date" },
        "biography":    { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "images":       { "type": "object", "x-widget": "image-gallery", "x-image-categories": { "portrait": { "ratio": "3/4" } }, "properties": { "default_image": { "type": "string" }, "portrait": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false },
        "external_ids": { "type": "object", "properties": { "wikidata": { "type": "string", "pattern": "^Q[1-9][0-9]*$" }, "bnf_ark": { "type": "string", "pattern": "^ark:/12148/[a-z0-9]+$" } }, "additionalProperties": false },
        "sources":      { "type": "array", "items": { "type": "object", "required": ["source"], "properties": { "source": { "type": "string", "enum": ["wikidata", "bnf", "manual"] }, "retrieved_at": { "type": "string", "format": "date" } }, "additionalProperties": false } }
    },
    "additionalProperties": false
}'),

-- Maison d'édition
('b0000000-0000-0000-0000-000000000017', 'Publisher', 'Maison d''édition', '{
    "type": "object",
    "required": ["name"],
    "x-title": { "fr": "Maison d''édition", "en": "Publisher" },
    "x-description": { "fr": "Éditeur (Glénat, Kana, Dargaud, Dupuis…)", "en": "Publishing house" },
    "x-label": "{name}",
    "x-order": ["name", "country", "website", "description", "images", "external_ids", "sources"],
    "x-labels": {
        "fr": { "name": "Nom", "country": "Pays (code)", "website": "Site web", "description": "Description", "images": "Images", "external_ids": "Identifiants externes", "sources": "Sources" },
        "en": { "name": "Name", "country": "Country (code)", "website": "Website", "description": "Description", "images": "Images", "external_ids": "External IDs", "sources": "Sources" }
    },
    "properties": {
        "name":         { "type": "string", "minLength": 1 },
        "country":      { "type": "string", "pattern": "^[A-Z]{2}$", "description": "ISO 3166-1 alpha-2" },
        "website":      { "type": "string", "format": "uri" },
        "description":  { "type": "object", "x-widget": "i18n-text", "properties": { "fr": { "type": "string" }, "en": { "type": "string" } }, "propertyNames": { "pattern": "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$" }, "additionalProperties": { "type": "string" } },
        "images":       { "type": "object", "x-widget": "image-gallery", "x-image-categories": { "logo": { "ratio": "1/1" } }, "properties": { "default_image": { "type": "string" }, "logo": { "type": "array", "items": { "type": "string" } } }, "additionalProperties": false },
        "external_ids": { "type": "object", "properties": { "wikidata": { "type": "string", "pattern": "^Q[1-9][0-9]*$" }, "bnf_ark": { "type": "string", "pattern": "^ark:/12148/[a-z0-9]+$" } }, "additionalProperties": false },
        "sources":      { "type": "array", "items": { "type": "object", "required": ["source"], "properties": { "source": { "type": "string", "enum": ["wikidata", "bnf", "manual"] }, "retrieved_at": { "type": "string", "format": "date" } }, "additionalProperties": false } }
    },
    "additionalProperties": false
}');

-- =============================================================================
-- 2. RELATION TYPES
-- =============================================================================

INSERT INTO relation_types (id, name, description, attributes_schema) VALUES

('c0000000-0000-0000-0000-000000000010', 'contains_saga', 'Univers contient une saga', '{
    "type": "object",
    "x-title": { "fr": "Contient la saga", "en": "Contains saga" },
    "x-order": ["order"],
    "x-labels": { "fr": { "order": "Ordre" }, "en": { "order": "Order" } },
    "properties": { "order": { "type": "integer", "minimum": 1 } },
    "additionalProperties": false
}'),

('c0000000-0000-0000-0000-000000000011', 'contains_work', 'Univers ou saga contient une œuvre', '{
    "type": "object",
    "x-title": { "fr": "Contient l''œuvre", "en": "Contains work" },
    "x-order": ["order"],
    "x-labels": { "fr": { "order": "Ordre" }, "en": { "order": "Order" } },
    "properties": { "order": { "type": "integer", "minimum": 1 } },
    "additionalProperties": false
}'),

-- Rôles multiples : une seule relation par (œuvre, personne), `roles` en tableau
-- (respecte l'unicité (relation_type_id, source_item_id, target_item_id)).
('c0000000-0000-0000-0000-000000000012', 'created_by', 'Œuvre créée par une personne (rôles)', '{
    "type": "object",
    "required": ["roles"],
    "x-title": { "fr": "Créée par", "en": "Created by" },
    "x-order": ["roles"],
    "x-labels": { "fr": { "roles": "Rôles" }, "en": { "roles": "Roles" } },
    "properties": {
        "roles": {
            "type": "array",
            "minItems": 1,
            "uniqueItems": true,
            "items": { "type": "string", "enum": ["author", "writer", "artist", "colorist", "translator"] },
            "x-enum-labels": {
                "fr": { "author": "Auteur", "writer": "Scénariste", "artist": "Dessinateur", "colorist": "Coloriste", "translator": "Traducteur" },
                "en": { "author": "Author", "writer": "Writer", "artist": "Artist", "colorist": "Colorist", "translator": "Translator" }
            }
        }
    },
    "additionalProperties": false
}'),

('c0000000-0000-0000-0000-000000000013', 'published_as', 'Œuvre publiée sous forme d''une édition', '{
    "type": "object",
    "x-title": { "fr": "Publiée en", "en": "Published as" },
    "properties": {},
    "additionalProperties": false
}'),

('c0000000-0000-0000-0000-000000000014', 'edited_by', 'Édition publiée par une maison d''édition', '{
    "type": "object",
    "x-title": { "fr": "Éditée par", "en": "Edited by" },
    "properties": {},
    "additionalProperties": false
}'),

('c0000000-0000-0000-0000-000000000015', 'contains_volume', 'Édition contient un tome', '{
    "type": "object",
    "x-title": { "fr": "Contient le tome", "en": "Contains volume" },
    "x-order": ["order"],
    "x-labels": { "fr": { "order": "Ordre" }, "en": { "order": "Order" } },
    "properties": { "order": { "type": "integer", "minimum": 0 } },
    "additionalProperties": false
}'),

('c0000000-0000-0000-0000-000000000016', 'adapted_as', 'Œuvre adaptée (anime, film…)', '{
    "type": "object",
    "x-title": { "fr": "Adaptée en", "en": "Adapted as" },
    "x-order": ["adaptation_type"],
    "x-labels": { "fr": { "adaptation_type": "Type d''adaptation" }, "en": { "adaptation_type": "Adaptation type" } },
    "properties": {
        "adaptation_type": {
            "type": "string",
            "enum": ["anime", "film", "ova", "live_action"],
            "x-enum-labels": {
                "fr": { "anime": "Anime", "film": "Film", "ova": "OAV", "live_action": "Prises de vues réelles" },
                "en": { "anime": "Anime", "film": "Film", "ova": "OVA", "live_action": "Live action" }
            }
        }
    },
    "additionalProperties": false
}');

-- Types source / cible autorisés
-- b…10 Universe, b…11 Saga, b…12 Manga, b…13 Comic, b…14 Edition, b…15 Volume,
-- b…16 Person, b…17 Publisher, b…01 Series (seed.sql)
INSERT INTO relation_type_sources (relation_type_id, item_type_id) VALUES
('c0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000010'),
('c0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000010'),
('c0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000011'),
('c0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000012'),
('c0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000013'),
('c0000000-0000-0000-0000-000000000013', 'b0000000-0000-0000-0000-000000000012'),
('c0000000-0000-0000-0000-000000000013', 'b0000000-0000-0000-0000-000000000013'),
('c0000000-0000-0000-0000-000000000014', 'b0000000-0000-0000-0000-000000000014'),
('c0000000-0000-0000-0000-000000000015', 'b0000000-0000-0000-0000-000000000014'),
('c0000000-0000-0000-0000-000000000016', 'b0000000-0000-0000-0000-000000000012'),
('c0000000-0000-0000-0000-000000000016', 'b0000000-0000-0000-0000-000000000013');

INSERT INTO relation_type_targets (relation_type_id, item_type_id) VALUES
('c0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000011'),
('c0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000012'),
('c0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000013'),
('c0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000016'),
('c0000000-0000-0000-0000-000000000013', 'b0000000-0000-0000-0000-000000000014'),
('c0000000-0000-0000-0000-000000000014', 'b0000000-0000-0000-0000-000000000017'),
('c0000000-0000-0000-0000-000000000015', 'b0000000-0000-0000-0000-000000000015'),
('c0000000-0000-0000-0000-000000000016', 'b0000000-0000-0000-0000-000000000001');

-- =============================================================================
-- 3. COLLECTION TYPES
-- =============================================================================
-- Aucun nouveau : Ownership / Wishlist / Watching (seed.sql) conviennent aux Tomes et
-- Éditions. Le CollectionType « Lecture » est « à définir » dans les.collection.md.

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
