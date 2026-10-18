# Phase 4 — Social : conception

Raffinage du 2026-10-17, branche `phase4-social`, un commit par lot. Les lots et leurs cases à cocher sont dans
`Collectionnard.md` § Phase 4 ; ce document porte les décisions, le modèle et l'API visés.

## Décisions du mainteneur

| Sujet         | Décision |
| ------------- | -------- |
| Visibilité    | **Au choix**, par collection et pour le profil : privé, membres de l'instance (connectés), public (visiteurs anonymes). Privé par défaut. |
| Inscription   | **Sur invitation** : lien à usage unique, sans e-mail à envoyer. Un admin invite ; les membres aussi si l'admin l'autorise. |
| Contributions | **Les deux** : propositions modérées pour tous les utilisateurs, et un rôle **contributeur** de confiance qui publie directement. Historique dans les deux cas. |
| Échanges      | **Reportés** après la Phase 4 (exemplaires « à échanger », correspondances avec les listes de souhaits, messagerie). |
| Graphiques    | ECharts (Apache 2.0) : courbes, camemberts, barres ; gère aussi les graphes. |

## État de départ

- Comptes créés en ligne de commande (`create-user`) ; rôle unique `admin` ou `user` (contrainte `CHECK`).
- Catalogue commun (items, types, relations) lisible par tout connecté, modifiable par les admins seuls.
- Collections privées : chacun ne voit que les siennes. Types : Possession, Visionnage, Liste de souhaits.
- `GET /stats` : statistiques globales du catalogue, sans graphique ni rien de personnel.
- Images (`/uploads/…`) déjà servies sans connexion (noms non devinables) : rien à changer pour les pages publiques.

## P4-0 — Comptes, invitations, profils

**Modèle** (migration 008) :

- `users` : `display_name`, `bio`, `avatar` (nom d'un envoi), `disabled_at` (compte désactivé : connexion refusée, jetons
  de rafraîchissement révoqués).
- `invitations` : `id`, `token_hash` (SHA-256, comme les jetons de rafraîchissement), `created_by`, `role` (rôle donné au
  compte), `note`, `expires_at` (1 à 90 jours, 7 par défaut), `used_at`, `used_by`. Un admin invite un admin ou un
  membre ; un membre n'invite que des membres. Une invitation utilisée ne se retire plus (409 `invitation_used`).
- `password_resets` : même principe (hash du jeton, 24 h, un nouveau lien annule les précédents) ; l'utiliser révoque
  toutes les sessions. Refusé pour un compte désactivé.
- Invitations par les membres : réglage d'instance (variable `MEMBER_INVITES`, faux par défaut).

**API** :

- `POST /invitations` (admin, ou membre si permis) → lien `…/register?token=…` affiché une fois ; `GET /invitations`
  (les miennes, toutes pour un admin) ; `DELETE /invitations/{id}`.
- `GET /auth/invitation?token=…` (public ; jeton en paramètre de requête plutôt que dans le chemin, hors des logs de
  routes) : invitation valide ou non ; `POST /auth/register` (public) : `{token, username, email, password}` → compte
  créé, connecté (une transaction : en cas de conflit, l'invitation reste utilisable).
- `GET /users`, `PUT /users/{id}` (admin) : rôle, désactivation ; `POST /users/{id}/reset-link` (admin) : lien de
  réinitialisation du mot de passe, même mécanisme que l'invitation (pas de SMTP) ; `POST /auth/reset-password`
  (public) le consomme. Ni se rétrograder ni se désactiver soi-même (409 `self_lockout`), ni retirer le dernier admin
  actif (409 `last_admin`). Un compte désactivé : 401 `account_disabled` à la connexion et au rafraîchissement ; le
  jeton d'accès déjà émis reste valable jusqu'à son expiration (15 min).
- `GET /auth/me` porte le profil et `can_invite` ; `PUT /me/profile` : nom affiché, présentation, avatar (URL d'un
  envoi, gardée par la purge des images) ; `PUT /me/password` : ancien mot de passe exigé, autres sessions révoquées,
  nouveaux jetons renvoyés.

**Front** : page admin « Utilisateurs » (`/admin/users` : liste, rôle, désactivation, lien de réinitialisation,
invitations), pages publiques `/register` et `/reset-password`, page « Mon profil » (`/profile`, ouverte depuis la carte
de l'utilisateur ; invitations si permis). Livré le 2026-10-17.

## P4-1 — Statistiques personnelles

Toujours génériques : aucune clé d'attribut codée en dur, tout se déduit des schémas.

- **Par collection** : nombre d'exemplaires par type d'item ; pour chaque attribut **numérique** du CollectionType
  (prix payé…) somme, moyenne, minimum, maximum ; pour chaque **enum** (état…) la répartition ; pour chaque **date**
  (date d'acquisition…) la courbe par mois.
- **Complétude** (le graphe sert enfin aux statistiques) : pour un item « contenant » dont l'utilisateur possède une
  partie des enfants, possédés / total — « Dragon Ball (Glénat) : 30 tomes sur 42 », « Saison 1 : 5 épisodes sur 7 ».
  Les relations de composition sont marquées dans leur schéma par une nouvelle extension **`x-completeness: true`**
  (`contains_volume`, `contains_episode`, `contains_track`, `includes_episode`…), gérée par le SchemaBuilder.
- **Progression** (Visionnage) : la complétude d'une collection de visionnage compte les épisodes vus (« Saison 1 : 5
  sur 7 ») ; l'attribut numérique `progress` y est agrégé comme les autres (moyenne, min, max).
- **API** : `GET /me/stats`, `GET /collections/{id}/stats` ; `GET /stats` reste le tableau de bord admin.
- **Front** : page « Statistiques » (toutes mes collections) et onglet « Statistiques » d'une collection, graphiques
  ECharts, thèmes clair / sombre de l'identité « Étiquette ».

**Livré le 2026-10-17.** Choix faits à la réalisation :

- Calcul dans `server/internal/collstats` (Go pur, testé sans base) : `number` / `integer` → somme, moyenne, min, max ;
  `enum` et `boolean` → répartition (valeurs de l'enum dans leur ordre, zéros compris, puis valeurs hors enum, et un
  compte « non renseigné ») ; `format` `date` / `date-time` → nombre par mois, mois vides comblés (sur 50 ans au plus).
  Les ajouts par mois (date d'ajout à la collection) s'y ajoutent.
- Complétude : enfants directs par les relations `x-completeness` (pas de cumul sur plusieurs niveaux) ; les ensembles
  incomplets d'abord, les plus avancés en tête, puis les complets ; 200 au plus (`completeness_truncated`).
  Marquées dans les paquets : `contains_volume`, `contains_work` (books v5), `contains_track` (music v3),
  `contains_episode`, `includes_episode` (tv v5). Une instance existante les reçoit en mettant ces paquets à jour
  (page « Domaines »).
- `GET /stats` reste inchangé (statistiques du catalogue, tableau de bord).
- ECharts 6.1 (Apache 2.0), modules à la carte, rendu SVG ; chargé seulement par la page « Statistiques » et
  l'onglet d'une collection.

## P4-2 — Visibilité et partage

- **Modèle** (migration) : `collections.visibility` et `users.profile_visibility` (`private` | `members` | `public`,
  défaut `private`) ; `share_links` (`collection_id`, `token_hash`, `created_at`, `revoked_at`) pour un lien secret.
- **Attributs sensibles** (décision du 2026-10-18) :
  - **Défaut du type** : une nouvelle extension **`x-private: true`** sur une propriété d'un CollectionType (prix payé,
    budget max…), réglée par l'admin (SchemaBuilder) ; posée dans le paquet `collections` sur `purchase_price` et
    `max_budget`.
  - **Choix du propriétaire, par collection** : `collections.hidden_fields` (`TEXT[]`, NULL par défaut). NULL : les
    champs `x-private` du type sont cachés. Renseignée : c'est la liste complète des champs cachés, qui remplace le
    défaut (cacher un champ de plus, ou montrer un champ privé par défaut, comme le budget d'une liste de souhaits). Clés
    vérifiées contre le schéma du CollectionType ; réglage dans les paramètres de la collection.
  - **Filtrage par le serveur, jamais par l'interface** : pour tout lecteur autre que le propriétaire, les champs cachés
    sont **retirés des réponses** de l'API, sans exception pour les admins (un admin qui lit la collection d'un autre
    est un lecteur comme un autre). Un seul filtre Go (schéma + `hidden_fields` + lecteur → attributs nettoyés), testé
    seul, appelé par toutes les lectures non propriétaires : routes `/public/…` (collection, items), `/share/{token}`,
    et **statistiques** (un champ caché n'entre dans aucun agrégat : un total ou une moyenne du prix payé le révélerait).
    Export et copie hors ligne restent réservés au propriétaire. Tests : chaque route publique est vérifiée sans la
    valeur cachée dans le JSON brut, y compris avec un jeton admin.
- **API** : authentification facultative sur des routes de lecture `GET /public/users/{username}`,
  `GET /public/users/{username}/collections`, `GET /public/collections/{id}` (+ items, statistiques) — `public` pour tous,
  `members` pour les connectés ; `GET /share/{token}` (anonyme, lecture seule) ; `POST` / `DELETE
  /collections/{id}/share-links`. Annuaire des membres (profils `members` ou `public`) pour les connectés.
- **Front** : sélecteur de visibilité sur une collection et sur le profil, page publique d'un profil et d'une collection
  (lecture seule, sans menu d'administration), gestion des liens de partage. Pages publiques non indexées par défaut
  (`noindex`).
- Les listes de souhaits partagées viennent de là : une Wishlist publique ou partagée par lien.

**Livré le 2026-10-18.** Choix faits à la réalisation :

- Migration 009 : `collections.visibility`, `collections.hidden_fields`, `users.profile_visibility`, table `share_links`.
- `server/internal/privacy` (Go pur, testé seul) : `CanView`, `HiddenFields`, `StripAttributes`, `StripSchema` (propriété,
  `required`, `x-order`, `x-labels` : même le libellé du champ caché ne sort pas). Les statistiques reçoivent la liste
  des champs cachés et les retirent du schéma avant tout calcul.
- Les routes publiques appliquent le filtre à **tout** lecteur, propriétaire compris : il y voit sa page telle que les
  autres la voient (sa vue complète reste `GET /collections/{id}`). Réponses 404 (jamais 403) pour ce qui n'est pas
  visible, compte du propriétaire désactivé compris. Authentification facultative : sans jeton = anonyme, jeton
  invalide = 401 ; le client rejoue alors en anonyme si la session ne se renouvelle pas (pas de renvoi vers /login).
- Routes : `GET /public/users/{username}` (+ `/collections`), `GET /public/collections/{id}` (+ `/items`, `/stats`),
  `GET /share?token=…` (+ `/share/items`, `/share/stats`, jeton en paramètre de requête comme pour les invitations),
  `GET|POST /collections/{id}/share-links`, `DELETE /collections/{id}/share-links/{linkId}`, `GET /members`. La liste
  d'un profil exige que le profil soit visible ; une collection publique reste ouverte par son lien même si le profil
  est privé. Les statistiques portent les ItemTypes cités (`item_types`) pour un lecteur anonyme.
- `collections` v2 : `x-private` sur `purchase_price` (Possession) et `max_budget` (Liste de souhaits). Une instance
  existante le reçoit en mettant à jour ce paquet (page « Domaines ») ; d'ici là, rien n'est public (tout est privé
  par défaut).
- Front : bouton « Partager » d'une collection (visibilité, champs cachés, liens), visibilité du profil, pages
  `/u/:username`, `/c/:id`, `/share` (coquille `PublicLayout`, lecture seule), annuaire `/members`, case « Privé »
  par champ dans l'éditeur de types de collection ; `<meta name="robots" content="noindex, nofollow">`.

## P4-3 — Contributions au catalogue

Raffiné le 2026-10-18. Décisions du mainteneur : modération par les **admins et les contributeurs** ; corrections au
format de l'import **avec version de départ** (conflits détectés) ; **notifications dans l'application** (table) dès la V1,
résumé quotidien des non-lues par e-mail plus tard (SMTP) ; brouillons, suggestions de champ et fil de discussion
retenus ; attribution des contributions affichée selon la visibilité **actuelle** du profil ; quota de propositions qui
grandit avec les propositions acceptées.

### Rôles et droits

- Rôles : `admin` | `contributor` | `user` (migration du `CHECK` ; page « Utilisateurs » et `create-user`).
- **Contributeur** : écrit directement dans le catalogue (items, relations, images, sources externes), sans modération ;
  modère les propositions ; **ne supprime pas** (il signale, l'admin supprime : un item présent dans les collections des
  autres ne disparaît pas sur un coup de tête) ; n'accède ni aux méta-types ni à l'administration.
- **Modération** : admins et contributeurs, jamais sur **leurs propres** propositions.
- Membre (`user`) : propose, commente ses propositions, signale.

### Historique (socle de tout le lot)

- Tables `item_history` et `relation_history` : entité, version (entier croissant par entité), auteur (`user_id`, jamais
  un nom), date, **origine** (`direct`, `proposal` + id, `import`, `external` + connecteur, `domain`, `revert`),
  **instantané complet** avant / après (la différence se calcule à l'affichage). Une écriture du catalogue = une ligne,
  dans la même transaction. Tout est conservé (volume faible).
- Fiche d'un item : onglet « Historique » (versions, différences champ par champ, origine) ; **retour à une version**
  (admin, contributeur), lui-même historisé (`revert`).
- **Attribution** (« ajouté par Camille ») : calculée à la lecture depuis `user_id`, selon la visibilité **actuelle** du
  profil pour ce lecteur (P4-2) ; sinon « un membre ». Un compte désactivé ou un profil rendu privé plus tard disparaît
  donc de tout l'historique affiché, rétroactivement. Les admins voient toujours l'auteur (modération).

### Propositions

- Table `proposals` : `id`, `author_id`, `kind` (`catalog` | `schema_suggestion` | `report`), `status`, `title`,
  `payload`, `base`, `reviewer_id`, `review_comment`, dates (création, soumission, décision).
- États : `draft` (brouillon, invisible des modérateurs) → `pending` → `changes_requested` (retour à l'auteur, qui
  corrige et resoumet) → `accepted` | `rejected` (commentaire obligatoire) ; `withdrawn` par l'auteur avant décision.
- **`catalog`** : `payload` au **format de l'import du catalogue** (items et relations à créer ou modifier ; images
  comprises), `base` = **instantané des entités modifiées au moment de la proposition**. À l'acceptation, pour chaque
  entité modifiée : inchangée depuis `base` → appliquée ; changée → **conflit à trois volets** (départ / actuel /
  proposé) que le modérateur tranche champ par champ (même logique que `domains.Merge3`). Le modérateur peut corriger
  la charge avant d'accepter. L'application passe par `ImportCatalog`, tout ou rien, historisée avec l'origine
  `proposal`.
- **`schema_suggestion`** : texte libre adressé aux admins (« il manque le champ illustrateur sur Volume ») ; aucune
  écriture des méta-types.
- **`report`** (signalement) : « cette fiche est fausse » / « à supprimer », commentaire seul, rattaché à un item.
- **Fil de discussion** : table `proposal_comments` (auteur, texte, date) ; auteur et modérateurs échangent sur une
  proposition, à tout état non final.
- **Doublons** : à la saisie, même identifiant qu'un item existant (ISBN, EAN, QID, ARK, MBID… via
  `FindItemsByAttribute`) → avertissement avant l'envoi.
- **Images** : la purge garde les envois cités par une proposition non finale ; un refus ou un retrait les libère.
- **Quota** : propositions `pending` par auteur limitées à `PROPOSAL_QUOTA` (défaut 5) + 1 par proposition acceptée,
  plafonné (50) ; création limitée par jour. Contributeurs et admins sans quota. Un compte désactivé : ses propositions
  restent visibles des modérateurs, marquées comme telles.
- Points d'entrée : formulaire d'item (bouton « Proposer » au lieu de « Enregistrer » pour un membre), « Suggérer une
  correction » et « Signaler » sur une fiche, page « Sources externes » ouverte aux membres en mode proposition,
  « Mes propositions » (brouillons, états, discussion), file « À relire » pour les modérateurs (aperçu des différences,
  conflits, actions).

### Notifications

- Table `notifications` : `user_id`, `kind`, `payload` (références : proposition, item…), `created_at`, `read_at`.
  Types V1 : proposition acceptée, refusée, modifications demandées, nouveau commentaire ; pour les modérateurs, la file
  « À relire » est une pastille calculée (pas une notification par proposition).
- API : `GET /me/notifications` (non lues d'abord), `POST /me/notifications/read` (toutes ou une liste) ; cloche dans
  l'en-tête avec le nombre de non-lues. Même brique réutilisée par les échanges.
- **Plus tard** : résumé quotidien des notifications non lues par e-mail (SMTP, réglage par compte).

### Découpage

1. **P4-3a — Historique et rôle contributeur** : tables d'historique écrites par toute écriture du catalogue, onglet
   « Historique », retour à une version, attribution, rôle `contributor` (écriture directe, pas de suppression).
2. **P4-3b — Propositions et modération** : propositions `catalog` (brouillon, version de départ, conflits), file de
   modération, discussion, doublons, images gardées, quota.
3. **P4-3c — Notifications, signalements, suggestions** : table et cloche, `report` et `schema_suggestion`.

### Livré

- **P4-3a (2026-10-18)** : migration 010 (rôle `contributor`, `item_history`, `relation_history`, déclencheurs). Les
  déclencheurs historisent toute écriture d'`items` / `item_relations` (une écriture sans changement n'ajoute rien) ;
  auteur et origine lus dans les réglages de transaction posés par `store.begin` depuis l'acteur du contexte
  (`store.WithActor`, middleware `withActor` ; `store.WithOrigin` pour l'import et les sources externes). Sans acteur :
  `system` (seeds). Le contributeur peut aussi supprimer une **relation** (corriger un lien), historisée ; jamais un item.
  `GET /items/{id}/history` (500 dernières écritures + la création), `POST /items/{id}/revert` (validé contre le schéma
  actuel). Auteur résolu à chaque lecture : `user` / `member` / `system`. `create-user … --contributor`. Correctif
  au passage : `POST /uploads` ouvert à tout compte connecté (avatar de P4-0, images des propositions).

### Reporté

- Contributeur **par domaine** et **promotion suggérée** (« Camille pourrait devenir contributrice » après N propositions
  acceptées) : ensemble, si l'instance grandit.
- **Fusion de doublons** (regrouper collections, relations et liens sur l'item gardé).
- **Gamification** : liste des champs incomplets à remplir, quota et reconnaissance liés aux contributions.
- Acceptation partielle, acceptation automatique des changements sans risque, votes, expiration des propositions, fil
  « Activité du catalogue », champs verrouillés (`x-locked`), annulation en masse des écritures d'un auteur.

## Reporté : échanges

Exemplaires « à échanger » (attribut ou drapeau d'un CollectionItem), correspondances automatiques avec les listes de
souhaits visibles des autres membres, messagerie simple entre membres. À raffiner après la Phase 4, sur les bases de P4-2
(visibilité) et P4-0 (profils).
