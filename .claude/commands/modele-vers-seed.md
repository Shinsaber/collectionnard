---
description: Transformer un document de modèle de données (ex. docs/domaines/musique.md) en paquet de domaine (types) et seed de démonstration
argument-hint: <fichier du modèle, ex. docs/domaines/musique.md>
---

Lis le document de modèle `$ARGUMENTS`, puis un paquet de domaine existant (`server/internal/domains/packages/music.json`)
et son seed (`db/seed_music.sql`) pour reprendre exactement le format en place. Un changement de schéma BDD = nouvelle
migration `db/migrations/NNN_*.sql`, jamais une édition de `001_init.sql`.

Produis :

1. Un **paquet de domaine** `server/internal/domains/packages/<domaine>.json` (format de l'import, `package_version: 1`,
   `requires` si le modèle réutilise `people` ou un autre paquet) : `item_types` avec leur `attributes_schema` JSON Schema
   complet (`required`, `properties`, `enum`, `x-order`, `x-labels` fr + en, `x-title` / `x-description`, `x-enum-labels`,
   `x-widget` (`i18n-text`, `image-gallery`, `links`), `x-image-categories`, `x-identifier` sur les identifiants,
   `external_ids` et `sources`) ; `relation_types` avec `source_item_type_ids` / `target_item_type_ids` (plusieurs permis)
   et `x-completeness` sur les relations de composition ; `collection_types` nouveaux uniquement (Possession / Visionnage /
   Wishlist existent dans le paquet `collections`).
2. Un **seed de démonstration** `db/seed_<domaine>.sql` (items et relations seulement : les types viennent du paquet),
   avec des données réelles et légalement réutilisables (`external_ids`, `sources`), en réutilisant les items existants
   quand le modèle y renvoie (ex. une Série TV déjà présente). Ne modifie pas `seed.sql`.

Contraintes :

- UUID lisibles avec les préfixes du projet (`b…` item_types, `c…` relation_types, `d…` collection_types, `e…` items)
  dans une plage libre : vérifie les UUID déjà pris dans les paquets et les seeds avant d'en choisir.
- Les points « à définir » ou « TODO » du document : ne les invente pas, liste-les-moi à la fin.
- Branchement : le seed s'ajoute à `defaultSeedNames` (`server/internal/dbmigrate/dbmigrate.go`) et à son test ; les
  fixtures des tests front se régénèrent (`web/CLAUDE.md`, `scripts/seed-fixtures.mjs`) ; les tableaux de `CLAUDE.md`
  et du README se complètent. Indique ces modifications, mais demande-moi avant de les faire.
