# Phase 5 — Mobile : appareil photo et scanner : conception

Raffinage du 2026-10-25, branche `phase5-mobile`, un commit par lot. Les lots et leurs cases à cocher sont dans
`Collectionnard.md` § Phase 5 ; ce document porte les cas d'usage, les décisions et les détails techniques.

## Décisions du mainteneur

| Sujet            | Décision |
| ---------------- | -------- |
| Support          | **L'application installable (PWA) existante**, pas d'application native : l'appareil photo et le scanner passent par les API du navigateur (`<input capture>`, `getUserMedia`, `BarcodeDetector`). Rien à publier sur un store, une seule base de code. |
| Recadrage        | **cropperjs 2** (MIT, composants web, gestes tactiles) ; cadre contraint au ratio de la catégorie (`x-image-categories`), rotation par quarts de tour, image déplaçable et zoomable. Pas de correction de perspective (voir « Plus tard »). |
| Lecture des codes | **`BarcodeDetector` natif** quand le navigateur l'offre (Chrome Android), sinon le ponyfill **barcode-detector** (MIT) sur **zxing-wasm** (MIT). Le `.wasm` (≈ 0,9 Mo) est servi par l'application elle-même (jamais depuis un CDN) et précaché par le service worker : le scanner marche aussi hors ligne. |
| Taille des photos | Réduites dans le navigateur à **2000 px** de côté (JPEG qualité 0,9 ; le PNG garde sa transparence) : une couverture n'a pas besoin de 12 Mpx, et `MAX_UPLOAD_SIZE` (10 Mo) n'est plus atteint. |
| Où scanner       | Partout où l'on saisit ou cherche un code : recherche du catalogue, ajout à une collection, consultation hors ligne, champ `x-identifier` d'un formulaire, recherche d'éditions dans les sources externes ; plus une page « Scanner » dédiée au geste « je suis en boutique, est-ce que je l'ai déjà ? ». |
| HTTPS            | L'accès à l'appareil photo exige un **contexte sécurisé** (HTTPS, ou `localhost` en développement) : déjà requis par l'application installable (README « Mise en production »). |

## État de départ

- `ImageGalleryField` (widget `image-gallery`) : un `<input type="file">` par catégorie ; sur mobile le navigateur
  propose déjà l'appareil photo, mais la photo part telle quelle (plusieurs Mo, cadrage approximatif) et une catégorie
  `strict` la **refuse** si le ratio ne colle pas.
- Les identifiants scannables sont stockés **en chiffres seuls** : `isbn` (`^97[89][0-9]{10}$`), `ean` et `barcode`
  (`^[0-9]{8,14}$`), tous `x-identifier`. La recherche plein texte de `GET /items` (`search`) indexe ces chaînes :
  un code scanné la trouve tel quel, sans nouvel endpoint ; la recherche hors ligne (`lib/offlineSearch.ts`) aussi.
- `POST /items/duplicates` signale un item du même type portant déjà l'identifiant (P4-3b) : utile après un scan à la
  création.
- Les sources externes cherchent une édition par ISBN (BnF) ou code-barres (MusicBrainz) : `ExternalEditionSearch`,
  mode « code ».

## Cas d'usage

| Cas | Lot | Détail |
| --- | --- | ------ |
| Photographier une couverture et la recadrer | **M1** | Tuile « Photo » (écrans tactiles, `capture="environment"`) → dialogue de recadrage au ratio de la catégorie → réduction → envoi. Un fichier hors format n'est plus refusé : le recadrage est proposé. |
| Chercher un item du catalogue par son code | **M2** | Bouton scanner dans la barre de recherche du catalogue (`?search=<code>`), dans le dialogue d'ajout à une collection et dans la consultation hors ligne. |
| Enregistrer un code à la création d'un item | **M2** | Bouton scanner dans tout champ `x-identifier` de `DynamicForm` (création, modification, proposition, correction d'un nœud externe). Les doublons restent signalés. |
| Importer depuis les sources externes par le code | **M2** | Bouton scanner dans `ExternalEditionSearch` (mode code : ISBN ou code-barres selon le connecteur). |
| Redresser une couverture photographiée de biais | **M4** | Étape « Redresser » avant le recadrage : bords détectés automatiquement (fond contrasté), quatre coins ajustables, homographie en JavaScript pur (pas d'OpenCV). D'office pour une photo, sur demande pour un fichier. |
| Photographier l'état d'un exemplaire | **M5** | Propriété `photos` (galerie, catégorie `condition`, ratio 3/4) du type de collection « Possession » (paquet `collections` v3) ; vignettes dans la liste d'une collection, la fiche d'un item et les pages publiques. |
| « Est-ce que je l'ai déjà ? » | **M3** | Page `/scan` : caméra permanente, chaque code lu interroge le catalogue ; résultat avec « Dans N de mes collections », ouverture de la fiche, ajout à une collection ; sinon raccourcis vers les sources externes (code prérempli) ou la création. Mode continu : les codes lus s'empilent dans une liste pour inventorier une pile de livres. |

### Plus tard (tracés, non planifiés)

- **Recadrer une image déjà dans la galerie** (re-téléchargement puis nouvel envoi) : simple à ajouter sur M1.
- **OCR** du titre ou du dos (Tesseract.js, Apache 2.0, ≈ 2 Mo + données de langue) pour préremplir la recherche quand il
  n'y a pas de code-barres (BD anciennes, vinyles).
- **QR code des liens de partage** (P4-2) affiché sur la page de partage, pour passer une collection d'un écran à
  l'autre (lib `qrcode`, MIT, ≈ 20 ko).
- **Scan d'étagère** : plusieurs codes dans une même image (`BarcodeDetector.detect` en rend déjà plusieurs) ;
  dépend du mode continu de M3.

## M1 — Photo de couverture et recadrage

- `lib/images.ts` : `imageSize`, `fitWithin`, `outputType` / `outputName`, `encodeImage` (canvas → Blob, fond blanc
  sous un JPEG), `shrinkIfNeeded` (photo gardée telle quelle mais réduite à `MAX_EDGE`).
- `components/common/ImageCropDialog.vue` : cropperjs instancié à l'ouverture (gabarit maison : pas de création de cadre
  par glissement, le fond déplace l'image ; huit poignées ; grille et croisée), cadre initial = plus grande zone de
  l'image au ratio, « Ajuster », rotation ±90°, « Sans recadrage », « Recadrer » (`$toCanvas()` à la résolution
  d'origine puis `encodeImage`). Le dialogue est `append-to-body` (il s'ouvre au-dessus du dialogue d'édition).
- `ImageGalleryField.vue` : tuile « Photo » quand `(pointer: coarse)` ; une photo passe toujours par le recadrage ;
  un fichier dont les contrôles bloquent (`strict`) ouvre le recadrage avec le motif, au lieu d'être refusé. Après
  recadrage, les contrôles sont rejoués (la taille minimale peut encore manquer) puis l'envoi a lieu.
- Pas de nouvelle extension `x-*` : tout vient de `x-image-categories` (ratio).

## M2 — Scanner de codes-barres

- `lib/barcode.ts` : `supportedFormats` (EAN-13, EAN-8, UPC-A, UPC-E, Code 128, Code 39, ITF, QR), `createDetector()`
  (natif si `BarcodeDetector` existe et sait lire l'EAN-13, sinon import dynamique du ponyfill avec le `.wasm` local
  via `prepareZXingModule({ overrides: { locateFile } })`), `normalizeCode` (UPC-A à 12 chiffres rendu tel quel ;
  le front ne reformate jamais un identifiant).
- `components/common/BarcodeScannerDialog.vue` : `getUserMedia` (caméra arrière, 1280 px), aperçu `<video playsinline>`,
  boucle de détection (`requestVideoFrameCallback` sinon toutes les 150 ms), lampe torche si la piste l'offre
  (`torch`), arrêt des pistes à la fermeture ; sans caméra ou accès refusé : repli sur un `<input capture>` et détection
  dans la photo. Émet `detected(code, format)` et se ferme (M3 ajoute le mode continu).
- `components/common/ScanButton.vue` : bouton icône `pi-barcode` qui ouvre le dialogue, utilisé par les points
  d'intégration ci-dessus. Visible partout (une webcam convient aussi) ; le dialogue explique quand rien n'est
  disponible.
- `vite.config.ts` : `**/*.wasm` précaché par Workbox (≈ 0,9 Mo, sous la limite par défaut de 2 Mio).
- nginx (`web/nginx-security-headers.conf`) : `Permissions-Policy: camera=(self)` (était `camera=()`, ce qui
  interdisait la caméra), CSP `script-src 'self' 'wasm-unsafe-eval'` (sans quoi Chrome refuse de compiler le wasm),
  `img-src blob:` déjà présent pour l'aperçu du recadrage ; `gzip_types application/wasm`.

## M3 — Page « Scanner »

- `views/ScanView.vue` (`/scan`, liens « Scanner » dans le menu latéral et la feuille « Plus ») : `BarcodeScannerDialog`
  en mode `continuous` ; chaque code lu devient une entrée (le plus récent en premier ; un code déjà lu remonte sans
  nouvelle requête) cherchée par `GET /items?search=<code>` puis, pour chaque item trouvé, `GET /items/{id}/collections`
  (« Dans N de mes collections » / « Dans aucune de mes collections »). Le dernier résultat s'affiche aussi sous
  l'aperçu de la caméra (slot par défaut du dialogue) : on enchaîne les codes sans quitter le scanner.
- Actions : ouvrir la fiche, « Ajouter à une collection » (`AddToCollectionDialog`, collections déjà présentes
  exclues) ; code inconnu : « Chercher dans les sources externes » (`/admin/external-import?code=…`, la recherche
  d'édition se lance d'office en mode code, `?connector=` facultatif) ou « Créer l'item » / « Proposer l'item »
  (`ItemEditDialog` avec `prefill-code` : au choix du type, les propriétés `x-identifier` de premier niveau dont le
  `pattern` accepte le code sont préremplies — `lib/barcode.ts` `identifierPrefill`, rien de métier).
- Après création ou proposition, le code est recherché à nouveau pour refléter le catalogue.

## M4 — Redressement de la perspective

- `lib/perspective.ts` (JavaScript pur, sans OpenCV ni WebAssembly supplémentaire, testé sur des images de synthèse) :
  `detectQuad` cherche sur une copie de 320 px l'objet le plus grand qui se distingue du fond (couleur médiane de la
  bordure, seuil d'Otsu sur l'écart RVB, plus grande composante connexe), prend les quatre coins extrêmes de son
  enveloppe convexe et valide la forme (convexe, côtés francs, ≥ 85 % de l'enveloppe, ≥ 70 % de remplissage) ; sinon
  null (fond uniforme, couverture pleine image, objet trop petit, disque). `homography` (huit équations, Gauss à pivot
  partiel) et `warpPerspective` (correspondance inverse, bilinéaire, blanc hors champ) redressent le quadrilatère vers
  un rectangle de la taille moyenne de ses côtés, borné à 2000 px (source ramenée à 2500 px).
- `components/common/PerspectiveEditor.vue` : aperçu + SVG par-dessus (polygone, quatre poignées déplaçables au
  pointeur ou aux flèches, 1 % par appui, 10 % avec Maj) ; « Détecter les bords », « Image entière », « Passer »,
  « Redresser » → fichier redressé rendu au dialogue.
- `ImageCropDialog.vue` : étape `straighten` avant `crop` (prop `straighten`, posée par `ImageGalleryField` pour une
  photo prise à l'instant) ; bouton « Redresser » dans la barre du recadrage pour un fichier ; le recadrage au ratio
  porte sur l'image redressée. Limites connues : le fond doit contraster avec la couverture et celle-ci ne doit pas
  toucher les bords du cadre ; sinon les coins se posent à la main.

## M5 — Photos de l'état d'un exemplaire

- Paquet `collections` **v3** : le `CollectionType` « Possession » (`Ownership`) gagne `photos` (`image-gallery`,
  catégorie `condition`, ratio 3/4, sans contrôle de tolérance : cadre 3/4 au recadrage, photo libre sinon), libellés
  fr / en, en dernier dans `x-order`. Mise à jour par fusion (page « Domaines » ou `install-domain collections`) : les
  instances modifiées localement gardent leurs champs, `photos` s'ajoute. Pas `x-private` : le propriétaire peut le
  cacher par collection (`hidden_fields`).
- Affichage : `lib/schema.ts` `galleryUrls(schema, attrs, key?)` ; `components/common/GalleryThumbs.vue` (vignettes
  `?w=240` cliquables, « +n ») dans `CollectionItemList` (colonne par galerie sur bureau, ligne de vignettes sur
  mobile ; donc aussi pages publiques et liens de partage) et `ItemCollectionsCard` (onglet « Dans vos collections »).
- Rien de métier : tout `CollectionType` portant une propriété `image-gallery` a le même rendu ; l'envoi d'images est
  déjà ouvert à tout compte connecté et la purge garde les images référencées par `collection_items`.
