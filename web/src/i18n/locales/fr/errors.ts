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
      "Types BD / Manga introuvables dans le catalogue : {missing} (type renommé ou db/seed_bd_manga.sql non chargé)",
    external_unavailable:
      "Source externe (Wikidata, BnF) injoignable, réessayez plus tard",
  },
};
