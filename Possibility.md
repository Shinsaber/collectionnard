# Lots — branche `evolutions-socle`

Un commit par lot ; contrat d'API d'abord (`api/openapi.yaml` → model → store → handler + test →
`types.ts` / `client.ts` → vues). Ordre : du plus petit au plus gros.

## Lot E1 — Identifiants insensibles à la casse

- [x] Migration `004` : contrôle préalable (échec explicite si deux comptes ne diffèrent que par la
      casse), remplacement de la contrainte `UNIQUE (username)` par un index unique sur `lower(username)`.
- [x] Connexion : recherche par `lower(username) = lower($1)` ; le nom reste affiché tel que saisi.
- [x] Tests d'intégration du store : `Admin` / `ADMIN` trouvent le même compte, un doublon de casse est refusé
      (le handler, testé sur MockStore, ne voit pas la casse).

## Lot E2 — Ordre d'affichage des types d'item + lien vers le catalogue

- [x] Migration `005` : colonne `item_types.display_order INT NOT NULL DEFAULT 100` (petit = en premier) ;
      valeurs pour les types des seeds (Series, Manga, Comic… avant Episode, Volume…).
- [x] API : champ `display_order` (lecture / écriture, export / import du catalogue) ; listes d'ItemTypes triées par
      `display_order` puis `name` ; `items_by_type` des statistiques dans le même ordre, avec `type_id`.
- [x] Éditeur de type : champ « Ordre d'affichage » dans l'identité.
- [x] Accueil, « Répartition par type » : ordre des types, chaque ligne mène au catalogue filtré
      (`/items?item_type_id=…`).
- [x] Sélecteurs de type (catalogue, dialogues) : même ordre (tri déjà fait par l'API).

## Lot E3 — Type de collection limité à des types d'item

- [x] Migration `006` : table de jonction `collection_type_item_types` (liste vide = tout est permis).
- [x] API : champ `item_type_ids` sur `CollectionType` (lecture / écriture) ; ajout d'un item non
      permis à une collection → 422 avec un code d'erreur dédié (traduit côté front).
- [x] Les items déjà présents et devenus non permis sont conservés (seuls les nouveaux ajouts sont refusés).
- [x] Éditeur de type de collection : choix des types d'item permis.
- [x] Dialogue « Ajouter un item » : ne propose que les types permis. Dialogue « Ajouter à une collection »
      (fiche d'item) : seules les collections dont le type permet l'item. Import de collection : même règle.

## Lot E4 — Images

- [x] **E4a — Doublons** : fichier nommé d'après le SHA-256 de son contenu (`<hash>.jpg`) ; un envoi
      identique renvoie la même URL sans réécrire. Les fichiers existants gardent leur nom.
- [x] **E4b — Purge** (admin, page « Images », `POST /uploads/purge?dry_run=`) : recherche des fichiers de `UPLOAD_DIR` qu'aucun attribut (items,
      items de collection) ne référence plus ; délai de grâce de 24 h ; simulation (liste, taille)
      puis confirmation, comme l'import.
- [x] **E4c — Contrôle à l'envoi**, par catégorie de `x-image-categories` :
      `tolerance` (écart de ratio permis, en %), `strict` (bloquer ou seulement avertir),
      `min_width` / `min_height` (pixels). Contrôle dans le formulaire (ImageGalleryField) ;
      extension gérée par `DynamicForm`, `SchemaBuilder`, `lib/schema.ts`, documentée (README,
      `Collectionnard.md` § 3). Éditeur extrait dans `SchemaImageCategories.vue`, contrôle pur `checkImage` (`lib/schema.ts`).

## Lot E5 — Mobile hors ligne (lecture seule)

Besoin : en magasin sans réseau, vérifier si l'on possède déjà un item (titre, numéro, ISBN).
Lecture seule ; l'instance est servie en HTTPS (le service worker l'exige).

- [x] **E5a — Application installable** : `vite-plugin-pwa` (MIT, Workbox) ; manifeste (nom, icônes
      dérivées de `favicon.svg`, couleurs « Étiquette »), service worker qui garde l'application
      (JS, CSS, polices) : la PWA s'ouvre sans réseau. Mise à jour proposée quand une nouvelle
      version est publiée. Rien de l'API n'est mis en cache par le service worker à ce stade.
- [x] **E5b — Copie locale des collections** : IndexedDB, par utilisateur. Contenu : collections,
      items (libellé, type, attributs), attributs contextuels, schémas utiles à l'affichage (types
      d'item et de collection). Source : l'export JSON des collections (déjà existant) ou un endpoint
      dédié `GET /me/offline` si l'export ne suffit pas. Rafraîchie à l'ouverture et au retour du
      réseau ; `navigator.storage.persist()` demandé (Safari efface sinon après quelques semaines) ;
      effacée à la déconnexion.
- [x] **E5c — Mode hors ligne** : détection réseau ; « Mes collections », le détail d'une collection
      et une recherche (titre, numéro, ISBN…) servis par la copie locale ; bandeau
      « Hors ligne — données du … » ; actions d'écriture masquées.
- [x] **E5d — Couvertures, espace borné** :
      - vignettes côté serveur (`/api/v1/uploads/<nom>?w=240`, JPEG redimensionné, mis en cache sur
        disque) : ~10–20 Ko au lieu de 100–300 Ko pour l'original ;
      - seules les vignettes des items de mes collections sont gardées hors ligne ;
      - plafond réglable (défaut 50 Mo, soit ~3 000 couvertures) : au-delà, les plus anciennement
        consultées sont évincées ; option « sans couvertures » ;
      - espace utilisé affiché (`navigator.storage.estimate()`) avec un bouton « Vider ».
- Plus tard (hors lot) : écritures hors ligne (file d'attente + synchronisation), catalogue complet
  hors ligne, scan de code-barres (Phase 5) sur la copie locale.
