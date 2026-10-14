---
description: Transformer un document de modèle de données (ex. les.collection.md) en SQL de seed (types + exemple)
argument-hint: <fichier du modèle, ex. les.collection.md>
---

Lis le document de modèle `$ARGUMENTS`, puis `db/migrations/001_init.sql` et `db/seed.sql` pour reprendre exactement le format existant (un changement de schéma = nouvelle migration `db/migrations/NNN_*.sql`, jamais une édition de 001).

Produis un nouveau fichier `db/seed_<domaine>.sql` (ne modifie pas `seed.sql`) contenant :

1. Les `item_types` avec leur `attributes_schema` JSON Schema complet : `required`, `properties`, `enum`, `x-order`, `x-labels` (fr + en), `x-widget` (`i18n-text`, `image-gallery`…) et `x-image-categories` si le modèle mentionne des images.
2. Les `relation_types` avec leurs lignes dans `relation_type_sources` / `relation_type_targets` (plusieurs sources/cibles autorisées) et le schéma de leurs attributs.
3. Les `collection_types` nouveaux uniquement (réutilise Possession / Visionnage / Wishlist existants).
4. Le jeu d'exemple décrit dans le document (items + relations), en réutilisant les items existants quand le modèle y fait référence (ex. une Série TV déjà présente).

Contraintes :

- UUID lisibles avec les préfixes du projet (`b…` item_types, `c…` relation_types, `d…` collection_types, `e…` items) dans une plage libre : vérifie les UUID déjà utilisés avant d'en choisir.
- Les points listés comme « à définir » ou « TODO » dans le document : ne les invente pas, liste-les-moi à la fin.
- Indique comment brancher le fichier dans `docker-compose.yml` (`03-seed_<domaine>.sql`) et dans le script `pg-start` du `flake.nix`, mais demande-moi avant de modifier ces fichiers.
