#!/usr/bin/env bash
# Mesures objectives pour /audit : outils de la CI, vulnérabilités, métriques, écarts routes ↔ OpenAPI.
# Usage : outils.sh <dossier-de-sortie> [--depuis <ref git>] [--rapide]
#   --depuis : borne basse pour les diffs (dernier audit) ; --rapide : sans build ni govulncheck ni tests.
# Lecture seule sur le dépôt ; écrit uniquement dans <dossier-de-sortie>. Sortie : un fichier par mesure + RESUME.md.
set -u
ROOT=$(cd "$(dirname "$0")/../../../.." && pwd)
OUT=${1:?dossier de sortie requis}; shift
DEPUIS=""; RAPIDE=0
while [ $# -gt 0 ]; do
  case "$1" in
    --depuis) DEPUIS=$2; shift 2 ;;
    --rapide) RAPIDE=1; shift ;;
    *) echo "argument inconnu : $1" >&2; exit 2 ;;
  esac
done
if { ! command -v go >/dev/null || ! command -v npm >/dev/null; } && [ -z "${COLEC_AUDIT_IN_NIX:-}" ]; then
  export COLEC_AUDIT_IN_NIX=1
  exec nix develop "$ROOT" -c "$0" "$OUT" ${DEPUIS:+--depuis "$DEPUIS"} $([ $RAPIDE = 1 ] && echo --rapide)
fi
mkdir -p "$OUT"
RESUME="$OUT/RESUME.md"
: > "$RESUME"
cd "$ROOT"

# run <nom> <dossier> <commande…> : exécute, garde la sortie, note le code de retour.
run() {
  local nom=$1 dir=$2; shift 2
  ( cd "$dir" && "$@" ) > "$OUT/$nom.txt" 2>&1
  local rc=$?
  printf "| `%s` | %s | %s | %s lignes |\n" "$(printf "%s" "$*" | tr "\n" " " | cut -c1-80)" "$dir" "$([ $rc = 0 ] && echo OK || echo "code $rc")" "$(wc -l < "$OUT/$nom.txt")" >> "$RESUME.cmds"
  return $rc
}
: > "$RESUME.cmds"

# ---- Contexte git -----------------------------------------------------------
{
  echo "commit: $(git rev-parse --short HEAD) ($(git log -1 --format=%cd --date=short))"
  echo "branche: $(git rev-parse --abbrev-ref HEAD)"
  echo "tag le plus proche: $(git describe --tags --abbrev=0 2>/dev/null || echo aucun)"
  echo "arbre: $([ -z "$(git status --porcelain)" ] && echo propre || echo 'modifications non commitées')"
  if [ -n "$DEPUIS" ]; then
    echo; echo "## Depuis $DEPUIS"
    git log --oneline "$DEPUIS..HEAD" | head -50
    echo; echo "### Fichiers modifiés"; git diff --stat "$DEPUIS..HEAD" | tail -40
    echo; echo "### Migrations touchées (autre chose que A = ajout → à signaler)"
    git diff --name-status "$DEPUIS..HEAD" -- db/migrations | grep -v '^A' || echo "aucune modification d'une migration existante"
  fi
} > "$OUT/git.txt"

# ---- Backend Go -------------------------------------------------------------
run gofmt server sh -c 'gofmt -l . ; test -z "$(gofmt -l .)"'
run go-vet server go vet ./...
run golangci-lint server golangci-lint run ./...
if [ $RAPIDE = 0 ]; then
  run go-test server go test -cover ./...
  run govulncheck server govulncheck ./...
fi
run go-outdated server go list -m -u -f '{{if .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all
run go-licences server go list -m -f '{{.Path}} {{.Version}}' all

# ---- Frontend ---------------------------------------------------------------
run npm-lint web npm run lint
run npm-lint-openapi web npm run lint:openapi
run npm-format web npm run format:check
run npm-type-check web npm run type-check
if [ $RAPIDE = 0 ]; then
  run npm-test web npm test
  run npm-build web npm run build
fi
run npm-audit web npm audit --audit-level=high
run npm-outdated web npm outdated
run npm-licences web node -e '
const fs=require("fs");const p=require("./package.json");
for (const [k,v] of Object.entries({...p.dependencies})) {
  let l="?"; try { l=JSON.parse(fs.readFileSync("node_modules/"+k+"/package.json","utf8")).license||"?"; } catch(e){}
  console.log(k, v, "->", typeof l==="string"?l:JSON.stringify(l));
}'

# ---- Métriques de code ------------------------------------------------------
{
  echo "## Fichiers de plus de 400 lignes (hors tests, hors locales i18n)"
  find server web/src -type f \( -name '*.go' -o -name '*.ts' -o -name '*.vue' \) ! -name '*_test.go' ! -name '*.test.ts' ! -path '*/i18n/locales/*' ! -path '*/node_modules/*' \
    -exec wc -l {} + | awk '$1 > 400 && $2 != "total" {print}' | sort -rn
  echo; echo "## SQL construit par fmt.Sprintf dans le store (à vérifier : jamais une clé utilisateur)"
  grep -rn 'fmt.Sprintf' server/internal/store --include='*.go' --exclude='*_test.go' || echo aucun
  echo; echo "## w.Write / http.Error directs dans les handlers (règle : respondJSON / respondError)"
  grep -rnE 'w\.Write\(|http\.Error\(' server/internal/handler --include='*.go' --exclude='*_test.go' || echo aucun
  echo; echo "## pool.Begin hors store.begin (règle : s.begin pour l'acteur de l'historique)"
  grep -rn 'pool\.Begin' server/internal/store --include='*.go' --exclude='*_test.go' || echo aucun
  echo; echo "## catch silencieux côté front"
  grep -rnE 'catch\s*(\([^)]*\))?\s*\{\s*\}' web/src --include='*.ts' --include='*.vue' || echo aucun
  echo; echo "## axios importé hors api/client.ts (règle : un seul point d'appel HTTP)"
  grep -rln "from 'axios'" web/src | grep -v 'api/client.ts' || echo aucun
  echo; echo "## Couleurs Tailwind brutes ou variante dark: dans les vues et composants (règle : utilitaires sémantiques)"
  grep -rnoE '\b(bg|text|border|ring|from|to)-(gray|slate|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-[0-9]{2,3}\b|\bdark:[a-z-]+' web/src/views web/src/components --include='*.vue' | sort | uniq -c | sort -rn | head -40 || true
  echo; echo "## any explicites en TypeScript"
  grep -rnE ':\s*any\b|as any\b' web/src --include='*.ts' --include='*.vue' | grep -v '.test.ts' | wc -l
  echo; echo "## TODO / FIXME / XXX"
  grep -rnE 'TODO|FIXME|XXX' server web/src --include='*.go' --include='*.ts' --include='*.vue' | grep -v node_modules || echo aucun
  echo; echo "## Boutons icône sans aria-label (heuristique : ElButton avec <i et sans aria-label sur la même ligne)"
  grep -rnE '<(el-button|ElButton)[^>]*>\s*<i ' web/src --include='*.vue' | grep -v 'aria-label' | head -20 || echo aucun
} > "$OUT/metriques.txt"

# ---- Routes Go ↔ chemins OpenAPI ------------------------------------------------
{
  norm() { sed -E 's/\{[^}]+\}/{}/g' | sort -u; }
  grep -oE 'r\.(Get|Post|Put|Delete|Patch)\("/api/v1[^"]*"' server/internal/handler/handler.go | sed -E 's/.*"\/api\/v1([^"]*)"/\1/' | sed 's/\*$//' | sed -E 's/\/$//' | norm > "$OUT/.routes"
  grep -E '^  /[a-zA-Z]' api/openapi.yaml | sed -E 's/^  (\/[^:]*):.*/\1/' | norm > "$OUT/.paths"
  echo "## Routes du Router() absentes d'openapi.yaml"; comm -23 "$OUT/.routes" "$OUT/.paths" | sed 's/^/  /' ; echo "  (fin)"
  echo "## Chemins d'openapi.yaml sans route"; comm -13 "$OUT/.routes" "$OUT/.paths" | sed 's/^/  /' ; echo "  (fin)"
  echo "(les paramètres sont normalisés en {} ; /uploads/{} côté Router est servi par un handler générique)"
  rm -f "$OUT/.routes" "$OUT/.paths"
} > "$OUT/routes-openapi.txt"

# ---- Extensions x-* : gérées partout ? ----------------------------------------
{
  echo "## Extensions x-* des paquets de domaine et nombre d'occurrences dans chaque lieu qui doit les connaître"
  printf '%-22s %-14s %-16s %-14s %-8s %-14s\n' extension DynamicForm SchemaBuilder lib/schema.ts README conception
  for x in $(grep -ohE '"x-[a-z-]+"' server/internal/domains/packages/*.json | tr -d '"' | sort -u); do
    c() { grep -rc -- "$x" "$@" 2>/dev/null | awk -F: '{s+=$NF} END {print s+0}'; }
    printf '%-22s %-14s %-16s %-14s %-8s %-14s\n' "$x" "$(c web/src/components/common/DynamicForm.vue web/src/components/common/ImageGalleryField.vue)" \
      "$(c web/src/components/common/SchemaBuilder.vue web/src/components/common/SchemaRootCard.vue web/src/components/common/SchemaFieldCard.vue web/src/components/common/SchemaFieldConstraints.vue web/src/components/common/SchemaImageCategories.vue)" \
      "$(c web/src/lib/schema.ts)" "$(c README.md)" "$(c docs/conception.md)"
  done
} > "$OUT/extensions.txt"

# ---- Résumé --------------------------------------------------------------------
{
  echo "# Mesures collectées le $(date +%F) — $(head -1 "$OUT/git.txt")"
  echo
  echo "| Commande | Dossier | Résultat | Sortie |"; echo "| --- | --- | --- | --- |"
  cat "$RESUME.cmds"
  echo
  echo "Fichiers : git.txt, metriques.txt, routes-openapi.txt, extensions.txt, et un fichier par commande ci-dessus."
  echo
  echo "## Extraits"
  echo '```'; sed -n '1,12p' "$OUT/metriques.txt"; echo '```'
  echo '```'; cat "$OUT/routes-openapi.txt"; echo '```'
  [ -f "$OUT/govulncheck.txt" ] && { echo '```'; grep -E 'Vulnerability|No vulnerabilities|vulnerabilit' "$OUT/govulncheck.txt" | head -20; echo '```'; }
  [ -f "$OUT/npm-audit.txt" ] && { echo '```'; tail -5 "$OUT/npm-audit.txt"; echo '```'; }
  [ -f "$OUT/go-test.txt" ] && { echo '```'; grep -E 'coverage|FAIL' "$OUT/go-test.txt"; echo '```'; }
} > "$RESUME"
rm -f "$RESUME.cmds"
echo "Mesures écrites dans $OUT (voir RESUME.md)"
