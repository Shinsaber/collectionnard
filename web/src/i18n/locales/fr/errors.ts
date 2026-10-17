// Messages d'erreur : `api.<code>` traduit le champ `error` des réponses d'erreur de l'API
// (voir APIError dans api/openapi.yaml) ; le `message` du serveur n'est qu'un repli.
export default {
  unknown: "Erreur inconnue",
  network: "Serveur injoignable",
  required: "Ce champ est requis",
  invalidUrl: "URL invalide",
  // Vérification d'un champ contre son schéma avant l'envoi (lib/validateValue.ts).
  check: {
    type: "Type attendu : {type}",
    enum: "Valeur non permise",
    pattern: "Format attendu : {hint}",
    minLength: "Au moins {n} caractère(s)",
    maxLength: "Au plus {n} caractères",
    minimum: "Valeur minimale : {n}",
    maximum: "Valeur maximale : {n}",
    date: "Date invalide (AAAA-MM-JJ)",
    uri: "URL invalide",
    email: "Adresse e-mail invalide",
    minItems: "Au moins {n} élément(s)",
    maxItems: "Au plus {n} éléments",
    uniqueItems: "Éléments en double",
    requiredKey: "« {key} » est requis",
    keyPattern: "Clé non permise : {key}",
    unknownKey: "Clé non prévue : {key}",
    json: "JSON invalide",
    types: {
      string: "texte",
      integer: "nombre entier",
      number: "nombre",
      boolean: "oui / non",
      array: "liste",
      object: "objet",
    },
  },
  api: {
    bad_request: "Requête invalide",
    unauthorized: "Authentification requise ou identifiants incorrects",
    forbidden: "Droits administrateur requis",
    not_found: "Élément introuvable",
    conflict: "Conflit : cet élément existe déjà ou est encore utilisé",
    payload_too_large: "Fichier ou requête trop volumineux",
    validation_error: "Données invalides",
    rate_limited: "Trop de tentatives, réessayez dans un instant",
    timeout: "Le serveur a mis trop de temps à répondre",
    internal: "Erreur interne du serveur",
    connector_unavailable:
      "Types introuvables dans le catalogue : {missing}. Installez ou mettez à jour le domaine depuis Administration → Domaines.",
    external_unavailable:
      "Source externe (Wikidata, BnF) injoignable, réessayez plus tard",
    item_type_not_allowed:
      "Ce type d'item n'est pas permis dans ce type de collection",
  },
};
