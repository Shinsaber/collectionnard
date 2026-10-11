# Backend Go — conventions

Module : `github.com/shinsaber/colec` · Entrée : `main.go` · Config par variables
d'environnement (`config/config.go`) : `DATABASE_URL` et `JWT_SECRET` (≥ 32 caractères) sont
**obligatoires** (le serveur refuse de démarrer sinon) ; `PORT`, `ACCESS_TOKEN_TTL` (15m),
`REFRESH_TOKEN_TTL` (720h), `LOGIN_RATE_LIMIT` (10/min/IP), `UPLOAD_DIR`, `MAX_UPLOAD_SIZE`, `MAX_IMPORT_SIZE` (20 Mio), `TRUSTED_PROXY` (false) ont un défaut.

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
| `csvexport`  | Mise à plat CSV des attributs (colonnes depuis le schéma, i18n-text → `clé.langue`, BOM, anti-injection de formules) |
| `dbmigrate`  | Migrations tern (`db/migrations`) + seed : sous-commandes `colec-server migrate` / `seed` (`cmd.go`) |

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
- Les listes renvoient `model.PaginatedResponse` (`data`, `total`, …).
- Lire un corps JSON avec `decodeJSON(w, r, &in)` + `respondDecodeError(w, err)` (limite 1 Mio → 413).
- Logs : `log/slog` (JSON par défaut, `LOG_FORMAT=text` en dev) ; `respondInternal(w, r, err, msg)` et
  `handleStoreError(w, r, err)` loguent la cause des 500 avec le `request_id` — ne jamais renvoyer `err` au client.
- `GET /api/v1/healthz` (public) ping la base : 200 `{"status":"ok"}` ou 503.
- Auth : access token JWT court (`Auth.GenerateToken`) + refresh token opaque dont seul le hash
  SHA-256 est stocké (`store/refresh_token.go`, rotation à chaque `/auth/refresh`).

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
