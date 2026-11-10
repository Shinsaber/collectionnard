# Idées et pistes futures

Pistes notées en cours de conception mais **pas retenues pour la phase en cours** : à reprendre quand une phase les
raffinera (elles passent alors dans le document de la phase, et sortent d'ici). Chaque piste indique d'où elle vient.
Les reports propres à une phase livrée restent dans son document (sections « Reporté » de
[`phases/social.md`](phases/social.md) et « Plus tard » de [`phases/mobile.md`](phases/mobile.md)).

## Interactions entre utilisateurs

Issues de l'exploration de la Phase 6 ([`phases/interactions.md`](phases/interactions.md)), 2026-11-02.

| Piste | Idée | Remarques |
| ----- | ---- | --------- |
| **Prêts entre membres** | **Retenu en Phase 7** ([`phases/exemplaires-prets.md`](phases/exemplaires-prets.md)). « Mon DVD est chez Camille depuis le 3 mars » : l'emprunteur est notifié, rappel, historique des prêts, drapeau sur l'exemplaire. | Très « collectionneur », peu de modération. Porte sur un `CollectionItem` (l'exemplaire), pas sur l'item. Un emprunteur hors instance (simple nom) doit rester possible. Mutualisable avec les cadeaux réservés (rattachement au lot, contrepartie membre ou nom libre, période, historique, rappels par une tâche périodique) ; deux tables distinctes, la visibilité étant opposée. Suppose les exemplaires multiples. |
| **Réserver un cadeau** | **Retenu en Phase 7** ([`phases/exemplaires-prets.md`](phases/exemplaires-prets.md)). Dans la liste de souhaits d'un autre, marquer « je l'offre » sans que le propriétaire le voie. | S'appuie sur les listes partagées (P4-2). Le propriétaire ne doit jamais voir la réservation, même par l'API ; visible des autres lecteurs pour éviter les doublons. Jamais en lecture anonyme ni par un lien de partage (le propriétaire pourrait s'y voir). Sur une instance publique, réservation limitée aux **amis** (niveau de visibilité `friends`, relation réciproque) : lot à part, qui touche `privacy.Levels`. |
| **Rappel des cadeaux remis** | Rappeler au propriétaire, au bout de quelques jours, un cadeau remis qu'il n'a ni confirmé (« Bien reçu ») ni refusé. | Écarté de P7-4b le 2026-11-10 : la remise reste visible sur son lot, en attente. S'ajouterait à `RunReminders`. |
| **Exemplaires multiples** | **Retenu en Phase 7** ([`phases/exemplaires-prets.md`](phases/exemplaires-prets.md)). Posséder 5 fois la même carte : une ligne de collection par **lot** d'exemplaires identiques, avec une quantité ; un état ou une finition différents font un autre lot. | Prérequis des prêts, cadeaux et échanges (2 cartes sur 5). Rupture : `UNIQUE (collection_id, item_id)` levée, routes adressées par l'identifiant du lot, import / export, statistiques (somme des quantités), hors ligne. Types concernés marqués par des extensions (`x-lendable`, `x-reservable`, paquet `collections` v5). Piste du 2026-11-07. |
| **Comparer deux collections** | « Ce qu'on a en commun », « ce qu'il a et que je n'ai pas ». | Rien à modérer : un calcul sur ce qui est déjà visible du lecteur (filtre `privacy`). |
| **Suivre un membre** | Fil d'activité : ajouts, avis, articles des membres suivis. | Réutilise les notifications ; attention au bruit : regroupé (résumé quotidien) et réglable par sorte d'événement. |
| **Collections à plusieurs propriétaires** | Une collection de foyer ou de famille, gérée à plusieurs. | Changement de modèle profond : une collection a aujourd'hui un seul `user_id`. Rôles par collection (propriétaire, éditeur, lecteur). |
| **Échanges** | Exemplaires « à échanger », correspondances avec les listes de souhaits des autres, messagerie entre membres. | Déjà reportés à la Phase 4 ([`phases/social.md`](phases/social.md) « Reporté : échanges »). Les prêts et la messagerie en partagent une partie : à raffiner ensemble. |

## Comptes

| Piste | Idée | Remarques |
| ----- | ---- | --------- |
| **Choix à la suppression du compte** | Demander à l'utilisateur ce que deviennent ses contributions publiques (avis, commentaires, articles) : les **anonymiser** (la note reste dans les moyennes) ou les **effacer**. | Pas pour la V1 de la Phase 6. Les contributions au catalogue (historique, propositions) suivent leurs propres règles. Lien avec le RGPD (droit à l'effacement). |

## Avis et modération

| Piste | Idée | Remarques |
| ----- | ---- | --------- |
| **Conservation du journal de modération** | Fixer combien de temps garder les entrées de `moderation_log` (copies de contenu comprises), puis les purger. | Données personnelles : à régler avant l'ouverture publique (RGPD). Piste : 1 ou 2 ans. |
| **Signaler un profil** | Bouton « Signaler » sur un profil (pseudo, avatar, présentation), en plus des avis, commentaires et articles. | Hors V1 de la Phase 6 : le bouton vise les contenus. À revoir avant l'ouverture publique. |
| **Détection des mots injurieux** | Signaler d'office à la modération un avis ou un commentaire contenant des termes injurieux. | Après la modération humaine (I-1). Listes de mots par langue ; jamais de masquage automatique sans relecture. |
| **Spoiler en ligne** | Masquer un passage seulement (`\|\|texte\|\|`) plutôt que l'avis entier. | Extension du rendu Markdown d'I-0. |
| **Note d'exécution par plateforme** | Sur la fiche d'un jeu, une moyenne par console (« Switch 8,2 · PS5 9,1 ») tirée des avis qui précisent leur édition (`context_item_id`, puis `on_platform`). | Regroupe les éditions d'une même plateforme, donc atteint plus vite le seuil de 5 notes. Ne porte que sur les avis signés qui précisent l'édition. |
| **Notes remontées dans le graphe** | Moyenne d'une saison déduite de ses épisodes, d'une série de ses tomes. | Écartée en V1 : la moyenne des parties n'est pas la note du tout. Peut s'appuyer sur `x-completeness`. |
| **Aperçu des articles** | Titre et image Open Graph récupérés pour un article partagé sur une fiche. | Requête sortante vers une URL fournie par un utilisateur : garde-fous SSRF (adresses privées refusées, taille, délai). |
