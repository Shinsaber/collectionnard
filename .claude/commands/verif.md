---
description: Vérifier le projet (format, lint, vet, tests Go, lint/type-check/tests front) et résumer les problèmes
---

Lance les vérifications suivantes (préfixe `nix develop -c` si `go`/`npm` n'est pas dans le PATH) et donne-moi un résumé court (OK / problèmes avec fichier:ligne) :

1. `nix flake check`
2. `cd server && gofmt -l . && go vet ./... && golangci-lint run ./... && go test ./...`
3. `cd web && npm run lint && npm run format:check && npm run type-check && npm test`

Ce sont exactement les étapes de la CI (`.github/workflows/ci.yml`).
Ne corrige rien sans me le demander : propose seulement les corrections.
