---
name: audit
description: Audit récurrent du code (sécurité, qualité et tests, conformité aux conventions) — mesures outillées puis agents auditeurs en lecture seule, rapport daté dans docs/audits/ et suivi des constats. Ne corrige rien.
argument-hint: "[securite|qualite|conformite|tout] [--depuis <ref git>] [--rapide]"
disable-model-invocation: true
allowed-tools: Bash(${CLAUDE_SKILL_DIR}/scripts/outils.sh *) Bash(git log *) Bash(git diff *) Bash(git show *) Bash(git rev-parse *) Bash(git describe *) Read Grep Glob Agent
---

Tu conduis un **audit en lecture seule** du projet Collectionnard. Tu n'écris que dans `docs/audits/` ; aucun fichier de
code, de configuration ou de documentation n'est modifié, aucune dépendance installée, aucune commande git qui change
l'arbre. Les correctifs sont **proposés**, jamais appliqués : le mainteneur choisit ensuite un lot.

Arguments : `$ARGUMENTS`. Périmètre : `securite`, `qualite`, `conformite`, ou `tout` (défaut). `--depuis <ref>` : commit
ou tag du dernier audit (sinon, lis-le dans la dernière ligne de « Audits réalisés » de `docs/audits/suivi.md` ; sans
suivi, prends le dernier tag `v*`). `--rapide` : sans build, tests ni govulncheck (revue de code seule, résultat plus
pauvre : le dire dans le rapport).

## Déroulé

1. **Contexte.** Lis `docs/audits/suivi.md` (constats `ouvert`, dernière date, dernier commit audité) et le dernier
   rapport `docs/audits/*.md` s'il existe (ses « Limites » et son « Lot proposé »). Lis `CLAUDE.md`, `server/CLAUDE.md`,
   `web/CLAUDE.md` : ce sont les conventions dont on vérifie la conformité.
2. **Mesures.** Lance `${CLAUDE_SKILL_DIR}/scripts/outils.sh <scratchpad>/audit [--depuis <ref>] [--rapide]` (il se
   place lui-même dans `nix develop` si `go` ou `npm` manquent ; compte plusieurs minutes sans `--rapide`). Lis
   `RESUME.md`, puis les fichiers détaillés utiles (`metriques.txt`, `routes-openapi.txt`, `extensions.txt`,
   `govulncheck.txt`, `npm-audit.txt`, `go-test.txt`…). Ne relance pas ces outils toi-même.
3. **Agents.** Lance **en parallèle** (un seul message, plusieurs appels `Agent`, `subagent_type: auditeur`) un agent
   par dimension du périmètre. Brief de chaque agent, complet car il ne voit pas cette conversation :
   - la dimension et le contenu de sa checklist (`${CLAUDE_SKILL_DIR}/checklists/<dimension>.md`, à lui donner en
     chemin : il la lit lui-même) ;
   - le dossier des mesures (`<scratchpad>/audit`) et les points de `RESUME.md` qui le concernent ;
   - les constats `ouvert` du suivi relevant de sa dimension (identifiant, titre, emplacement) à revérifier ;
   - la borne `--depuis` et la liste des fichiers modifiés depuis (`git.txt`) : audit **complet**, mais regard
     renforcé sur ce qui a changé ;
   - le format de sortie attendu (il le connaît : constats avec gravité, constaté / supposé, `fichier:ligne`, impact,
     correctif, vérification).
4. **Consolidation.** À la réception des rapports :
   - **Revérifie toi-même** (lecture du code, `grep`) chaque constat **critique** ou **haut** et tout constat
     « supposé » que tu peux trancher ; écarte ce qui ne tient pas, rétrograde ce qui est exagéré. Les agents se
     trompent dans le même ton que lorsqu'ils ont raison.
   - Dédoublonne entre dimensions (un même emplacement cité deux fois : un seul constat, la dimension la plus pertinente).
   - Attribue les identifiants en **continuant** la numérotation du suivi (`S-`, `Q-`, `C-` + trois chiffres).
   - Pour chaque constat ouvert au départ : `ouvert` (toujours là), `corrigé` (commit cité), `obsolète` ; ne passe
     jamais un constat en `accepté` toi-même, c'est au mainteneur.
5. **Écriture.** Rédige `docs/audits/AAAA-MM-DD.md` d'après `${CLAUDE_SKILL_DIR}/modele-rapport.md` (même date qu'un
   rapport existant : suffixe `-2`). Mets à jour `docs/audits/suivi.md` : lignes des nouveaux constats, états des anciens,
   ligne « Audits réalisés » (date, commit court, périmètre, lien du rapport, nombre de nouveaux et de corrigés).
   Chemins cités en Markdown cliquable relatif au fichier (`[store/item.go:42](../../server/internal/store/item.go#L42)`).
6. **Compte rendu** au mainteneur, court : état général en une phrase, tableau des constats critique / haute, ce qui
   s'est amélioré depuis le dernier audit, lot de correction proposé (ordre, effort), limites. Ne commite pas : il relit
   d'abord.

## Garde-fous

- Pas de modification hors `docs/audits/`. Si un agent signale qu'il lui manque un droit, ne contourne pas : note-le en
  « Limites ».
- Un constat sans `fichier:ligne` n'entre pas dans le rapport. Les gravités suivent les définitions du modèle ; un
  rapport où tout est « haute » ne sert à rien.
- Les constats `accepté` du suivi ne sont pas repris, sauf si leur emplacement a changé (le signaler alors).
- Les mesures qui échouent par manque de réseau ou d'outil sont consignées telles quelles (« non mesuré »), jamais
  devinées.
- Le rapport doit se lire sans cette conversation : chaque constat est autoportant.
