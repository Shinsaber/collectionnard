# Phase 7 — Exemplaires, amis, prêts et cadeaux : conception

Ouverte le 2026-11-07. Ce document porte les décisions, le modèle et l'API visés ;
les lots et leurs cases à cocher sont reportés dans [`docs/conception.md`](../conception.md), Phase 7 (questions tranchées
le 2026-11-07, voir la fin du document). Les échanges restent dans [`docs/idées.md`](../idées.md) : les lots ci-dessous les préparent.

**Cible** : une instance publique à terme. Les prêts et les cadeaux mettent en jeu des personnes réelles
(un nom libre, un membre) : rien de ce qu'ils écrivent ne doit fuiter vers un lecteur qui n'a pas à le voir.

## Décisions du mainteneur

| Sujet               | Décision                                                                                                                                                                                                        |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Multiple possession | **Une ligne par lot** : un lot regroupe des exemplaires identiques, avec une quantité (1 par défaut). Un état ou une finition différents (carte brillante) font un autre lot.                                   |
| Amis                | **Relation réciproque** (demande, puis acceptation) et nouveau niveau de visibilité `friends`. Elle resservira aux prêts, aux cadeaux et aux échanges.                                                          |
| Prêts et cadeaux    | **Deux tables distinctes**, la visibilité étant opposée (le prêt est le drapeau du propriétaire, la réservation lui est cachée). On mutualise le code, pas la table.                                            |
| Rien en dur         | « Possession » et « Liste de souhaits » ne sont que des `CollectionType` du paquet `collections` : prêts et réservations sont ouverts par des extensions à la racine du type.                                   |
| Quantité            | **Sur tous les types de collection**, sans extension : la quantité est une colonne du lot, pas un attribut du schéma.                                                                                           |
| Ajout d'un doublon  | **Toujours demander** : ajouter un item déjà présent ouvre un choix (augmenter un lot existant ou créer un lot).                                                                                                |
| Prix                | `purchase_price` est **unitaire** ; Possession gagne **`estimated_value`** (valeur estimée unitaire, utile aux échanges), `x-private` comme le prix d'achat. Paquet `collections` v5.                           |
| API                 | **Rupture directe** : les routes adressent le lot ; le front, seul client, est livré avec l'API. À signaler dans les notes de version.                                                                          |
| Portée de `friends` | **Collections, profil et avis**, partout où un réglage de visibilité existe déjà.                                                                                                                               |
| Emprunteur          | Un **ami** (relation acceptée) ou **un simple nom** pour n'importe qui. L'ami est **notifié**, sans acceptation ; le rappel de retard va **aux deux**. Drapeau vu du propriétaire et de l'emprunteur seulement. |
| Réservation         | **Amis seulement** ; les autres offrants voient **qui a réservé** (« Réservé par Camille »), pour se coordonner. Jamais montré au propriétaire.                                                                 |
| Notifications       | Dans l'application seulement (cloche de P4-3c) : pas d'e-mail, faute de SMTP.                                                                                                                                   |
| Limite acceptée     | Un second compte du propriétaire peut voir les réservations de sa propre liste : impossible à empêcher.                                                                                                         |
| Reporté             | **Échanges** (« 3 de ces 5 à échanger » est un état de lot, croisé avec les listes de souhaits des autres, messagerie) : à affiner après P7-4.                                                                  |

## État de départ

- `collection_items` porte `UNIQUE (collection_id, item_id)` (`001_init.sql`) : les attributs du `CollectionType` (état,
  prix, photos d'exemplaire de M5) décrivent un seul exemplaire.
- Les routes adressent par item : `PUT/DELETE /collections/{collectionId}/items/{itemId}` ; « Dans N de mes collections »
  (`GET /items/{id}/collections`) suppose une ligne par collection.
- Paquet `collections` v4 : `Ownership`, `Watching`, `Wishlist`.
- Visibilité P4-2 : `private` | `members` | `public`, filtre `internal/privacy` (`CanView`, `Levels`), liens de partage
  (`share_links`, migration 009), champs `x-private` et `hidden_fields` retirés par le serveur.
- Tâche périodique : `RunUploadPurge` (`handler/upload_purge.go`), sur un `time.Ticker` lancé par `main.go`.
- Blocage (I-4) : `user_blocks`, `IsBlocked`, `Blockers`, règle « anonymat d'abord ».
- Dernière migration : `019_recommendations.sql`.

## P7-1 — Exemplaires multiples (socle)

**Livré le 2026-11-08.** Prérequis de tous les lots suivants. Écarts et précisions par rapport à ce qui était prévu :

- `quantity` bornée de 1 à 100 000 (`model.MaxQuantity`, `CHECK` de la migration 020) ; hors bornes : 422
  `invalid_quantity`. Un `PUT` sans `quantity` garde celle du lot.
- `item_count` d'une collection compte les **items différents** ; nouveau `copy_count` (somme des quantités). L'en-tête
  d'une collection affiche « 1 item · 4 exemplaires » quand ils diffèrent.
- Scission : le nouveau lot garde la date d'ajout du lot d'origine (statistiques par mois inchangées) et copie ses
  attributs si aucun n'est donné. Fusion : 422 `lots_not_mergeable` (lot lui-même, autre item) ; les photos des
  galeries du lot fusionné (`x-widget: image-gallery` du CollectionType) rejoignent celles du lot qui reçoit, catégorie
  par catégorie et sans doublon ; ses autres attributs sont perdus (demande du mainteneur du 2026-11-08).
- Liste d'une collection (demande du 2026-11-08) : recherche plein texte, type d'item, « Plusieurs exemplaires »
  (`multiple=true`) ; tri par date d'ajout, quantité, pertinence ou attribut du type choisi (`attr.<clé>`, filtres
  `attr.*` aussi acceptés par l'API), mêmes règles que le catalogue (`parseItemSearch`, `buildItemQuery`). À valeur
  égale, les lots d'un même item se suivent. La colonne « Qté » n'apparaît que si un lot de la page compte plusieurs
  exemplaires.
- Export : `id` (lot) et `quantity` en JSON, colonnes `lot_id` et `quantity` en CSV, format inchangé (version 1).
  Import : un lot est rapproché par son identifiant, ou, pour un fichier antérieur, par item (le plus ancien lot non
  encore rapproché) ; un même identifiant deux fois est une erreur, un même item deux fois fait deux lots.
- Statistiques pondérées par la quantité : sommes, moyennes et répartitions par exemplaire, ajouts par mois,
  exemplaires par type (`total_items` = exemplaires). La complétude compte toujours des items distincts.
- Front : `lib/lots.ts` (lots d'une collection, résumé, `addCopies`), `LotTargetPicker` (choix du lot à l'ajout, dans
  `AddToCollectionDialog` et `CollectionAddItemDialog` ; bouton « Ajouter » grisé tant que rien n'est choisi),
  `LotEditDialog`, `LotSplitDialog`, `LotMergeDialog`, `LotActionsMenu` ; colonne « Qté » et pastille « 3 exemplaires »
  sur mobile ; « Dans N de mes collections » compte les collections distinctes. Une copie hors ligne antérieure
  (sans lot) reste lisible.

### Modèle

- Migration `020_collection_lots.sql` : suppression de `UNIQUE (collection_id, item_id)`, ajout de
  `quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity BETWEEN 1 AND 100000)`. Les lignes existantes deviennent des lots de 1 : aucune
  reprise, `quantity = 1` correspond exactement au comportement actuel.
- Un lot reste un `CollectionItem` : ses attributs (état, prix, photos) valent pour chacun de ses exemplaires.
- Paquet `collections` v5 : `estimated_value` (nombre, unitaire, `x-private`) sur Possession ; `purchase_price`
  documenté comme unitaire. Aucune extension pour la quantité : elle vaut pour tous les types.

### API

- Les routes d'un lot passent à son identifiant : `PUT/DELETE /collections/{collectionId}/lots/{lotId}`, sans
  route de transition. `POST /collections/{id}/items` crée toujours un nouveau lot (`quantity` facultative) ;
  augmenter un lot existant, c'est un `PUT` de sa quantité. Le choix est fait par le front : si l'item est déjà dans
  la collection, une boîte de dialogue liste ses lots (quantité, état) et propose « +1 sur ce lot » ou « nouveau lot ».
- Scission : `POST …/lots/{lotId}/split` (`{quantity, attributes}`) détache N exemplaires vers un nouveau lot (une
  carte abîmée sort d'un lot de 5). Fusion : `POST …/lots/{lotId}/merge` (`{into}`), seulement entre lots du même item.
- `GET /items/{id}/collections` renvoie les lots (avec leur quantité), pas seulement les collections.

### À revoir

- **Statistiques** (`store/stats.go`, `internal/collstats`) : nombre d'exemplaires = somme des quantités ; la
  complétude (« 30 tomes sur 42 ») compte toujours des items distincts ; dépense = somme des `purchase_price` ×
  quantité, valeur estimée = somme des `estimated_value` × quantité (deux montants privés, comme aujourd'hui).
- **Import / export** : colonne `quantity` ; un ancien export reste importable (quantité 1) ; à l'import, deux lignes
  du même item ne sont plus un doublon.
- **Copie hors ligne** (`offlineDb.ts`, `OfflineSnapshot`) : clé par lot, plus par item.
- **Front** : `CollectionItemList` (quantité affichée, « ×5 »), formulaire d'ajout, scission et fusion, fiche d'item
  (« Dans N de mes collections » devient « 7 exemplaires dans 2 collections »).
- **Historique et propositions** (P4-3) : rien à changer, ils portent sur l'item.

### P7-1b — Types d'item acceptés et tri par défaut d'une collection

**Retenu et livré le 2026-11-08** (piste du mainteneur pour la question 4). Écarts : les types acceptés se choisissent
dans une liste (`CollectionSettingsFields`, à la création et dans `CollectionEditDialog`, extrait de la page) ; sans
`sort`, l'API applique le tri par défaut, le front envoie donc toujours le sien ; un tri par défaut devenu sans objet
quand on change les types acceptés est remis à « Ajoutés récemment » dans le dialogue. Aides : `lib/collections.ts`
(`acceptedTypeIds`, `specializedTypeId`, `lotSortOptions`) ; Go : `Collection.AcceptedItemTypes` / `SpecializedType`,
`checkCollectionSettings`, `applyDefaultSort`. Seules les propriétés scalaires se trient : un titre multilingue
(`i18n-text`) ne figure pas parmi les tris.

Le propriétaire précise, à la création de sa collection ou dans « Modifier », les **types d'item qu'elle accepte**, et
l'ordre dans lequel ses lots s'affichent à l'ouverture. Rien n'est figé dans les paquets : « Mes mangas » est une
Possession ordinaire, resserrée par son propriétaire sur les tomes, sans type de collection dédié ni intervention d'un
admin.

**Deux niveaux.** Le type de collection fixe la limite (`collection_type_item_types`, choisi par l'admin) ; la
collection la resserre (`collection_item_types`, choisie par son propriétaire). Les types acceptés sont ceux que les
deux permettent ; une collection sans liste propre accepte tout ce que permet son type.

| Collection      | Définie par                                                      | Tris proposés                                                                                                                      |
| --------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| **Générique**   | Accepte plusieurs types d'item, ou tous.                         | Tris communs à tous les items : date d'ajout (récente ou ancienne), quantité.                                                      |
| **Spécialisée** | N'accepte qu'un type d'item (une collection de tomes, de cartes). | Tris communs, plus les attributs triables de ce type (`attr.<clé>` : numéro, titre, date de parution…), sans avoir à filtrer par type. |

La nature dépend de ce que la collection accepte, pas de son contenu du moment : une collection générique qui ne contient
qu'un type d'item reste générique, et ses options ne changent pas au premier ajout d'un autre type.

Esquisse :

- Migration `021` : table `collection_item_types (collection_id, item_type_id)`, sur le modèle de
  `collection_type_item_types` (suppression en cascade avec la collection ou le type d'item), et colonne
  `collections.default_sort TEXT` (vide = défaut actuel, ajout récent).
- `Collection` gagne `item_type_ids` (liste propre, vide = celle du type) et `default_sort`, écrits par
  `POST /collections` et `PUT /collections/{id}`. Un type d'item que le type de collection ne permet pas est refusé
  (422 `item_type_not_allowed`).
- Ajout d'un item (`POST /collections/{id}/items`, import) : contrôlé contre les types acceptés, comme aujourd'hui contre
  ceux du type de collection. Seuls les nouveaux ajouts sont contrôlés : resserrer une collection garde ce qu'elle
  contient déjà.
- `default_sort` est vérifié comme le paramètre `sort` de la liste ; un tri `attr.<clé>` n'est permis qu'à une
  collection spécialisée, contre le schéma de son type d'item (422 sinon). `GET /collections/{id}/items` sans `sort`
  l'applique, pages publiques et liens de partage compris. Si l'attribut n'existe plus (type d'item modifié, ou types
  acceptés changés depuis), la liste revient sans erreur au tri par date d'ajout ; le dialogue « Modifier » le signale.
  Dans un tri par attribut, les items d'un autre type (gardés après un resserrement) passent en fin de liste.
- Front : champ facultatif « Types d'item acceptés » (liste, vide = tous ceux du type) et « Tri par défaut » à la
  création et dans « Modifier » ; le dialogue d'ajout ne cherche que parmi les types acceptés ; la barre de tri
  (`CollectionItemsToolbar`) propose les attributs d'une collection spécialisée sans qu'un type soit choisi, et son
  filtre de type se limite aux types acceptés.
- Export et import d'une collection : la liste des types acceptés et le tri par défaut voyagent avec elle ; à l'import,
  un type d'item inconnu de l'instance est ignoré. La copie hors ligne suit la collection.

Écarté : **le tri par libellé** (`x-label`). Le libellé est calculé en Go et en TypeScript, dépend de la langue, et
diffère d'un type à l'autre. Le trier en SQL obligerait à stocker un libellé calculé par item et par langue, à tenir à
jour à chaque écriture d'un item, et à tout recalculer à chaque modification d'un `x-label` sur un ItemType. C'est un
coût permanent, et une source d'écarts, pour un projet dont les types sont faits pour évoluer.

Écarté aussi : des **types de collection spécialisés dans les paquets** (« Possession de tomes »…), un par domaine et
par usage, ou créés par l'admin à la demande.

Questions tranchées le 2026-11-08 : le tri par défaut s'applique aux pages publiques et aux liens de partage ; il se
règle dans le dialogue « Modifier » ; une collection générique qui ne contient qu'un type d'item reste générique ; les
types acceptés se choisissent sur la collection, par son propriétaire.

## P7-2 — Amis

**Coût : moyen.** Touche toutes les lectures filtrées de P4-2.

### Modèle

- Migration : `friendships (id, requester_id, addressee_id, status pending|accepted, created_at, accepted_at)`,
  `UNIQUE` sur la paire quel que soit le sens, `CHECK (requester_id <> addressee_id)`.
- Bloquer un membre (I-4) rompt l'amitié et retire les demandes en cours ; un membre bloqué ne peut pas faire de
  demande (refus muet, comme les recommandations).
- Quota de demandes par jour (`limitedIn`, `rateDay`).

### Visibilité `friends`

- Nouveau niveau entre `private` et `members` : `private` < `friends` < `members` < `public`.
- `privacy.CanView` et `privacy.Levels` deviennent dépendants des amis du lecteur : `Levels(viewer)` ne suffit plus, il
  faut un fragment SQL (`friendsOf`) sur le modèle de `notBlockedBy`.
- `VisibilitySelect` propose le niveau sur les collections, le profil et les avis. Un avis `friends` est signé pour
  les amis ; pour les autres, il compte dans la moyenne comme un avis `private` (note seule).
- Un lien de partage reste indépendant : il ouvre la collection à qui le détient, ami ou non.

### API

- `GET /me/friends`, `GET /me/friend-requests`, `POST /me/friends` (`{username}`), `POST /me/friend-requests/{id}/accept`,
  `DELETE /me/friends/{id}` (refus, annulation ou rupture).
- Notifications `friend_request`, `friend_accepted`.

### Livré (2026-11-08)

Migrations `022_friendships.sql` (table, unicité de la paire par `LEAST` / `GREATEST`, niveau `friends` ajouté aux
contraintes des collections, profils et avis) et `023_friend_links.sql`. Choix faits en cours de lot :

- **Deux façons d'inviter** (piste 2, retenue par le mainteneur le 2026-11-08). Une première version invitait par
  nom exact quelle que soit la visibilité du profil : la réponse (404 ou 201) laissait deviner les noms d'utilisateur.
  Désormais :
  - **par nom, un membre dont je vois le profil** (annuaire, avis, commentaires ; bouton « Inviter en ami » sur son
    profil) ; tout autre nom répond 404, comme un nom inconnu ;
  - **par lien d'ami**, pour un profil privé : un lien secret (jeton aléatoire, seul son hash stocké, comme les
    invitations de comptes), à usage unique, valable 7 jours, 10 valides au plus, 20 créations par jour. Je le
    transmets moi-même (copie, ou feuille de partage du téléphone) ; qui l'ouvre voit « Camille vous propose d'être
    amis » et devient ami en acceptant (une demande en attente entre nous est acceptée). Le lien d'un membre qui m'a
    bloqué répond 404, comme un lien expiré. Ouvert sans session, il ramène au lien après la connexion
    (`/login?redirect=`, chemin interne seulement).
- Une demande envoyée à un profil devenu invisible depuis ne montre que le nom saisi tant qu'elle n'est pas acceptée.
- **Demande croisée** : inviter un membre qui m'a déjà invité vaut acceptation (`friend_accepted`).
- **Refus muet** : la demande d'un membre que le destinataire a bloqué est enregistrée mais reste cachée de lui, jamais
  notifiée ni acceptable ; le demandeur la voit « en attente », comme une demande ignorée. Inviter un membre que j'ai
  moi-même bloqué répond 409 `member_blocked`. 20 demandes par jour.
- **Lecteur** : `privacy.Viewer` (compte + amis, chargé une fois par requête par `h.viewer`) remplace l'UUID dans
  `CanView` ; `Levels` reste sur le compte, `LevelsOf(viewer, owner)` ajoute `friends` pour une liste d'un seul
  propriétaire ; les listes multi-auteurs (avis d'un item, moyennes) signent un avis `friends` en SQL (`reviewSigned`,
  `friendOf`). Les lectures qui ne font que nommer (historique, articles) traitent le lecteur en inconnu si ses amis ne
  se chargent pas.
- **Annuaire** : il reste limité aux profils `members` et `public` ; un profil `friends` n'y figure pas.
- **Notification d'un commentaire** : l'émetteur n'y est nommé que si son profil est `members` ou `public`.
- **Page « Membres »** (`/members`, une seule entrée de menu) : onglet « Amis » (inviter, liens d'ami, demandes reçues
  et envoyées, liste des amis avec lien vers le profil quand il m'est ouvert, `profile_visible`, retrait confirmé) et
  onglet « Annuaire » (`?tab=directory`) ; ouverture d'un lien sur `/friend/:token`.

Restent en suspens : le défaut de visibilité d'un nouvel avis ne tient pas compte d'un profil `friends` (règle de la
Phase 6 inchangée : public si le profil est public, anonyme sinon) ; pas de QR code pour le lien d'ami (il faudrait
une bibliothèque de génération, la feuille de partage du téléphone suffit pour l'instant).

## P7-3 — Prêts

**Coût : moyen.**

### Modèle

- Extension `x-lendable` à la racine d'un `CollectionType` (`Ownership`).
- Table `loans (id, lot_id → collection_items ON DELETE CASCADE, quantity, borrower_id NULL → users, borrower_name NULL,
lent_at, due_at NULL, returned_at NULL, note, created_at)`, avec
  `CHECK ((borrower_id IS NULL) <> (borrower_name IS NULL))` : un membre **ou** un simple nom.
- Un prêt en cours n'emprunte pas plus que la quantité du lot (somme des prêts non rendus ≤ `quantity`, vérifiée dans la
  transaction).
- L'historique est la liste des prêts rendus : rien à archiver.

### Règles

- Seul le propriétaire du lot crée, modifie et clôt un prêt.
- Emprunteur membre : un **ami** seulement (amitié acceptée, vérifiée à la création) ; notification `loan_started`,
  sans acceptation. Pour toute autre personne, un simple nom. Rompre l'amitié ne touche pas les prêts en cours.
- Page « Mes emprunts » pour un emprunteur membre : ce qu'on lui a prêté, chez qui, retour prévu.
- Drapeau « prêté » sur le lot (« 2 sur 5 chez Camille ») : vu du **propriétaire et de l'emprunteur seulement** ;
  les autres lecteurs de la collection ne voient ni le prêt ni le nom.
- Supprimer un lot supprime ses prêts ; supprimer le compte d'un emprunteur garde le prêt avec son nom figé (à
  confirmer avec le RGPD).

### Rappels

Tâche périodique `RunReminders` sur le modèle de `RunUploadPurge`, partagée avec P7-4 : une fois par jour, elle notifie
le propriétaire et l'emprunteur membre d'un retour en retard, une seule fois par échéance (`reminded_at`).

### API

`GET/POST /collections/{collectionId}/lots/{lotId}/loans`, `PATCH /loans/{id}` (retour prévu, rendu le), `DELETE
/loans/{id}`, `GET /me/loans` (prêtés), `GET /me/borrowings` (empruntés).

### Livré (2026-11-08)

Migration `024_loans.sql`, paquet `collections` v6 (`x-lendable` sur Possession), case « Prêts possibles » dans
l'éditeur des types de collection. Écarts et choix faits en cours de lot :

- **`PUT /loans/{id}`** plutôt que `PATCH` : la convention du projet (PUT = remplacement des champs modifiables :
  quantité, prêté le, retour prévu, rendu le, note ; l'emprunteur ne change pas).
- **Nom figé toujours présent** : `borrower_name` est rempli aussi pour un ami (son nom affiché au moment du prêt), et
  `borrower_id` passe à `NULL` si le compte disparaît. Le `CHECK` « membre **ou** nom » du modèle visé est donc remplacé
  par « nom toujours, membre en plus ». La liste montre le nom actuel de l'ami tant que son compte est actif.
- **Un emprunteur nommé qui n'est pas un ami** (ou un nom inconnu) répond le même 422 `borrower_not_friend` : rien à
  deviner. Rompre l'amitié ne touche pas les prêts en cours.
- **Exemplaires d'un lot** : un lot ne descend pas sous ses exemplaires prêtés (quantité, scission : 422
  `invalid_quantity`) ; une fusion déplace les prêts du lot fusionné ; supprimer un lot supprime ses prêts.
- **Ce que voit chacun** : le propriétaire voit ses prêts sur le lot (drapeau « 2 prêtés · Camille », rouge en
  retard), dans « Prêts… » (en cours et historique) et sur la page « Prêts » ; l'ami voit, dans « Empruntés », l'item, le
  prêteur, les dates — ni la note ni le nom de la collection. Les autres lecteurs de la collection ne voient rien.
- **Rappels** : `RunReminders` tourne toutes les heures (une fois par échéance grâce à `reminded_at`, remis à zéro quand
  l'échéance change) et notifie le prêteur et l'ami (`loan_overdue`). Pas de réglage par variable d'environnement.
- **Dates** : champs de date natifs (comme `DynamicForm`), pas de `ElDatePicker`.

Restent en suspens : les prêts ne voyagent pas avec l'export / import d'une collection (données personnelles, à
trancher) ; l'ami ne voit pas le drapeau sur la page publique de la collection (seulement sa page « Empruntés ») ; pas
de rappel « bientôt à rendre » avant l'échéance.

## P7-4 — Cadeaux réservés

**Coût : petit à moyen.**

### Modèle

- Extension `x-reservable` à la racine d'un `CollectionType` (`Wishlist`).
- Table `gift_reservations (id, lot_id → collection_items ON DELETE CASCADE, giver_id → users, quantity, status
reserved|given|cancelled, note, created_at, updated_at)`.
- Si le lot a une quantité (« je veux 3 boosters »), plusieurs membres peuvent réserver chacun une part.

### Règles de visibilité (le piège)

- Les réservations ne sont **jamais** renvoyées au propriétaire : ni par l'API, ni par l'export, ni par la copie hors
  ligne, ni par les statistiques.
- Elles n'apparaissent **ni en lecture anonyme, ni par un lien de partage** : le propriétaire peut ouvrir sa liste
  déconnecté ou par son propre lien. Seul un membre connecté, autre que le propriétaire, les voit.
- Seuls les **amis** du propriétaire voient et posent des réservations (ce qui exclut d'office la lecture anonyme et
  les liens de partage). Ils voient qui a réservé et combien (« Réservé par Camille, 1 sur 3 »), pour se coordonner.
- Rompre l'amitié ou être bloqué retire l'accès aux réservations, sans les supprimer : l'offrant les retrouve dans
  « Mes cadeaux ».

### Rappels

Même tâche `RunReminders` : l'offrant est notifié quand le lot a quitté la liste (supprimé, ou déplacé dans une
collection de possession) ; la réservation passe alors en `given` ou reste à confirmer par l'offrant.

### API

`GET/POST /collections/{collectionId}/lots/{lotId}/reservations` (refusé au propriétaire : 404, pour ne rien révéler),
`PATCH/DELETE /reservations/{id}` (l'offrant seul), `GET /me/gifts` (ce que je compte offrir).

### Livré (2026-11-08)

Migration `025_gift_reservations.sql`, paquet `collections` v7 (`x-reservable` sur Liste d'envies), case « Cadeaux
réservables » dans l'éditeur des types de collection. Écarts et choix faits en cours de lot :

- **Pas de `GET` par lot** : les réservations sont jointes aux lots de `GET /public/collections/{id}/items` quand le
  lecteur est un ami du propriétaire (`can_reserve` sur la collection) ; jamais par un lien de partage, jamais pour
  le propriétaire ni un non-ami (vérifié par un test qui échoue si la table est seulement lue).
- **`PUT /reservations/{id}`** (convention du projet) ; **annuler supprime** la réservation : plus d'état `cancelled`,
  seulement `reserved` et `given`.
- **Le lot qui quitte la liste** : `lot_id` passe à `NULL` (au lieu d'une suppression en cascade) ; l'item et le
  propriétaire sont gardés sur la réservation. `RunReminders` prévient l'offrant une fois (`gift_left_list`) ; il la
  marque offerte ou l'annule depuis « À offrir ». Une fusion de lots déplace les réservations, comme les prêts.
- **Quantité** : les réservations d'un lot (offertes comprises) ne dépassent pas sa quantité au moment de réserver
  (422 `reservation_exceeds_lot`). Le propriétaire, lui, garde la main sur sa quantité : la borner trahirait des
  réservations qu'il ne doit pas voir. L'affichage peut donc montrer « 3 sur 2 » après une baisse.
- **Où ça se passe** : sur la page publique de la liste (`/c/:id`), un ami voit « Camille, vous · 2 sur 3 » et un
  bouton « Réserver » / « Ma réservation » (quantité, note à soi, annulation). La page « Prêts » devient « Prêts et
  cadeaux », avec un onglet « À offrir ».

- **Durée limitée** (migration `026_gift_reservation_expiry.sql`, choix du mainteneur le 2026-11-08) : une réservation
  non offerte vaut jusqu'à `expires_at` — 60 jours par défaut, de aujourd'hui à un an, réglable à la réservation et
  ensuite. Le serveur l'applique à chaque lecture : échue, elle ne compte plus dans le lot et sort de la vue des amis ;
  son offrant la garde dans « À offrir », marquée « Expirée », pour la prolonger (si les exemplaires sont libres) ou
  l'annuler. Une réservation offerte n'expire pas. Rappel `gift_expiring` 7 jours avant, une fois par échéance
  (`expiry_reminded_at`).

### P7-4b — Remise d'un cadeau

**Le défaut de P7-4** (relevé par le mainteneur le 2026-11-10) : « C'est offert » était le seul état qui bloquait des
exemplaires **sans échéance** tout en restant **invisible du propriétaire**. Une fausse manœuvre, ou un cadeau jamais
fait, bloquait sa liste sans qu'il puisse le voir ni le corriger. Le bouton mêlait deux moments : avant la remise (le
secret doit tenir ; la date limite y suffit, réglée sur l'occasion) et après (plus rien à cacher, et c'est le
propriétaire qui sait s'il a reçu le cadeau).

**Retenu le 2026-11-10** :

- **« Je l'ai remis »** remplace « C'est offert », avec une confirmation (« le propriétaire sera prévenu : la surprise
  est levée »). Avant ce clic, la réservation reste secrète et limitée dans le temps : aucun état ne bloque plus sans
  échéance en restant caché. Une fois remise, l'offrant ne la repasse plus en « réservé » (le propriétaire le sait) ;
  il peut seulement la retirer.
- **Le propriétaire est notifié** (`gift_given` : « Camille vous a offert « Breaking Bad » ») et voit sur le lot de sa
  liste « 1 offert par Camille ». Il ne voit que les réservations **remises**, jamais celles en cours.
- **« Bien reçu »** : les exemplaires quittent la liste (la quantité du lot baisse, ou le lot disparaît). Le
  propriétaire choisit de les **déplacer dans une autre de ses collections qui accepte l'item** (attributs du nouveau
  lot saisis au passage), ou de **simplement les retirer** s'il les a déjà ajoutés à la main. L'offrant est prévenu
  (`gift_received`) ; la réservation passe dans son historique (« Reçu le … »), hors de « À offrir » par défaut (case
  « Reçus compris »).
- **« Pas reçu »** : la réservation est **invalidée** (supprimée) ; l'offrant est prévenu (`gift_declined`).
- **Lot déjà retiré** : si le propriétaire a retiré l'item de sa liste avant la remise, « Je l'ai remis » clôt
  directement la réservation comme reçue (il n'y a plus de lot à confirmer) ; le propriétaire est notifié.
- Les autres amis voient « Camille (offert) » dans le compte d'un lot.
- **Pas de rappel** au propriétaire qui ne répond pas, dans un premier temps (`docs/idées.md`) : la réservation remise
  reste en attente, visible sur son lot.

**Livré (2026-11-10)** : migration `027_gift_handover.sql` (état `received`, `given_at`, `received_at`), routes
`POST /reservations/{id}/hand-over`, `POST /collections/{collectionId}/gifts/{id}/receive` et `…/decline`, `gifts` sur
les lots de la liste du propriétaire (seules les remises, lues par `GivenGifts`), `GET /me/gifts?status=open|received`.
Le `PUT` d'une réservation ne porte plus d'état (409 `gift_state` une fois remise). Choix faits en cours de lot :

- **Un lot supprimé vaut réception** de ses cadeaux remis (déclencheur `gift_lot_deleted`) : si le propriétaire retire
  l'item de sa liste au lieu de confirmer, la remise ne reste pas en attente sans lot. L'offrant n'en est pas notifié.
- **Collections proposées par « Bien reçu »** : toutes les autres collections du propriétaire qui acceptent l'item, celles
  qui se prêtent (une possession) en tête ; les attributs du nouveau lot se saisissent dans le dialogue.
- **Pas de choix du lot de destination** : « Bien reçu » crée toujours un nouveau lot, même si l'item est déjà dans la
  collection choisie (la règle « toujours demander » de P7-1 n'y est pas encore appliquée ; la fusion de lots le rattrape).
- La page d'une collection confie ses dialogues de lot à `LotDialogs` (au-delà de 400 lignes sinon).

## Briques mutualisées

| Brique                   | Prêt                              | Cadeau                              |
| ------------------------ | --------------------------------- | ----------------------------------- |
| Rattachement             | lot d'une collection `x-lendable` | lot d'une collection `x-reservable` |
| Contrepartie             | ami ou nom libre                  | toujours un ami                     |
| Période                  | prêté le, retour prévu, rendu le  | réservé, offert, annulé             |
| Rappels (`RunReminders`) | retour en retard                  | l'item a quitté la liste            |
| Drapeau sur le lot       | vu par le propriétaire            | vu par ses amis seulement           |

Code commun : la résolution de la contrepartie (ami ou nom libre, blocage), le contrôle de quantité sur
un lot, la tâche de rappels, le composant de drapeau.

## Découpage et ordre

1. **P7-1 — Exemplaires multiples** (socle, rupture d'API).
   **P7-1b — Types d'item acceptés et tri par défaut d'une collection.**
2. **P7-2 — Amis** (visibilité `friends`).
3. **P7-3 — Prêts**.
4. **P7-4 — Cadeaux réservés**.
5. Échanges : à affiner ensuite (`docs/idées.md`).

Un commit par lot, branche `phase7-exemplaires`.

## Questions tranchées le 2026-11-07

| #   | Lot  | Question                                         | Décision                                                                               |
| --- | ---- | ------------------------------------------------ | -------------------------------------------------------------------------------------- |
| Q1  | P7-1 | Prix d'un lot unitaire ou total ?                | Unitaire, et ajout d'une valeur estimée unitaire `estimated_value`, privée par défaut. |
| Q2  | P7-1 | Sur quels types la quantité se saisit-elle ?     | Tous, sans extension.                                                                  |
| Q3  | P7-1 | Ajouter un item déjà présent ?                   | Toujours demander : augmenter un lot ou en créer un.                                   |
| Q4  | P7-1 | Rupture d'API ou route de transition ?           | Rupture directe.                                                                       |
| Q5  | P7-2 | Où s'applique `friends` ?                        | Collections, profil et avis.                                                           |
| Q6  | P7-3 | Acceptation du prêt, destinataires des rappels ? | Emprunteur membre = ami, notifié sans acceptation ; rappel aux deux.                   |
| Q7  | P7-3 | Qui voit le drapeau « prêté » ?                  | Le propriétaire et l'emprunteur.                                                       |
| Q8  | P7-4 | Qui réserve, les offrants se voient-ils ?        | Amis seulement ; nom de l'offrant visible des autres amis.                             |

## Points encore ouverts (à voir en cours de lot)

- Compte supprimé d'un emprunteur ou d'un offrant : nom figé ou effacé (à caler sur le RGPD, avant l'ouverture publique).

- Scission et fusion d'un lot avec des photos d'exemplaire (M5) : les photos restent sur le lot d'origine.
