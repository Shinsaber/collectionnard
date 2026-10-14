---
description: Audit approfondi du projet (cohérence, qualité/sécurité, feuille de route) — à lancer avec Claude Fable
argument-hint: "[optionnel : livrable à produire, ex. 2]"
---

Lis intégralement `docs/audit/BRIEF.md` et exécute la mission qui y est décrite.

Si un argument est fourni (`$ARGUMENTS`), ne produis que ce livrable-là (1, 2 ou 3), en relisant
les livrables précédents déjà présents dans `docs/audit/` s'ils existent.

Rappels : lecture seule sur le code, écriture uniquement sous `docs/audit/`, citer `fichier:ligne`,
commandes via `nix develop -c` si les outils ne sont pas dans le PATH.
