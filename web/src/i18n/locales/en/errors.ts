import type fr from "../fr/errors";

const en: typeof fr = {
  unknown: "Unknown error",
  network: "Server unreachable",
  required: "This field is required",
  invalidUrl: "Invalid URL",
  api: {
    bad_request: "Invalid request",
    unauthorized: "Authentication required or invalid credentials",
    forbidden: "Administrator rights required",
    not_found: "Not found",
    conflict: "Conflict: this element already exists or is still in use",
    payload_too_large: "File or request too large",
    validation_error: "Invalid data",
    rate_limited: "Too many attempts, please retry in a moment",
    timeout: "The server took too long to respond",
    internal: "Internal server error",
    connector_unavailable:
      "Types not found in the catalogue: {missing}. Install or update the domain from Administration → Domains.",
    external_unavailable:
      "External source (Wikidata, BnF) unreachable, try again later",
    item_type_not_allowed:
      "This item type is not allowed in this collection type",
  },
};

export default en;
