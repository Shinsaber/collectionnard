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

## P4-3 — Contributions au catalogue

- **Rôle contributeur** : migration du `CHECK` (`admin` | `contributor` | `user`) ; le contributeur écrit dans le
  catalogue (items, relations, sources externes) sans modération, pas dans les méta-types ni l'administration.
- **Propositions** : table `proposals` (`id`, `author_id`, `status` `pending` | `accepted` | `rejected` | `withdrawn`,
  `title`, `payload`, `reviewer_id`, `review_comment`, dates). La charge utile est **au format de l'import du catalogue**
  (items et relations à créer ou modifier) : une proposition acceptée passe par `ImportCatalog`, tout ou rien, comme un
  import externe. Un utilisateur propose depuis les mêmes écrans : formulaire d'item (« Proposer »), correction d'une
  fiche, page « Sources externes » ouverte en mode proposition.
- **Modération** : file des propositions pour les admins (aperçu des différences, acceptation, correction avant
  acceptation, refus commenté) ; pastille du nombre en attente ; l'auteur suit l'état de ses propositions.
- **Historique** : table `item_history` (item, auteur, date, origine — direct, proposition, import —, attributs avant et
  après) pour toute écriture du catalogue ; fiche d'un item : « Historique », retour à une version (admin).

## Reporté : échanges

Exemplaires « à échanger » (attribut ou drapeau d'un CollectionItem), correspondances automatiques avec les listes de
souhaits visibles des autres membres, messagerie simple entre membres. À raffiner après la Phase 4, sur les bases de P4-2
(visibilité) et P4-0 (profils).
