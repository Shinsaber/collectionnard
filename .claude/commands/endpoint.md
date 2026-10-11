---
description: Ajouter ou modifier un endpoint de l'API de bout en bout (OpenAPI → Go → tests → front)
argument-hint: <description de l'endpoint, ex. "GET /items/{id}/tags">
---

Implémente l'endpoint suivant de bout en bout : **$ARGUMENTS**

Suis la checklist de `server/CLAUDE.md` dans l'ordre :

1. Lis `api/openapi.yaml` autour des endpoints voisins pour reprendre le style, puis ajoute/modifie la spécification.
2. Structs dans `server/internal/model`.
3. Signature dans l'interface `Store` + implémentation pgx dans `server/internal/store/<domaine>.go`.
4. Champ `XxxFn` + méthode dans `server/internal/testutil/testutil.go`.
5. Handler dans `server/internal/handler/<domaine>.go` et route dans `Router()` (bon groupe : public / auth / admin).
6. Tests dans `server/internal/handler/<domaine>_test.go` : succès, 401/403, 404, 422 si attributs dynamiques.
7. `web/src/api/types.ts` et `web/src/api/client.ts`.
8. Lance `nix develop -c go vet ./... && nix develop -c go test ./...` (dans server/) et `nix develop -c npm run type-check` (dans web/).

Avant de coder, présente-moi en quelques lignes le contrat (méthode, chemin, corps, réponses, droits) et attends ma validation.
