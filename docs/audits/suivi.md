# Audits — suivi des constats

Les audits de qualité, conformité et sécurité sont lancés à la main avec `/audit` (Claude Code ; skill
`.claude/skills/audit/`, agent lecture seule `.claude/agents/auditeur.md`). Chaque audit produit un rapport daté
`docs/audits/AAAA-MM-DD.md` et met ce tableau à jour. Rien n'est corrigé pendant l'audit : les correctifs font l'objet d'un
lot séparé, choisi par le mainteneur.

- **Identifiants stables** : `S-nnn` sécurité, `Q-nnn` qualité et tests, `C-nnn` conformité aux conventions. Un constat garde
  son identifiant d'un audit à l'autre ; la numérotation ne repart jamais à zéro.
- **États** : `ouvert` · `corrigé` (commit ou lot) · `accepté` (risque assumé, justification dans la colonne Suite) ·
  `obsolète` (le code concerné a disparu).
- **Gravités** : critique · haute · moyenne · basse (définitions dans le modèle de rapport).
- Un constat `accepté` n'est plus listé par les audits suivants tant que son emplacement n'a pas changé.

Lancer sans surveillance (facultatif, non installé) : `nix develop -c claude -p "/audit" --permission-mode dontAsk`
dans un timer systemd ou une crontab ; le rapport et ce tableau sont alors à relire et à commiter à la main.

## Constats

| Id | Gravité | Titre | Où | Ouvert le | État | Suite |
| -- | ------- | ----- | -- | --------- | ---- | ----- |

## Audits réalisés

| Date | Commit | Périmètre | Rapport | Nouveaux | Corrigés |
| ---- | ------ | --------- | ------- | -------- | -------- |
