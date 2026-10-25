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
| « Est-ce que je l'ai déjà ? » | **M3** | Page `/scan` : caméra permanente, chaque code lu interroge le catalogue ; résultat avec « Dans N de mes collections », ouverture de la fiche, ajout à une collection ; sinon raccourcis vers les sources externes (code prérempli) ou la création. Mode continu : les codes lus s'empilent dans une liste pour inventorier une pile de livres. |

### Plus tard (tracés, non planifiés)

- **Détection automatique des bords** d'une couverture photographiée de biais (correction de perspective) : demande
  OpenCV.js (≈ 8 Mo) ou un modèle maison ; écarté tant que le recadrage manuel suffit.
- **Recadrer une image déjà dans la galerie** (re-téléchargement puis nouvel envoi) : simple à ajouter sur M1.
- **OCR** du titre ou du dos (Tesseract.js, Apache 2.0, ≈ 2 Mo + données de langue) pour préremplir la recherche quand il
  n'y a pas de code-barres (BD anciennes, vinyles).
- **QR code des liens de partage** (P4-2) affiché sur la page de partage, pour passer une collection d'un écran à
  l'autre (lib `qrcode`, MIT, ≈ 20 ko).
- **Photo de l'état d'un exemplaire** : déjà possible sans code, il suffit à un `CollectionType` d'avoir une propriété
  `image-gallery` (l'envoi est ouvert à tout utilisateur connecté) ; à documenter dans un paquet de domaine.
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
- `vite.config.ts` : `**/*.wasm` précaché (Workbox, limite de taille relevée à 3 Mio).
