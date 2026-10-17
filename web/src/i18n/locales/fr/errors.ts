// Messages d'erreur : `api.<code>` traduit le champ `error` des réponses d'erreur de l'API
// (voir APIError dans api/openapi.yaml) ; le `message` du serveur n'est qu'un repli.
export default {
  unknown: "Erreur inconnue",
  network: "Serveur injoignable",
  required: "Ce champ est requis",
  invalidUrl: "URL invalide",
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
