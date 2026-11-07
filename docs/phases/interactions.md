# Phase 6 — Interactions entre utilisateurs : conception

Exploration ouverte le 2026-11-02, phase close le 2026-11-07 (I-0 à I-4 livrés). Les lots et leurs cases à
cocher sont reportés dans [`docs/conception.md`](../conception.md), Phase 6 ; ce document porte les décisions, le modèle et
l'API visés. Les pistes écartées pour l'instant vivent dans [`docs/idées.md`](../idées.md).

**Cible** : l'instance est appelée à devenir **publique** une fois la phase exploratoire terminée. Tout ce qui suit doit
donc tenir face à des inconnus (abus, spam, harcèlement), même si l'inscription reste sur invitation au départ.

## Décisions du mainteneur

| Sujet                | Décision                                                                                                                                                                                                                                                                                |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Avis                 | **Entité dédiée** `reviews`, une seule ligne par couple (utilisateur, item). Un avis porte sur l'œuvre, pas sur un exemplaire : on peut noter sans posséder.                                                                                                                            |
| Note de collection   | **Supprimée** : la propriété `rating` du type de collection _Visionnage_ disparaît, sans reprise des valeurs (instance encore en bêta).                                                                                                                                                 |
| Critères             | **Génériques dès la V1** : critères définis par type d'item, échelle **fixe sur 10**. Au choix de l'auteur : **note globale** seule, ou **note par critère** dont la **moyenne** donne la note. Affichage en petites étoiles colorées.                                                  |
| Activation           | **Par type d'item** : les avis ne sont ouverts que sur les types choisis, et fermés sur les personnes (Person), pour éviter les attaques contre des gens.                                                                                                                               |
| Contenu d'un avis    | Note obligatoire, texte facultatif (`body`, Markdown restreint), drapeau `spoiler`, langue (`lang`, parmi celles de l'interface, petit drapeau), visibilité, dates de création et de modification.                                                                                      |
| Visibilité d'un avis | **Un seul réglage** : `private`, `members`, `anonymous`, `public` (voir I-0). Défaut : `public` si le profil est public, sinon `anonymous`. Toute note non masquée compte dans la moyenne, affichée **à partir de 5 notes**.                                                            |
| Modération           | **Humaine**, par les **admins et un nouveau rôle `moderator`** : bouton « Signaler » sur chaque avis, commentaire et article, file de modération, **journal de toutes les actions**. La détection automatique des mots injurieux viendra plus tard.                                     |
| Réactions            | Sur un avis (« intéressant », « ça donne envie », « drôle », « désapprouvé ») : donner une **intention ou une émotion**. « Intéressant » sert au tri ; **jamais de signalement**.                                                                                                       |
| Commentaires         | (1) **Échanges sur un avis** ; (2) sur la fiche d'un item, **partage d'articles** (titre, petit message, lien), avec leur propre fil de commentaires, plutôt qu'un fil de discussion libre.                                                                                             |
| Recommandation       | **Retenue** : envoyer une fiche à un membre avec un mot, il reçoit une notification.                                                                                                                                                                                                    |
| Reporté              | Prêts, cadeaux réservés, comparaison de collections, fil d'activité, collections à plusieurs propriétaires, échanges : voir [`docs/idées.md`](../idées.md). Le choix proposé à la **suppression du compte** (anonymiser ou effacer ses contributions) aussi : il n'est pas prévu en V1. |

## État de départ

- `Visionnage` (`d0000000-…-000000000002`, paquet `collections`) porte `rating` (entier de 0 à 10) ; une valeur dans
  `db/seed.sql` (Breaking Bad, 10), deux mentions dans les descriptions d'`api/openapi.yaml`.
- **La mise à jour d'un paquet ne retire rien** : `Merge3` ne fusionne que les ajouts du paquet et garde les
  modifications locales. Retirer `rating` de `collections` v4 ne l'enlève pas d'une instance déjà installée ; il faut
  une migration explicite, même sans reprise des valeurs (I-0).
- Briques réutilisables : visibilité `private` | `members` | `public` et filtre `internal/privacy` (P4-2) ; signalement
  `report` et discussion `proposal_comments` (P4-3b), mais tous deux rattachés aux **propositions du catalogue** ;
  notifications et cloche (P4-3c) ; annuaire des membres ; quotas par compte (envois, propositions).
- Le front affiche déjà des drapeaux de langue (sélecteur de l'en-tête, registre `web/src/i18n/langs.ts`).

## I-0 — Avis et critères

### Où définir les critères : un quatrième méta-type

Fidèle au principe du projet (aucun type codé en dur, tout passe par un JSON Schema porté par un méta-type), les critères
sont décrits par un nouveau méta-type, **`ReviewType`**, dont le schéma liste les critères :

```json
{
  "type": "object",
  "x-order": ["story", "art"],
  "x-labels": {
    "fr": { "story": "Scénario", "art": "Dessin" },
    "en": { "story": "Story", "art": "Art" }
  },
  "properties": {
    "story": { "type": "integer", "minimum": 0, "maximum": 10 },
    "art": { "type": "integer", "minimum": 0, "maximum": 10 }
  },
  "additionalProperties": false
}
```

| Méta-type    | Instance | Le JSON Schema décrit…                                       |
| ------------ | -------- | ------------------------------------------------------------ |
| `ReviewType` | `Review` | les critères notés de l'avis (scénario, dessin, jouabilité…) |

- **Lien avec les types d'item** : `item_types.review_type_id` (NULL = avis fermés sur ce type). Plusieurs types d'item
  peuvent partager un même `ReviewType` (BD et Manga). Un `ReviewType` sans critère n'offre que la note globale.
- **Contraintes vérifiées par le serveur** sur le schéma d'un `ReviewType` (en plus de `checkAttributesSchema`) : chaque
  propriété est un entier de 0 à 10 ; ni objet ni tableau. L'échelle reste ainsi commune et les moyennes comparables.
- **Paquets de domaine** : chaque paquet porte ses `ReviewType` et leur rattachement (UUID préfixe `f1…`, les seeds
  utilisant `f0…` pour leurs collections), versionnés comme les autres types. `people` n'en rattache aucun à Person.
- **Front** : `SchemaBuilder` en mode « critères » (entiers 0–10 seulement), `DynamicForm` affiche déjà un curseur dès
  que `minimum` et `maximum` sont posés. Page admin « Types d'avis » à côté des autres méta-types.

### Deux façons de noter

L'auteur choisit, avis par avis (`criteria` dit lequel) :

| Mode             | Saisie                                 | `criteria` | `rating`                                      |
| ---------------- | -------------------------------------- | ---------- | --------------------------------------------- |
| Note globale     | Une note de 0 à 10 (mode par défaut)   | `NULL`     | La note saisie                                |
| Note par critère | Une note de 0 à 10 pour chaque critère | Renseigné  | Moyenne des critères, calculée par le serveur |

- **En mode « par critère », tous les critères sont obligatoires.** Celui qui ne veut pas tout noter passe
  en note globale. Il n'y a donc pas de critère facultatif, et les moyennes par critère restent comparables.
- **Changer de mode** : `PUT` remplace l'avis ; repasser en note globale efface les critères (le formulaire prévient) et
  préremplit la note avec l'ancienne moyenne.
- **Pas d'avis sans aucune note** : l'avis porte toujours une note, avec ou sans texte.

### Modèle

- Table **`review_types`** (comme les autres méta-types : `name`, `description`, `attributes_schema`, `display_order`).
- Table **`reviews`** : `id`, `user_id`, `item_id`, `review_type_id` (celui du type de l'item à l'écriture),
  `criteria` (JSONB validé contre le schéma, `NULL` en note globale), `rating` (`NUMERIC(3,1)`, note saisie ou moyenne
  calculée par le serveur, stockée pour trier et agréger), `body` (texte, longueur bornée), `spoiler` (booléen),
  `lang` (code du registre des langues),
  `visibility`, `created_at`, `updated_at`, `hidden_at` / `hidden_by` / `hidden_reason` (modération). Contrainte
  `UNIQUE (user_id, item_id)`.
- **Schéma modifié après coup** (critère ajouté ou retiré) : un avis garde ses valeurs et sa note jusqu'à sa prochaine
  modification, où le formulaire demande les critères manquants. Un critère retiré n'est plus affiché.
- **Agrégat sur la fiche** : note moyenne et nombre d'avis, tous modes confondus et à poids égal ; moyenne par critère
  sur les seuls avis détaillés, avec leur nombre (« sur 12 avis détaillés »). Tous les avis non masqués comptent, quelle
  que soit leur visibilité : le chiffre est le même pour tous les lecteurs. **Seuil** : sous 5 notes, le serveur ne
  renvoie que le nombre de notes (« pas encore assez de notes »), et pour chaque critère sous 5 avis détaillés.
  Pas de remontée dans le graphe en V1 (la moyenne des épisodes n'est pas la note de la saison).
- **Affichage** : note sur 10 rendue en 5 étoiles (demi-étoiles), colorées selon la note ; détail des critères au survol
  ou en dépliant l'avis.

### Texte de l'avis

- **Markdown restreint** : rendu côté front par une bibliothèque sous licence permissive, **sans HTML brut ni
  images**, puis assaini (DOMPurify ou équivalent) ; liens en `rel="nofollow ugc noopener"`. Le serveur stocke le
  source et borne sa longueur.
- **Spoiler** : drapeau `spoiler` sur l'avis entier en V1 (texte masqué derrière « Afficher le spoiler ») ; une syntaxe
  en ligne (`||texte||`) pourra suivre.
- **Langue** : choisie à l'écriture parmi celles de l'interface (registre `langs.ts`, défaut : langue courante ; le
  serveur contrôle le code avec une liste Go à garder alignée), petit drapeau sur l'avis ; filtre « avis dans ma
  langue » sur la fiche.

### Visibilité et anonymat

Le besoin exprimé : pouvoir **anonymiser** un avis, ou n'en **garder que la note**. Un **seul réglage**, `visibility` :

| Valeur      | Membres connectés | Visiteurs anonymes | Libellé dans le formulaire (à affiner)                                                  |
| ----------- | ----------------- | ------------------ | --------------------------------------------------------------------------------------- |
| `private`   | Note seule        | Note seule         | « Note seule : ta note compte dans la moyenne, sans ton nom, ton texte reste pour toi » |
| `members`   | Avis signé        | Note seule         | « Membres : avis signé pour les membres, note seule pour les visiteurs »                |
| `anonymous` | Texte sans auteur | Texte sans auteur  | « Anonyme : texte visible de tous, sans ton nom »                                       |
| `public`    | Avis signé        | Avis signé         | « Public : avis signé, visible de tous »                                                |

- L'auteur voit toujours tout son avis. « Note seule » : la note entre dans la moyenne et l'avis n'apparaît pas dans la
  liste de la fiche, seulement un compteur (« et 4 notes sans texte »). Il n'existe pas de note hors de la moyenne : le
  formulaire le dit clairement, et l'interface n'affiche pas « Privé ».
- **Un avis non signé reste attribué en base** : les **modérateurs** (admins et `moderator`) voient l'auteur et les notes (comme l'historique
  pour les admins). Le texte d'un avis `private`, lu par personne, n'a pas à être signalé ni modéré.
- **Le filtre passe par le serveur**, jamais par l'interface : selon le lecteur, la réponse ne contient ni `user_id` ni
  nom, ni texte ni critères pour une note seule. La liste des avis d'un profil ne montre que ceux signés pour ce lecteur.
- **Seuil et déduction** : le seuil de 5 notes rend difficile de lire une note seule dans la moyenne. La limite est
  acceptée, pour rester simple : avec 4 notes signées et 1 note seule, la moyenne permet encore de la recalculer.
- **Abus** : plus la note est anonyme, plus une moyenne peut baisser sans que cela se voie. Les limites de débit
  s'appliquent aussi aux notes seules.
- **Défaut d'un nouvel avis** : `public` si le profil est public, sinon `anonymous` ; le formulaire montre la valeur
  retenue avant l'envoi.

### Disparition de `Visionnage.rating`

Instance encore en bêta : les notes existantes sont **supprimées, sans reprise** en avis.

1. Migration SQL : retirer `rating` des attributs des `collection_items` concernés, puis du schéma installé de
   _Visionnage_ (et de `x-order`, `x-labels`). Le schéma interdit les propriétés inconnues (`additionalProperties: false`) :
   une valeur laissée en place ferait échouer la prochaine écriture de l'exemplaire.
2. Paquet `collections` v4 sans `rating` ; une instance qui avait modifié _Visionnage_ localement garde ses autres
   modifications.
3. `db/seed.sql` : `rating` retiré de l'exemplaire de Breaking Bad ; descriptions d'`api/openapi.yaml` mises à jour.

### API visée

- `GET /items/{id}/reviews` (pagination, tri `recent` | `rating`, plus `interesting` en I-1, filtre `lang`), authentification
  facultative comme `/public/…` ; `GET /items/{id}/reviews/summary` (moyenne, nombre, moyenne par critère).
- `GET|PUT|DELETE /items/{id}/reviews/mine` : un avis par item, donc pas d'identifiant à connaître (PUT = remplacement).
- `GET /me/reviews` ; avis publics d'un profil sur `GET /public/users/{username}/reviews`.
- Méta-type : `GET|POST /review-types`, `GET|PUT|DELETE /review-types/{id}` (admin) ; `review_type_id` sur les ItemTypes.
- Import / export : `review_types` rejoint le catalogue (règle « toute nouvelle table du catalogue… ») ; les avis,
  données personnelles, rejoignent l'export du compte.

### Livré

**I-0 livré le 2026-11-06.** Choix faits à la réalisation :

- Migration 013 : `review_types`, `item_types.review_type_id`, `reviews`, `meta_type_origins` accepte `review_types`, retrait
  de `rating` (attributs, schéma installé, copie de référence). `reviews.review_type_id` n'est renseigné qu'en mode détaillé,
  pour qu'un type d'avis sans avis détaillé reste supprimable. Les colonnes `hidden_*` arrivent avec I-1, qui les écrit.
- `internal/review` (Go pur, testé seul) : forme du schéma, note calculée, limites (`MinRatings` 5, `MaxBody` 10 000 caractères,
  langues à garder alignées sur `langs.ts`). Le filtre reste dans le handler : l'auteur ne sort que par `privacy.CanView`,
  `user_id` jamais (`json:"-"`). Les avis des comptes désactivés ne comptent nulle part.
- Paquets : `collections` v4 (et les domaines d'œuvres, versions ci-dessous). Avis ouverts sur Comic, Manga, Volume,
  Novel, Saga, Series, Season, Episode, Film, Album, Track (note globale seule) et Game ; fermés sur Person, Publisher, Artist,
  Label, les éditions, Broadcast, Universe et Platform. Un type fermé ou supprimé localement le reste à la mise à jour.
- Front : page « Types » à onglets (`TypesLayout`, onglets liés à l'URL), éditeur en mode critères (`features.criteria`),
  onglet « Avis » de la fiche ; sous le titre, moyenne, nombre de notes et ma note (un clic ouvre l'onglet), « Mes avis » sur le profil, avis signés sur la page publique ; Markdown par
  markdown-it (MIT) puis DOMPurify (Apache 2.0). Un curseur non touché part de 5. Étoiles colorées par paliers (0–4 rouge,
  5–7 orange, 8–10 vert, note arrondie ; jetons `--c-star-*`, mode dégradé disponible par `STAR_COLOR_MODE`) ; sur un avis
  détaillé, les critères en colonne à gauche, une ligne par critère.

- **Édition utilisée** (décision du 2026-11-07) : un avis sur une œuvre peut préciser l'édition jouée, vue, lue ou
  écoutée (`reviews.context_item_id`), plutôt que d'ouvrir les avis sur les éditions, ce qui éparpillerait les notes. Le
  type d'avis déclare les relations qui y mènent (`review_type_context_relations` : `game_released_as`, `released_as`,
  `published_as`, `issued_as`) et le libellé du choix (`context_label` : « Joué sur », « Vu dans », « Lu dans »,
  « Écouté sur ») ; le serveur vérifie le lien (422 sinon). L'édition suit l'auteur : **jamais montrée sur un avis non
  signé pour le lecteur** (une édition rare pourrait désigner son propriétaire), le formulaire le rappelle sous le
  sélecteur. Filtre des avis par édition (avis signés seulement). Supprimer l'édition garde l'avis, sans précision.
  Une saison, un épisode, un tome n'ont pas de lien direct vers une édition : pas de choix pour eux. Migration 014 (la 013 était déjà
  appliquée) ; paquets `books` v8, `tv` v8, `video` v6, `music` v6, `games` v5.

**Pas encore fait** (lots suivants ou à trancher) :

- Limites de débit sur les avis : prévues avec le signalement (I-1).
- Pages publiques d'un item : l'API des avis sert les visiteurs, mais aucune page ne les leur montre encore (seule la page
  d'un profil le fait).
- Avis dans l'export du compte : il n'existe pas encore d'export du compte (RGPD).

## I-1 — Signalements, modération et réactions

### Signalements

Le `report` de P4-3b vise une **fiche du catalogue** et passe par les propositions. Les avis, commentaires et articles
demandent un signalement **générique** :

- **Bouton « Signaler »** sur chaque avis, commentaire et article, pour tout membre connecté ; pas sur son propre
  contenu, ni sur une note seule (rien n'est affiché).
- Table `reports` : `target_kind` (`review` | `comment` | `article`), `target_id`, `reporter_id`, `reason`
  (énumération : injurieux, harcèlement, spoiler non signalé, spam, hors sujet, autre) + texte libre, `status`
  (`open` | `actioned` | `dismissed`), `handled_by`, `handled_at`. Un seul signalement ouvert par (signaleur, cible).
- `POST /reports` ; file « Signalements » pour les modérateurs (même pastille que « À relire »).
- **Action** : masquer (`hidden_at`, motif montré à l'auteur), rétablir, ou classer sans suite ; l'auteur est notifié.
  Un contenu masqué disparaît des agrégats et des listes, l'auteur le voit encore avec le motif.
- **Qui modère** : les admins et un **nouveau rôle `moderator`** : relire le catalogue et arbitrer des conflits entre
  personnes ne sont pas le même métier. Les contributeurs gardent les propositions et les signalements de fiches (P4-3),
  sans accès à cette file. Migration : `moderator` ajouté aux contraintes `CHECK` de `users.role` et `invitations.role`,
  `RequireRole(admin, moderator)`, `create-user … --moderator`, page « Utilisateurs ». Le rôle reste **unique** par
  compte : un modérateur n'écrit pas directement dans le catalogue, il passe par les propositions.

### Journal de modération

Toute action est tracée, pour repérer les personnes **trop souvent modérées** :

- Table `moderation_log`, en **ajout seul** : `action` (`hide` | `restore` | `dismiss`), `target_kind`, `target_id`,
  `target_author_id`, `moderator_id`, `report_id` (facultatif), `reason`, `excerpt` (copie du contenu au moment de
  l'action : l'entrée reste lisible si l'auteur supprime son avis), `created_at`.
- **Récapitulatif par membre**, réservé aux modérateurs : dans la file, à côté de chaque signalement (« 3 masquages en
  90 jours »), et `GET /moderation/users/{id}/history` pour le détail. Y figurent aussi les signalements classés sans
  suite d'un membre, pour repérer le signalement abusif.
- **Aucune sanction automatique** : le récapitulatif éclaire la décision. Les suites possibles restent celles d'un
  admin, comme la désactivation du compte (P4-0).
- Les modérateurs sont eux aussi tracés : le journal montre qui a masqué quoi. Durée de conservation : reportée
  (voir [`docs/idées.md`](../idées.md)).

### Réactions

Une réaction donne une **intention ou une émotion**. Ce n'est **pas un moyen de signalement** : aucune réaction, quel que
soit leur nombre, n'envoie un avis en modération.

- Table `review_reactions` : `review_id`, `user_id`, `kind`, `created_at` ; une réaction de chaque sorte par personne, jamais
  sur son propre avis.
- Quatre sortes : `interesting` (intéressant), `tempting` (ça donne envie), `funny` (drôle), `disapprove` (désapprouvé).
  Toutes sont affichées avec leur nombre et se cumulent. Seule `interesting` alimente le tri « les plus intéressants » :
  `tempting` favoriserait les avis positifs, et `disapprove` ferait descendre les avis minoritaires.
- Liste fixe (codes anglais, libellés via vue-i18n) plutôt qu'un méta-type : c'est un vocabulaire d'interface, pas une
  donnée de domaine.

### Livré

**I-1 livré le 2026-11-07.** Choix faits à la réalisation :

- Migration 015 : rôle `moderator` (contraintes de `users` et `invitations`), `reviews.hidden_*`, `reports` (un seul
  signalement ouvert par signaleur et cible, index partiel), `moderation_log`, `review_reactions`. Motifs : `abusive`,
  `harassment`, `spoiler`, `spam`, `off_topic`, `other`.
- `moderation_log` est en ajout seul **par le code** (aucune méthode n'y modifie ni n'y supprime) ; la copie du contenu
  est bornée à 2 000 caractères.
- Seul un avis **montré au lecteur** se signale ou reçoit une réaction : ni le sien (403 `own_content`), ni une note
  seule, ni un avis masqué (404). Un modérateur ne modère pas son propre avis.
- Masquer exige un motif, montré à l'auteur et notifié (`review_hidden`, `review_restored`) ; masquer ou classer ferme
  les signalements ouverts. Une action qui ne s'applique plus répond 409 `moderation_state`.
- Récapitulatif d'un membre : masquages sur 90 jours et au total, signalements classés sans suite sur 90 jours ; à côté
  de chaque signalement, le nombre de signalements classés du signaleur.
- Limites de débit par heure et par compte (`handler/rate.go`, en mémoire) : avis 60, signalements 20, réactions 300 ;
  admins exemptés ; 429 `rate_limited` (code déjà employé par la connexion).
- Front : page « Modération » (file, journal, historique d'un membre ; lien dans l'administration pour les admins et
  les modérateurs, pastille dans la cloche), réactions et « Signaler » sur chaque avis d'un autre membre, tri « Les plus
  intéressants », bannière « Masqué par la modération » sur son propre avis, rôle dans « Utilisateurs » et les
  invitations, `create-user … --moderator`. Sur chaque avis, réactions et « Signaler » en pied : à droite des critères
  (calés en bas) pour un avis détaillé ; texte en retrait marqué d'un filet pour une note globale.

## I-2 — Commentaires

Un seul fil pour les avis et, en I-3, pour les articles.

- Table `comments` : `target_kind` (`review` | `article`), `target_id`, `author_id`, `body` (Markdown restreint, même rendu que l'avis), `created_at`,
  `updated_at`, champs de modération. Fil **à plat** en V1 (une citation suffit pour répondre à quelqu'un).
- Visibilité : celle de l'avis ; une note seule n'a pas de fil. Dans le fil d'un avis `anonymous`, l'auteur apparaît
  comme « Auteur de l'avis », jamais sous son nom. Un avis `members` n'a de fil que pour les membres.
- Notifications : l'auteur de l'avis ou de l'article et les participants du fil, jamais l'émetteur (comme `notifyComment`).
- L'auteur de l'avis ou de l'article peut **fermer** son fil ; signalement et masquage comme en I-1.

### Livré

**I-2 livré le 2026-11-07.** Choix faits à la réalisation :

- Migration 016 : `comments` (`target_kind` `review` | `article`, champs de modération), `reviews.comments_closed_at`,
  déclencheur qui supprime le fil avec son avis (suppression par l'auteur ou de l'item en cascade).
- Le fil existe pour un avis listable (ni note seule, ni anonyme sans texte) et suit sa visibilité ; son auteur y a
  toujours accès. Les commentaires sont **signés** ; seul l'auteur d'un avis anonyme y apparaît comme « Auteur de
  l'avis » (le serveur n'envoie pas son nom), y compris dans la notification (`by_review_author`). Correctif du 2026-11-07 : le
  nom d'un commentateur, comme celui de l'auteur d'un article, n'est envoyé qu'aux lecteurs qui peuvent voir son profil
  (`privacy.CanView`) ; sinon « Un membre », notification comprise (`by_member`). Un avis signé garde le nom, choisi
  explicitement par sa visibilité.
- 1 à 5 000 caractères, Markdown restreint ; on modifie et supprime les siens. Traces (migration 018, 2026-11-07) : un fil
  étant à plat, un commentaire supprimé garde sa place (« Commentaire supprimé par son auteur », texte effacé en base),
  sauf s'il était le dernier du fil : il s'efface alors, avec les traces restées juste avant lui (un commentaire masqué
  arrête ce nettoyage) ; un commentaire masqué apparaît aux autres comme « Commentaire masqué par la
  modération », sans texte ni auteur ; « (modifié) » signale une modification (`edited_at`). Les traces ne comptent pas. Fil fermé (ou avis masqué) : 409
  `comments_closed`, les commentaires restent lisibles. 120 commentaires par heure et par compte.
- Notification `review_comment` à l'auteur de l'avis et aux participants, jamais à l'émetteur ni en double.
- Signalement et modération des commentaires comme pour les avis (`target_kind: comment`, routes
  `/moderation/comments/{id}/…`, notifications `comment_hidden` / `comment_restored`) ; la file montre le commentaire et
  l'avis de son fil. `Store.Moderate` traite les deux sortes.
- Front : bouton « Commenter » / « N commentaires » dans le pied de l'avis, fil déplié sous l'avis (auteur, badge
  « Auteur de l'avis », modifier / supprimer les siens, « Signaler » les autres, « Fermer le fil » sur son avis).

## I-3 — Articles sur la fiche d'un item

Relayer les nouvelles d'une œuvre (sortie d'un tome, annonce d'une saison, critique de presse) :

- Table `item_articles` : `item_id`, `author_id`, `title`, `url` (http/https seulement), `message` court, `lang`,
  `created_at`, champs de modération.
- **Pas d'aperçu récupéré par le serveur** en V1 (titre, image Open Graph) : ce serait une requête sortante vers une URL
  fournie par un utilisateur (risque SSRF) ; à revoir avec une liste de garde-fous si le besoin se confirme.
- Onglet « Actualités » de la fiche, les plus récents d'abord ; liens `rel="nofollow ugc noopener"`.
- Anti-spam : quota par compte (comme `PROPOSAL_QUOTA`), signalement, masquage.
- **Fil de commentaires** : chaque article a le sien (table `comments` d'I-2, `target_kind = article`).

### Livré

**I-3 livré le 2026-11-07.** Décision du mainteneur à la réalisation : articles ouverts sur **tous les types d'item**,
mais l'article d'un membre **attend d'être validé**, comme une proposition de fiche. Choix faits :

- Migration 017 : `item_articles` (`status` `pending` | `published` | `rejected`, motif du refus, champs de modération,
  `comments_closed_at`), déclencheur qui supprime le fil avec son article.
- Les admins et les modérateurs publient directement et valident ceux des autres (jamais le leur) ; les contributeurs
  passent par la validation comme les membres. Modifier un article publié le **renvoie en attente**. Refuser exige un
  motif, montré à l'auteur ; notifications `article_published`, `article_rejected`.
- Titre 1 à 200 caractères, lien `http(s)` sans identifiants (2 000 au plus), un mot de 1 000 au plus, langue de
  l'interface ; 10 articles (création ou modification) par heure et par compte. Le serveur ne récupère jamais le lien.
- La liste d'une fiche montre les articles publiés et non masqués, plus tous ceux du lecteur avec leur état ; le motif
  d'un refus ou d'un masquage ne va qu'à son auteur.
- Un article publié se signale, se masque, se rétablit comme un avis (`/moderation/articles/{id}/…`) et porte un fil de
  commentaires (commun avec les avis : `thread`, `commentApi` par sorte). Dans les commentaires, `review_author`
  devient `target_author` (auteur de l'avis ou de l'article).
- Front : onglet « Actualités · N » de la fiche (partage, liste, état, fil, signalement), onglet « Articles à valider »
  de la page Modération, pastille de la cloche (signalements + articles en attente).

## I-4 — Recommander un item à un membre

- `POST /items/{id}/recommendations` (`to` : nom d'utilisateur, `message` court) → notification du destinataire, qui
  ouvre la fiche, l'ajoute à une collection (liste de souhaits par exemple) ou l'ignore.
- Table `recommendations` (expéditeur, destinataire, item, message, `created_at`, `read_at`) ; page « Recommandations
  reçues » ; quota par jour.
- Qui peut recommander à qui : tout membre de l'annuaire (profil `members` ou `public`) ; le destinataire peut **refuser
  les recommandations** (réglage du compte), et bloquer un expéditeur (voir ci-dessous).

### Livré

**I-4 livré le 2026-11-07.** Décisions du mainteneur à la réalisation : le **blocage complet** fait partie du lot ; le
message d'une recommandation, privé, **ne se signale pas** (le destinataire bloque l'expéditeur). Choix faits :

- Migration 019 : `recommendations` (une par expéditeur, destinataire et item tant qu'elle n'est pas écartée : 409
  `already_recommended`), `users.accepts_recommendations` (vrai par défaut, case du profil), `user_blocks`.
- Destinataire : un membre de l'annuaire (profil `members` ou `public`), sinon 404 comme un inconnu. Refus et blocage
  répondent le même 403 `recommendations_refused` : l'expéditeur ne sait pas lequel. Message en texte brut, 500
  caractères au plus ; 20 recommandations **par jour** et par compte (`handler/rate.go` prend désormais une fenêtre).
  L'expéditeur est nommé au destinataire, même avec un profil privé : c'est lui qui envoie.
- Page « Recommandations » (`/recommendations`, menu) : l'item, l'expéditeur, son mot ; ouvrir, écarter, bloquer. Les
  nouvelles sont marquées lues à la visite ; la notification `recommendation` y mène.
- **Blocage** (`/me/blocks`) : par nom (profil visible), ou depuis un avis, un commentaire, un article ou une
  recommandation reçue (bouton « Bloquer » à côté de « Signaler », et sur le profil public). Le bloqué ne peut plus
  recommander d'item au bloqueur, ni commenter ou réagir à ses contenus **signés** (403 `blocked`) ; retirer sa
  réaction reste permis. Ses avis signés et ses articles sortent des listes du bloqueur, son profil n'y montre plus
  d'avis, ses commentaires deviennent des traces (« Commentaire d'un membre que vous avez bloqué »), ses messages ne
  notifient plus le bloqueur. Ses recommandations au bloqueur sont écartées. Les notes comptent toujours dans les
  moyennes.
- **Anonymat d'abord** : sur un contenu qui ne nomme pas son auteur, le blocage ne refuse rien et ne retire rien.
  Commenter ou réagir à un avis anonyme du bloqueur reste possible (le bloqueur ne voit pas le commentaire et n'est
  pas notifié) ; l'avis anonyme d'un membre bloqué reste listé, et sa réponse dans son fil n'est pas marquée : le
  retirer le nommerait. Bloqué depuis un contenu qui ne le nommait pas (avis anonyme, commentaire « Un membre »), le
  membre reste « Un membre » dans la liste des blocages (`user_blocks.anonymous`) ; un blocage par nom le lève.
  Limite acceptée : les commentaires « Un membre » d'un même auteur bloqué deviennent tous des traces, ce qui les
  relie entre eux.

## Questions transverses

- **Blocage et mise en sourdine** : indispensables avant l'ouverture publique. Blocage livré en I-4 (voir plus haut) ;
  la mise en sourdine seule (sans bloquer) n'est pas faite. Bloquer un membre : il ne peut plus
  commenter mes avis, réagir, me recommander d'item ; ses contenus me sont masqués.
- **Limites de débit** : avis, commentaires, articles, réactions et recommandations par heure et par compte (même principe
  que `UPLOAD_RATE_LIMIT`).
- **Cadre légal d'une instance publique** (à vérifier avant l'ouverture) : l'hébergeur de contenus d'utilisateurs doit
  permettre le signalement et retirer promptement le contenu manifestement illicite ; conservation éventuelle de
  données d'identification ; mentions légales. Le signalement d'I-1 est la base.
- **RGPD** : avis, commentaires et réactions sont des données personnelles ; ils entrent dans l'export du compte. Le choix
  à la suppression du compte (anonymiser ou effacer) est reporté (voir [`docs/idées.md`](../idées.md)).

## Découpage proposé

1. **I-0 — Avis et critères** : `ReviewType`, `reviews`, note moyenne et étoiles, Markdown, spoiler, langue,
   visibilité, suppression de `Visionnage.rating` (paquet `collections` v4).
2. **I-1 — Signalements, modération et réactions** : rôle `moderator`, bouton « Signaler », file de modération, masquage,
   journal et récapitulatif par membre, réactions.
3. **I-2 — Commentaires** : fil sur les avis (table générique), notifications, fermeture du fil.
4. **I-3 — Articles sur la fiche** : onglet « Actualités », fil de commentaires, quota.
5. **I-4 — Recommandations** : envoi d'une fiche à un membre, réglage et blocage.

I-1 suit de près I-0 : aucun contenu libre ne devrait être visible des autres sans signalement possible.
