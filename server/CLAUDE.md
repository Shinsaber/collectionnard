# Backend Go — conventions

Module : `github.com/shinsaber/colec` · Entrée : `main.go` · Config par variables
d'environnement (`config/config.go`) : `DATABASE_URL` et `JWT_SECRET` (≥ 32 caractères) sont
**obligatoires** (le serveur refuse de démarrer sinon) ; `PORT`, `ACCESS_TOKEN_TTL` (15m),
`REFRESH_TOKEN_TTL` (720h), `LOGIN_RATE_LIMIT` (10/min/IP, deux compteurs : identifiants / session), `UPLOAD_DIR`, `MAX_UPLOAD_SIZE`, `UPLOAD_RATE_LIMIT` (60 fichiers/h/compte, admins exemptés), `UPLOAD_RATE_BYTES` (300 Mio/h/compte), `UPLOAD_PURGE_INTERVAL` (24h, `0` désactive la purge automatique), `MAX_IMPORT_SIZE` (20 Mio), `GRAPH_MAX_NODES` (500), `TRUSTED_PROXY` (false), `EXTERNAL_USER_AGENT` (User-Agent envoyé à Wikidata / BnF), `MEMBER_INVITES` (false : seuls les admins invitent), `PROPOSAL_QUOTA` (5), `LOG_FORMAT` (json), `LOG_LEVEL` (info) ont un défaut.

## Couches (`internal/`)

| Paquet       | Rôle                                                                                                 |
| ------------ | ---------------------------------------------------------------------------------------------------- |
| `model`      | Structs JSON (entités, `*Create`, `*Update`, `Pagination`, `APIError`)                               |
| `store`      | Interface `Store` (`store.go`) + implémentation pgx (1 fichier/domaine)                              |
| `handler`    | Handlers HTTP, 1 fichier/domaine ; **toutes les routes** dans `handler.go` → `Router()`              |
| `middleware` | Cinq niveaux d'accès du `Router()` : public ; `OptionalAuthenticate` (routes `/public/…`, `/share` : anonyme ou jeton valide) ; `Authenticate` (JWT HS256, `exp` obligatoire) ; `RequireRole(admin, contributor)` (écriture du catalogue, modération des propositions, import externe) ; `RequireRole(admin, moderator)` (modération des avis, I-1) ; `RequireAdmin` (méta-types, administration, suppression d'item). Plus `RateLimiter` (`/auth/*`, deux compteurs), `TrustedProxyRealIP`, `RequestLogger` |
| `validator`  | `ValidateAttributes(schema, data)` + `FormatValidationError`                                         |
| `testutil`   | `MockStore` (champs `XxxFn`), `NewTestServer`, `AdminToken`/`UserToken`, `AuthHeader`                |
| `label`      | `Render(schema, attrs, lang)` : libellé d'un item via `x-label` (pendant de `web/src/lib/schema.ts`) |
| `itemquery`  | Recherche avancée de `GET /items` : filtres `attr.<clé>[.<op>]`, tri `attr.<clé>`, requête plein texte — validés contre le schéma du type (pendant de `web/src/lib/itemSearch.ts`) |
| `collstats`  | Statistiques d'une collection (P4-1) déduites du schéma du CollectionType : nombres, répartitions (enum, booléen), mois des dates, ajouts par mois ; la complétude (relations `x-completeness`) est calculée en SQL dans `store/stats.go` |
| `privacy`    | Visibilité et champs cachés (P4-2) : `CanView`, `Levels`, `HiddenFields` (`x-private` ou `hidden_fields`), `StripAttributes`, `StripSchema`. **Toute lecture faite pour un autre que le propriétaire** (routes `/public/…`, `/share`, statistiques) passe par ce filtre, admins compris |
| `review`     | Règles d'un avis (I-0) : `CheckSchema` (critères = entiers de 0 à 10), `Rate` (moyenne, tous les critères obligatoires), `MinRatings` (seuil d'affichage d'une moyenne), `MaxBody`, `Languages` (pendant de `web/src/i18n/langs.ts`) |
| `identifier` | Champs `x-identifier` d'un schéma (premier niveau ou un niveau dans un objet) et leurs valeurs : doublons (`POST /items/duplicates`, P4-3b) |
| `csvexport`  | Mise à plat CSV des attributs (colonnes depuis le schéma, i18n-text → `clé.langue`, BOM, anti-injection de formules) |
| `dbmigrate`  | Migrations tern (`db/migrations`) + seeds de démonstration (`SEED_FILE` : liste séparée par des virgules ; défaut `seed.sql`, `seed_bd_manga.sql`, `seed_music.sql`, `seed_video.sql` puis `seed_games.sql`), précédés de l'installation des paquets de domaine : sous-commandes `colec-server migrate` / `seed` / `install-domain` (`cmd.go`) |
| `domains`    | Paquets de domaine embarqués (`packages/*.json`, types au format de l'import) et plan d'installation : `Merge3` (référence / local / paquet ; les modifications locales sont gardées, seuls les ajouts du paquet sont fusionnés). `requires` : paquets requis (ex. `books` → `people`, qui porte Person et Publisher), installés d'abord dans la même transaction (`Requirements`, rapport `required`). Le store écrit par l'import et garde les copies de référence (`meta_type_origins`, migration 007) |
| `external`   | Sources ouvertes : client HTTP commun (User-Agent, délai, taille max, cache 15 min, espacement par hôte `Throttle`, 404 → `ErrNotFound`, `StatusError`), `wikidata/` (recherche par classe, entités, `wbgetclaims`), `bnf/` (SRU UNIMARC, ARK + caractère de contrôle, ISBN), `musicbrainz/` (JSON `ws/2`, 1,1 s entre deux requêtes, 503 retenté une fois, jamais tags ni genres), `tvmaze/` (CC BY-SA, 0,55 s entre deux requêtes, 429 retenté une fois, ni résumés ni images), `NormalizeBarcode` (EAN / UPC) ; connecteurs `connectors/` (un profil par sorte : `bd-manga`, `novels`, `music`, `films`, `series`, `games` ; recherche d'éditions BnF des films, séries et jeux par le squelette commun `editions.go` (`editionSearch`) ; correspondance par identifiant, ou par relation depuis le parent (`via` : la diffusion d'une série) ; liens des fiches sources ajoutés à chaque item (`links.go` : Wikidata, Wikipédia fr / en, BnF, MusicBrainz, TVmaze, IMDb ; fusion par URL) ; types du domaine trouvés par identifiant du paquet puis nom technique, rapprochement du catalogue, champs vides seulement, sortie au format de `POST /import`) ; `connectors/connectorstest` = sources simulées (réponses enregistrées dans `testdata/`) + catalogue en mémoire |

## Erreurs et réponses

- Store : renvoyer `store.ErrNotFound` / `store.ErrConflict` ; le handler les traduit en 404 / 409.
- Handler : utiliser `respondJSON`, `respondError(w, status, code, message)` et
  `respondValidationError` (422) — pas de `w.Write` à la main. Seules exceptions : les pièces jointes d'export
  (`handler/export.go`, `attachment` + `csv.Writer` ou corps JSON en flux) et `readImport`, qui lit le corps avec sa
  propre limite `MAX_IMPORT_SIZE` au lieu de `decodeJSON` (1 Mio). Tout code d'erreur émis par `respondError` est nommé
  dans la réponse correspondante d'`api/openapi.yaml` et traduit dans `web/src/i18n/locales/*/errors.ts`.
- Toute écriture d'attributs dynamiques (item, relation, collection item) doit passer par
  `validator.ValidateAttributes` avec le schéma du méta-type correspondant.
- Tout `attributes_schema` écrit sur un méta-type passe par `checkAttributesSchema` (→ `validator.CheckSchema` :
  objet JSON, `"type": "object"` à la racine, compilable, sans `$ref` externe). Les schémas compilés sont
  mis en cache par hash de contenu.
- Paramètres de requête : `parsePagination` borne `per_page` à 100 ; `parseOptionalUUID(w, r, "param")`
  répond 400 sur un UUID invalide ; `GET /items/{id}/graph` limite `max_depth` (5, 10 pour admin),
  `relation_type_ids` (20) et le nombre de nœuds (`GRAPH_MAX_NODES`).
- `GET /items` : filtres et tri par attribut exigent `item_type_id` et passent par `itemquery` (clé présente dans le
  schéma, opérateur adapté au type, valeurs typées) ; le store n'injecte jamais une clé dans le SQL (`attributes -> $n::text`).
  Plein texte : colonne générée `search_tsv` + `f_unaccent` (migration 003, extension `unaccent`).
- Import (`store/export.go`) : une transaction, un savepoint par entité (toutes les erreurs du fichier sont listées),
  upsert `ON CONFLICT (id) … WHERE IS DISTINCT FROM … RETURNING (xmax = 0)` → créé / mis à jour / inchangé ; rollback si
  erreur ou `dry_run`. Toute nouvelle table du catalogue doit être ajoutée à l'export **et** à l'import.
- Sources externes (`handler/external.go`, routes admin `/external/{connector}/*`, liste `GET /external/connectors`) :
  chaque connecteur implémente `external.Connector` (`ID`, `Domain`, recherche d'œuvres et d'éditions, aperçu, `Build`)
  et s'inscrit dans `Handler.Connectors` ; il trouve ses types par identifiant du paquet de domaine, puis par nom, et vérifie lui-même ses identifiants (`work`, `editions` : QID, ARK, MBID). Les
  connecteurs ne dépendent que de l'interface `connectors.Catalog` (sous-ensemble du store) ; erreurs → 400 `InputError`, 404 entité externe inconnue,
  409 `connector_unavailable` (types du domaine absents), 502 `external_unavailable` (source injoignable). L'import
  repasse par `Store.ImportCatalog` : jamais d'écriture directe. Tests sans réseau : `connectorstest.NewSources(t)`.
- Les listes renvoient `model.PaginatedResponse` (`data`, `total`, …).
- Lire un corps JSON avec `decodeJSON(w, r, &in)` + `respondDecodeError(w, err)` (limite 1 Mio → 413).
- Logs : `log/slog` (JSON par défaut, `LOG_FORMAT=text` en dev) ; `respondInternal(w, r, err, msg)` et
  `handleStoreError(w, r, err)` loguent la cause des 500 avec le `request_id` — ne jamais renvoyer `err` au client.
- `GET /api/v1/healthz` (public) ping la base : 200 `{"status":"ok"}` ou 503.
- Images : `/api/v1/uploads/<nom>?w=240` sert une vignette JPEG (liste blanche `thumbWidths`), générée une fois dans
  `UPLOAD_DIR/.thumbs/` et supprimée avec l'original par la purge (`handler/thumbnail.go`). Garde-fous (audit S-001, S-004) :
  `maxImagePixels` (25 Mpx, refusé à l'envoi `image_too_large` et jamais décodé en vignette), `thumbGate` (au plus
  `maxConcurrentThumbnails` décodages en vol, sinon l'original est servi), `uploadQuota` (fichiers et octets par heure et
  par compte, admins exemptés, 429 `upload_quota`), purge automatique `RunUploadPurge` (`UPLOAD_PURGE_INTERVAL`) en plus
  du bouton admin ; `purgeUploads` est la logique commune.
- Auth : access token JWT court (`Auth.GenerateToken`) + refresh token opaque dont seul le hash
  SHA-256 est stocké (`store/refresh_token.go`, rotation à chaque `/auth/refresh`).
- Comptes (`handler/account.go`, `store/account.go`, P4-0) : invitations et liens de réinitialisation suivent le même
  principe (jeton aléatoire `newToken` montré une fois, seul son hash stocké, lien construit par le front). Inscription
  en une transaction (invitation verrouillée `FOR UPDATE`). Désactiver un compte, changer ou réinitialiser son mot de
  passe révoque ses refresh tokens ; `Login` / `Refresh` refusent un compte désactivé (401 `account_disabled`).
  `UpdateUser` verrouille les admins et refuse de retirer le dernier actif (`store.ErrLastAdmin` → 409 `last_admin`).
  Toute colonne qui référence un envoi (comme `users.avatar`) doit figurer dans `ReferencedUploads` (purge des images).
- Historique du catalogue (P4-3a, migration 010) : des déclencheurs PostgreSQL écrivent `item_history` /
  `relation_history` à chaque écriture d'un item ou d'une relation. Toute transaction qui écrit dans le catalogue
  s'ouvre par `s.begin(ctx)` (pas `s.pool.Begin`) pour transmettre l'acteur du contexte (`store.WithActor`, posé
  par le middleware `withActor` ; `store.WithOrigin(ctx, store.OriginImport…)` pour préciser l'origine). Rôles :
  `RequireRole(model.RoleAdmin, model.RoleContributor)` pour l'écriture du catalogue, `RequireAdmin` pour la
  suppression d'item, les méta-types et l'administration. Auteurs d'historique résolus par `authors`
  (`handler/history.go`) selon la visibilité **actuelle** du profil ; les admins voient toujours l'auteur.
- Propositions (`handler/proposal.go`, `store/proposal.go`, P4-3b) : tout changement d'état est gardé en SQL
  (`WHERE status = ANY`, `ErrConflict` → 409 `proposal_state`) ; modération réservée à `model.CanEditCatalog`, jamais sur
  ses propres propositions (403 `own_proposal`). Acceptation par `AcceptCatalogProposal` : import et état dans une
  transaction (`importCatalog(…, before)`), acteur = **auteur** de la proposition, origine `proposal`. Les images d'une
  proposition non décidée sont gardées par la purge (`ReferencedUploads`).
- Notifications (`handler/notification.go`, `store/notification.go`, P4-3c) : `notifyProposal` / `notifyComment`
  après une décision ou un message ; l'émetteur n'est jamais notifié, un échec est journalisé sans faire échouer la
  requête.
- Avis (`handler/review.go`, `store/review.go`, I-0) : toute note compte dans la moyenne, quelle que soit sa visibilité ; ce que
  voit le lecteur se décide au serveur (`reviewListed` en SQL, auteur retiré par `signedFor` via `privacy.CanView`, `user_id`
  jamais sérialisé), admins compris. Une moyenne sous `review.MinRatings` notes ne sort pas de l'API. La liste d'un profil ne
  montre que les avis signés pour le lecteur. L'édition utilisée (`context_item_id`, cible de l'item par une relation de
  `review_type_context_relations`, vérifiée par `IsReviewContext`) est retirée avec l'auteur ; à l'import, ces relations
  s'écrivent après les types de relation (`reviewTypeContexts`).
- Modération (`handler/moderation.go`, `store/moderation.go`, I-1) : on ne signale ou ne réagit qu'à un avis montré au lecteur
  (`reviewShownTo` : ni le sien, ni une note seule, ni un avis masqué ; 404 sinon), jamais à son propre contenu (403
  `own_content`). `ModerateReview` masque / rétablit / classe dans une transaction avec son entrée de `moderation_log`
  (ajout seul, copie du contenu) ; un avis masqué sort des listes, des moyennes et du profil public, son auteur le voit
  avec le motif. Les réactions ne modèrent jamais. Écritures sociales limitées par heure et par compte (`handler/rate.go` :
  avis 60, signalements 20, réactions 300, commentaires 120, articles 10 ; recommandations 20 par jour ; admins exemptés ;
  429 `rate_limited`).
- Articles (`handler/article.go`, `store/article.go`, I-3) : `pending` pour un membre (et après modification),
  `published` directement pour un admin ou un modérateur ; validation par `ReviewArticle` (jamais son propre article) ;
  lien `http(s)` vérifié, jamais récupéré. Le fil d'un article n'existe qu'une fois publié (`articleShownTo`).
- Commentaires (`handler/comment.go`, `store/comment.go`, I-2) : le fil d'un avis suit sa visibilité (`threadVisibleTo` : pas de
  fil pour une note seule) ; `shapeComments` retire le nom de l'auteur d'un avis anonyme qui répond. Modération commune aux
  avis et commentaires (`Store.Moderate`, `moderationTables`).
- Recommandations et blocages (`handler/recommendation.go`, `store/recommendation.go`, I-4) : destinataire de l'annuaire
  (`privacy.CanView`, sinon 404) ; refus et blocage répondent le même 403 `recommendations_refused` ; quota par jour
  (`limitedIn`, fenêtre de `hourlyLimit`). Un bloqué est refusé (403 `blocked`, `refuseBlocked`) seulement quand le
  contenu le nomme au lecteur (`thread.AuthorNamed`) ; sinon le blocage masque sans refuser. Listes du bloqueur :
  `notBlockedBy` en SQL (avis signés, articles), traces `blocked` dans `shapeComments` (jamais pour l'auteur d'un avis
  anonyme), `withoutBlockers` avant toute notification d'un fil. `user_blocks.anonymous` : bloqué sans être nommé, son
  nom ne sort jamais de `/me/blocks`.
- Lots (P7-1, migration 020) : une ligne de `collection_items` est un lot de `quantity` exemplaires ; un item peut former
  plusieurs lots d'une collection. Toute route d'écriture adresse le lot par son identifiant **dans** sa collection
  (`/collections/{collectionId}/lots/{lotId}`, `ownCollection(w, r, "collectionId")`) ; `POST …/items` crée toujours un lot.
  Scission et fusion en transaction (`FOR UPDATE`, lots verrouillés dans l'ordre des id), `ErrInvalidQuantity` / `ErrNotMergeable`
  → 422 ; la fusion réunit les photos des galeries (`galleryKeys`, `mergeGalleries`). Liste d'une collection : la recherche du
  catalogue (`buildItemQuery`, `parseItemSearch` avec ses propres champs de tri, `lotOrderBy`) + `multiple`. Types acceptés
  (P7-1b, migration 021) : `collection_item_types` resserre `collection_type_item_types` (`Collection.AcceptedItemTypes`,
  contrôle à l'ajout et à l'import) ; une collection spécialisée (`SpecializedType`) trie et filtre par attribut sans
  `item_type_id` ; `default_sort` vérifié par `checkCollectionSettings` (422 `invalid_sort`), appliqué sans `sort` par
  `applyDefaultSort`, pages publiques comprises, ignoré s'il ne s'applique plus. Statistiques pondérées par la quantité (`collstats.Entry.Quantity`) ; `item_count` = items distincts, `copy_count` =
  exemplaires.
- Visibilité et partage (`handler/sharing.go`, `store/sharing.go`, P4-2) : routes publiques derrière
  `Auth.OptionalAuthenticate` (sans jeton : anonyme, `GetUserID` = `uuid.Nil` ; jeton invalide : 401) ; `loadReadable`
  vérifie visibilité et compte actif (404 sinon) et calcule les champs cachés ; une nouvelle lecture publique doit
  passer par lui et par `privacy.Strip*`. Liens de partage : même principe que les invitations (hash seul).

## Ajouter / modifier un endpoint — checklist

1. `api/openapi.yaml` (chemin, schémas, codes d'erreur)
2. `internal/model` (structs + tags JSON en snake_case)
3. `internal/store/store.go` (signature dans l'interface) + implémentation pgx
4. `internal/testutil/testutil.go` : ajouter le champ `XxxFn` **et** la méthode sur `MockStore`
   (sinon les tests ne compilent plus)
5. Handler + route dans `Router()` dans le bon groupe (public / authentifié / admin)
6. Test `internal/handler/<domaine>_test.go` (paquet `handler_test`, via `NewTestServer`
   - `MockStore`) : cas succès, 401/403, 404, 422 si validation
7. `go vet ./... && go test ./...`

## Tests

Trois niveaux :

- **Unitaires** (`go test ./...`) : handlers via `MockStore`, middleware, config, validator, dbmigrate,
  helpers du store — sans PostgreSQL.
- **Intégration** (`go test -tags integration ./internal/store/`, fichier `store/integration_test.go`) :
  crée une base `colec_test` à côté de `DATABASE_URL`, applique migrations + seed, exerce le SQL réel
  (graphe multi-chemin, filtres, cascades, 409, jonctions, refresh tokens) puis supprime la base.
  Nécessite PostgreSQL (`pg-start`) ; job dédié en CI.
- Les tests de handlers n'ont **pas besoin de PostgreSQL** (tout passe par `MockStore`).
  Un champ `XxxFn` non renseigné fait renvoyer `store.ErrNotFound` par défaut : ne définir
  que les fonctions utiles au cas testé. Fixtures prêtes : `SampleItemType()`, `SampleItem()`,
  `AdminUser()`, `RegularUser()`.
  Les requêtes SQL complexes se testent à la main via `db/queries_test.sql` contre la base Docker.
