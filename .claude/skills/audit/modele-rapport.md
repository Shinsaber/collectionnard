# Modèle de rapport — `docs/audits/AAAA-MM-DD.md`

Remplacer les `<…>`. Garder les titres et l'ordre : le suivi et les audits suivants s'appuient dessus.

```markdown
# Audit du <AAAA-MM-DD> — <périmètre : sécurité, qualité et tests, conformité>

> Commit `<abc1234>` (branche `<main>`, tag le plus proche `<v0.4.1>`), audit précédent : `<AAAA-MM-DD>` (`<def5678>`).
> Lecture seule ; revue défensive (constat, impact, correctif, jamais d'exploit). **Constaté** = vu dans le code ou sorti
> d'une commande ; **supposé** = à confirmer. Gravités : **critique** (compromission ou perte de données probable en
> production) · **haute** (exploitable par un utilisateur authentifié, ou impact fort sur la fiabilité) · **moyenne** ·
> **basse** (détail, style, dette mineure).

## 1. Synthèse

<Trois à cinq phrases : état général, ce qui a changé depuis le dernier audit, les deux ou trois points qui méritent un lot.>

| Id | Gravité | Dimension | Titre | Où | État |
| -- | ------- | --------- | ----- | -- | ---- |
| <S-004> | <haute> | <sécurité> | <titre court> | <[store/x.go:42](../../server/internal/store/x.go#L42)> | <nouveau / ouvert depuis AAAA-MM-DD> |

<Trié par gravité, puis par dimension. Tous les constats ouverts à la fin de l'audit, nouveaux comme anciens.>

## 2. Mesures

| Commande | Résultat | Remarque |
| --- | --- | --- |
| <`go vet ./...`> | <OK> | <> |
| <`govulncheck ./...`> | <2 vulnérabilités atteintes> | <détail en S-00x> |

<Reprendre `RESUME.md` ; ajouter couverture par paquet (`go test -cover`), nombre de tests Vitest, taille du build,
fichiers > 400 lignes, écarts routes ↔ OpenAPI, extensions `x-*` non gérées quelque part. « non mesuré » si un outil n'a pas tourné.>

## 3. Évolution depuis le dernier audit

- **Corrigés** : <S-001 (commit `…`), Q-003 (…)> — ou « premier audit ».
- **Toujours ouverts** : <identifiants, avec ce qui a changé le cas échéant>.
- **Obsolètes** : <identifiants et raison>.

## 4. Sécurité

### <S-004 — Titre court> — <gravité> — <constaté|supposé>

**Où** : <[fichier:lignes](lien)>, <autres emplacements>.
**Constat** : <ce qui est>.
**Impact** : <ce que ça permet, ou ce que ça coûte>.
**Correctif** : <quoi changer ; extrait de code court si utile ; effort petit / moyen / gros>.
**Vérification** : <test ou commande qui prouvera la correction>.

<Un bloc par constat, du plus grave au moins grave. Puis :>

**Vérifié sans constat** : <liste courte des points de la checklist passés avec succès>.

## 5. Qualité et tests

<Même structure, identifiants Q-.>

## 6. Conformité aux conventions

<Même structure, identifiants C-.>

## 7. Lot de correction proposé

<Liste ordonnée : d'abord ce qui est critique ou rapide, en précisant l'effort et les constats couverts. Rappeler que
chaque lot suit `CLAUDE.md` § Règles de travail (contrat d'API d'abord, vérifications, audits CVE avant le dernier commit).>

1. <Lot A — S-004, C-002 — petit — …>
2. <…>

## 8. Limites

<Ce qui n'a pas été vérifié et pourquoi : outil absent, réseau, périmètre `--rapide`, zones non lues.>
```

## Rappels pour le suivi (`docs/audits/suivi.md`)

- Une ligne par constat dans « Constats » : `| S-004 | haute | Titre | [store/x.go:42](../../server/internal/store/x.go#L42) | AAAA-MM-DD | ouvert | |`.
- Un constat corrigé garde sa ligne : état `corrigé`, colonne Suite = commit ou lot. Jamais de suppression de ligne.
- Une ligne dans « Audits réalisés » : `| AAAA-MM-DD | abc1234 | tout | [rapport](AAAA-MM-DD.md) | 5 | 2 |`.
