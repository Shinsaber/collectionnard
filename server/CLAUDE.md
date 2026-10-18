# Backend Go — conventions

Module : `github.com/shinsaber/colec` · Entrée : `main.go` · Config par variables
d'environnement (`config/config.go`) : `DATABASE_URL` et `JWT_SECRET` (≥ 32 caractères) sont
**obligatoires** (le serveur refuse de démarrer sinon) ; `PORT`, `ACCESS_TOKEN_TTL` (15m),
`REFRESH_TOKEN_TTL` (720h), `LOGIN_RATE_LIMIT` (10/min/IP), `UPLOAD_DIR`, `MAX_UPLOAD_SIZE`, `MAX_IMPORT_SIZE` (20 Mio), `TRUSTED_PROXY` (false), `EXTERNAL_USER_AGENT` (User-Agent envoyé à Wikidata / BnF), `MEMBER_INVITES` (false : seuls les admins invitent) ont un défaut.

## Couches (`internal/`)

| Paquet       | Rôle                                                                                                 |
| ------------ | ---------------------------------------------------------------------------------------------------- |
| `model`      | Structs JSON (entités, `*Create`, `*Update`, `Pagination`, `APIError`)                               |
| `store`      | Interface `Store` (`store.go`) + implémentation pgx (1 fichier/domaine)                              |
| `handler`    | Handlers HTTP, 1 fichier/domaine ; **toutes les routes** dans `handler.go` → `Router()`              |
| `middleware` | `Authenticate` (JWT HS256, `exp` obligatoire), `RequireAdmin`, `RateLimiter` (`/auth/*`)             |
| `validator`  | `ValidateAttributes(schema, data)` + `FormatValidationError`                                         |
| `testutil`   | `MockStore` (champs `XxxFn`), `NewTestServer`, `AdminToken`/`UserToken`, `AuthHeader`                |
| `label`      | `Render(schema, attrs, lang)` : libellé d'un item via `x-label` (pendant de `web/src/lib/schema.ts`) |
| `itemquery`  | Recherche avancée de `GET /items` : filtres `attr.<clé>[.<op>]`, tri `attr.<clé>`, requête plein texte — validés contre le schéma du type (pendant de `web/src/lib/itemSearch.ts`) |
| `collstats`  | Statistiques d'une collection (P4-1) déduites du schéma du CollectionType : nombres, répartitions (enum, booléen), mois des dates, ajouts par mois ; la complétude (relations `x-completeness`) est calculée en SQL dans `store/stats.go` |
| `privacy`    | Visibilité et champs cachés (P4-2) : `CanView`, `Levels`, `HiddenFields` (`x-private` ou `hidden_fields`), `StripAttributes`, `StripSchema`. **Toute lecture faite pour un autre que le propriétaire** (routes `/public/…`, `/share`, statistiques) passe par ce filtre, admins compris |
| `csvexport`  | Mise à plat CSV des attributs (colonnes depuis le schéma, i18n-text → `clé.langue`, BOM, anti-injection de formules) |
| `dbmigrate`  | Migrations tern (`db/migrations`) + seeds de démonstration (`SEED_FILE` : liste séparée par des virgules ; défaut `seed.sql`, `seed_bd_manga.sql`, `seed_music.sql`, `seed_video.sql` puis `seed_games.sql`), précédés de l'installation des paquets de domaine : sous-commandes `colec-server migrate` / `seed` / `install-domain` (`cmd.go`) |
| `domains`    | Paquets de domaine embarqués (`packages/*.json`, types au format de l'import) et plan d'installation : `Merge3` (référence / local / paquet ; les modifications locales sont gardées, seuls les ajouts du paquet sont fusionnés). `requires` : paquets requis (ex. `books` → `people`, qui porte Person et Publisher), installés d'abord dans la même transaction (`Requirements`, rapport `required`). Le store écrit par l'import et garde les copies de référence (`meta_type_origins`, migration 007) |
| `external`   | Sources ouvertes : client HTTP commun (User-Agent, délai, taille max, cache 15 min, espacement par hôte `Throttle`, 404 → `ErrNotFound`, `StatusError`), `wikidata/` (recherche par classe, entités, `wbgetclaims`), `bnf/` (SRU UNIMARC, ARK + caractère de contrôle, ISBN), `musicbrainz/` (JSON `ws/2`, 1,1 s entre deux requêtes, 503 retenté une fois, jamais tags ni genres), `tvmaze/` (CC BY-SA, 0,55 s entre deux requêtes, 429 retenté une fois, ni résumés ni images), `NormalizeBarcode` (EAN / UPC) ; connecteurs `connectors/` (un profil par sorte : `bd-manga`, `novels`, `music`, `films`, `series`, `games` ; correspondance par identifiant, ou par relation depuis le parent (`via` : la diffusion d'une série) ; liens des fiches sources ajoutés à chaque item (`links.go` : Wikidata, Wikipédia fr / en, BnF, MusicBrainz, TVmaze, IMDb ; fusion par URL) ; types du domaine trouvés par identifiant du paquet puis nom technique, rapprochement du catalogue, champs vides seulement, sortie au format de `POST /import`) ; `connectors/connectorstest` = sources simulées (réponses enregistrées dans `testdata/`) + catalogue en mémoire |

## Erreurs et réponses

- Store : renvoyer `store.ErrNotFound` / `store.ErrConflict` ; le handler les traduit en 404 / 409.
- Handler : utiliser `respondJSON`, `respondError(w, status, code, message)` et
  `respondValidationError` (422) — pas de `w.Write` à la main.
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
- Images : `/api/v1/uploads/<nom>?w=240` sert une vignette JPEG (liste blanche `thumbWidths`, garde-fou 50 Mpx), générée une fois dans `UPLOAD_DIR/.thumbs/` et supprimée avec l'original par la purge (`handler/thumbnail.go`).
- Auth : access token JWT court (`Auth.GenerateToken`) + refresh token opaque dont seul le hash
  SHA-256 est stocké (`store/refresh_token.go`, rotation à chaque `/auth/refresh`).
- Comptes (`handler/account.go`, `store/account.go`, P4-0) : invitations et liens de réinitialisation suivent le même
  principe (jeton aléatoire `newToken` montré une fois, seul son hash stocké, lien construit par le front). Inscription
  en une transaction (invitation verrouillée `FOR UPDATE`). Désactiver un compte, changer ou réinitialiser son mot de
  passe révoque ses refresh tokens ; `Login` / `Refresh` refusent un compte désactivé (401 `account_disabled`).
  `UpdateUser` verrouille les admins et refuse de retirer le dernier actif (`store.ErrLastAdmin` → 409 `last_admin`).
  Toute colonne qui référence un envoi (comme `users.avatar`) doit figurer dans `ReferencedUploads` (purge des images).
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
