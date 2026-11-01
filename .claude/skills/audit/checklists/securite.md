# Checklist — Sécurité

Revue **défensive** : constat, impact, correctif. Aucun exploit écrit ni testé. Pour chaque point : lire le code réel,
citer `fichier:ligne`, dire constaté / supposé. Les mesures (`govulncheck.txt`, `npm-audit.txt`) sont déjà collectées :
interpréter, pas relancer.

## Authentification et sessions

- JWT (`server/internal/middleware/auth.go`) : HS256 imposé (`WithValidMethods`), `exp` obligatoire, durée courte
  (`ACCESS_TOKEN_TTL`), secret ≥ 32 caractères refusé sinon (`config`). Le rôle est-il relu en base ou figé dans le jeton ?
- Refresh tokens (`store/refresh_token.go`) : seul le hash SHA-256 stocké, rotation à chaque `/auth/refresh`, révocation à la
  déconnexion, au changement de mot de passe, à la désactivation du compte.
- Invitations, liens de réinitialisation, liens de partage (`store/account.go`, `store/sharing.go`) : jeton aléatoire
  montré une fois, hash seul en base, expiration, usage unique (`FOR UPDATE`), jeton en paramètre de requête (hors journaux
  de routes) — vérifier que les journaux (`RequestLogger`) n'écrivent pas la query string.
- Rate limiting (`middleware/ratelimit`) : routes couvertes (`/auth/login`, `/refresh`, `/logout`, `/auth/register`,
  `/auth/reset-password`, `/auth/invitation` ?), clé par IP, `TRUSTED_PROXY` et `X-Real-IP` (usurpable sinon).
- Mots de passe : bcrypt / argon2, coût, longueur minimale (8), comparaison en temps constant pour les hash de jetons.
- Compte désactivé : refusé à la connexion **et** au rafraîchissement ; jeton d'accès déjà émis (15 min) : documenté.
- Front (`web/src/stores/auth.ts`, `api/client.ts`) : où sont stockés les jetons ; rejeu anonyme `publicRead` sans fuite du
  jeton ; déconnexion efface aussi la copie hors ligne (`stores/offline.ts`).

## Autorisation

- Chaque route de `Router()` (`handler/handler.go`) dans le bon groupe : public / authentifié / `RequireRole(admin,
  contributor)` / `RequireAdmin`. Comparer avec `api/openapi.yaml` (sécurité déclarée) et le tableau du README.
- Propriété : collections, items de collection, liens de partage, invitations, propositions, notifications — le `user_id`
  est-il contrôlé **dans le SQL** ou seulement dans le handler ? IDOR possible par identifiant (`collections/{id}/items`,
  `proposals/{id}`, `invitations/{id}`, `share-links/{linkId}`) ?
- Contributeur : écriture du catalogue sans suppression d'item ; suppression de **relation** permise et historisée ;
  jamais de modération de ses propres propositions (`own_proposal`) ; pas d'accès aux méta-types ni à l'administration.
- Admin : `self_lockout`, `last_admin`, impossibilité de retirer le dernier admin actif, verrou `FOR UPDATE`.
- Visibilité (`internal/privacy`, `handler/sharing.go`) : **toute** lecture faite pour un autre que le propriétaire passe
  par `loadReadable` + `privacy.Strip*` (routes `/public/…`, `/share`, statistiques, membres) ; admins compris ; 404 jamais
  403 ; `hidden_fields` validés contre le schéma ; `x-private` retiré du schéma renvoyé (`StripSchema`).
- Historique et attribution : auteur résolu selon la visibilité **actuelle** du profil ; `revert` validé contre le schéma
  courant ; acteur du contexte posé par `withActor` sur toutes les transactions d'écriture (`s.begin`, jamais `pool.Begin`).

## Entrées et injections

- SQL : tout `fmt.Sprintf` du store (liste dans `metriques.txt`) ; clés d'attributs passées en paramètre
  (`attributes -> $n::text`) jamais concaténées ; `itemquery` valide clé, opérateur, type contre le schéma ; tri
  (`buildSortClause`) sur liste blanche ; jokers `ILIKE` échappés ; `statement_timeout` sur le pool.
- JSON : `decodeJSON` partout (limite 1 Mio → 413), `MAX_IMPORT_SIZE` pour l'import, `DisallowUnknownFields` ?
- Attributs dynamiques : `validator.ValidateAttributes` sur **chaque** écriture (item, relation, collection item, revert,
  import, proposition, import externe) ; `checkAttributesSchema` sur chaque écriture de schéma (`$ref` externe refusé,
  taille, profondeur — déni de service par schéma pathologique ?).
- Graphe : `max_depth` (5 / 10 admin), `relation_type_ids` (20), `GRAPH_MAX_NODES`, délai.
- Pagination : `per_page` borné à 100 ; UUID invalides → 400.
- Uploads (`handler/upload.go`, `thumbnail.go`) : type MIME par **contenu** (`http.DetectContentType`), liste blanche
  (SVG refusé ? risque XSS), taille `MAX_UPLOAD_SIZE`, nom = hash (pas de traversée de chemin), service sans exécution,
  `Content-Disposition`, vignettes : largeurs en liste blanche, garde-fou 50 Mpx (bombe de décompression), cache
  `.thumbs/` purgé avec l'original.
- Purge des images : `ReferencedUploads` couvre items, collection items, avatars, propositions non décidées ; délai de
  grâce 24 h ; `dry_run`.
- CSV : protection contre l'injection de formules (`csvexport`), BOM, séparateur.
- Import du catalogue et propositions : `dry_run`, tout ou rien, `base` / conflits, quota (`PROPOSAL_QUOTA`, plafond 50,
  20 créations / 24 h), état gardé en SQL (`WHERE status = ANY`).
- Sources externes (`internal/external`) : hôtes fixes (pas d'URL fournie par l'utilisateur → pas de SSRF), délai 20 s,
  réponses ≤ 20 Mio, cache 15 min, espacement par hôte, User-Agent identifié ; entrées utilisateur (titre, ISBN, QID, ARK,
  MBID) validées avant d'être placées dans une requête (SPARQL / CQL / Lucene : échappement ?).

## Déploiement et configuration

- `docker-compose.yml`, `docker-compose.prod.yml`, `.env.example` : aucun secret par défaut, `SEED_ON_EMPTY=false` en
  prod, seul `web` publié, conteneurs non root, images épinglées, healthcheck.
- nginx (`web/nginx.conf`, `nginx-security-headers.conf`) : CSP (`'wasm-unsafe-eval'` justifié, pas d'`unsafe-inline`
  évitable), `Permissions-Policy: camera=(self)`, `X-Content-Type-Options`, `Referrer-Policy`, `frame-ancestors`, HSTS
  délégué au reverse-proxy (documenté ?), `sw.js` et manifeste en `no-cache`, taille de corps.
- Journaux (`slog`) : pas de jeton, mot de passe ni query string sensible ; `request_id` ; erreurs internes jamais
  renvoyées au client (`respondInternal`).
- CORS : même origine (nginx proxy) ; si un `Access-Control-*` existe, origine fermée.
- `.gitignore` : `.env*`, `.pgdata`, `uploads` ; aucun secret commité (`git log -p -S JWT_SECRET` ciblé si doute).
- Service worker (`web/src/pwa.ts`) : rien de l'API en cache ; IndexedDB effacée à la déconnexion ; `.wasm` servi en local.

## Dépendances

- `govulncheck.txt` : vulnérabilités **atteintes** par le code (pas seulement présentes) ; version du toolchain Go.
- `npm-audit.txt` : avis ≥ haute ; les avis acceptés sont listés dans `web/CLAUDE.md` (comparer : encore valides ?
  correctif publié depuis ?). Jamais recommander `npm audit fix --force`.
- `go-outdated.txt`, `npm-outdated.txt` : retards majeurs, dépendances abandonnées.
- Licences (`go-licences.txt`, `npm-licences.txt`) : tout en MIT / BSD / Apache-2.0 / ISC / CC0 ; signaler toute licence
  non libre ou inconnue (`?`), et PrimeVue ≥ 5 (exclu par décision).
