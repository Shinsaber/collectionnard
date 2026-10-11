# Backend Go — conventions

Module : `github.com/shinsaber/colec` · Entrée : `main.go` · Config par variables
d'environnement (`config/config.go`) : `DATABASE_URL` et `JWT_SECRET` (≥ 32 caractères) sont
**obligatoires** (le serveur refuse de démarrer sinon) ; `PORT`, `ACCESS_TOKEN_TTL` (15m),
`REFRESH_TOKEN_TTL` (720h), `LOGIN_RATE_LIMIT` (10/min/IP), `UPLOAD_DIR`, `MAX_UPLOAD_SIZE`, `TRUSTED_PROXY` (false) ont un défaut.

## Couches (`internal/`)

| Paquet       | Rôle                                                                  |
| ------------ | --------------------------------------------------------------------- |
| `model`      | Structs JSON (entités, `*Create`, `*Update`, `Pagination`, `APIError`) |
| `store`      | Interface `Store` (`store.go`) + implémentation pgx (1 fichier/domaine) |
| `handler`    | Handlers HTTP, 1 fichier/domaine ; **toutes les routes** dans `handler.go` → `Router()` |
| `middleware` | `Authenticate` (JWT HS256, `exp` obligatoire), `RequireAdmin`, `RateLimiter` (`/auth/*`) |
| `validator`  | `ValidateAttributes(schema, data)` + `FormatValidationError`          |
| `testutil`   | `MockStore` (champs `XxxFn`), `NewTestServer`, `AdminToken`/`UserToken`, `AuthHeader` |

## Erreurs et réponses

- Store : renvoyer `store.ErrNotFound` / `store.ErrConflict` ; le handler les traduit en 404 / 409.
- Handler : utiliser `respondJSON`, `respondError(w, status, code, message)` et
  `respondValidationError` (422) — pas de `w.Write` à la main.
- Toute écriture d'attributs dynamiques (item, relation, collection item) doit passer par
  `validator.ValidateAttributes` avec le schéma du méta-type correspondant.
- Les listes renvoient `model.PaginatedResponse` (`data`, `total`, …).
- Lire un corps JSON avec `decodeJSON(w, r, &in)` + `respondDecodeError(w, err)` (limite 1 Mio → 413).
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
   + `MockStore`) : cas succès, 401/403, 404, 422 si validation
7. `go vet ./... && go test ./...`

## Tests

Les tests de handlers n'ont **pas besoin de PostgreSQL** (tout passe par `MockStore`).
Un champ `XxxFn` non renseigné fait renvoyer `store.ErrNotFound` par défaut : ne définir
que les fonctions utiles au cas testé. Fixtures prêtes : `SampleItemType()`, `SampleItem()`,
`AdminUser()`, `RegularUser()`.
Les requêtes SQL complexes se testent à la main via `db/queries_test.sql` contre la base Docker.
