import type fr from "../fr/errors";

const en: typeof fr = {
  unknown: "Unknown error",
  network: "Server unreachable",
  required: "This field is required",
  invalidUrl: "Invalid URL",
  check: {
    type: "Expected type: {type}",
    enum: "Value not allowed",
    pattern: "Expected format: {hint}",
    minLength: "At least {n} character(s)",
    maxLength: "At most {n} characters",
    minimum: "Minimum value: {n}",
    maximum: "Maximum value: {n}",
    date: "Invalid date (YYYY-MM-DD)",
    uri: "Invalid URL",
    email: "Invalid email address",
    minItems: "At least {n} item(s)",
    maxItems: "At most {n} items",
    uniqueItems: "Duplicate items",
    requiredKey: "« {key} » is required",
    keyPattern: "Key not allowed: {key}",
    unknownKey: "Unexpected key: {key}",
    json: "Invalid JSON",
    types: {
      string: "text",
      integer: "whole number",
      number: "number",
      boolean: "yes / no",
      array: "list",
      object: "object",
    },
  },
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
    account_disabled:
      "This account is disabled: contact an admin of the instance.",
    invalid_invitation:
      "This invitation link is no longer valid (already used, expired or withdrawn).",
    account_exists: "This username or email address is already in use.",
    invalid_username:
      "Username: 2 to 100 letters, digits, « _ », « . » or « - ».",
    invalid_email: "Invalid email address.",
    weak_password: "Password too short (at least 8 characters) or too long.",
    invalid_reset_link:
      "This reset link is no longer valid: ask an admin for a new one.",
    wrong_password: "Current password is incorrect.",
    self_lockout:
      "You cannot disable your own account or remove your own admin role.",
    last_admin: "The instance must keep at least one active admin.",
    invitation_used:
      "This invitation has been used: it can no longer be withdrawn.",
    proposal_state: "This proposal changed state meanwhile: reload the page.",
    proposal_quota: "Too many proposals waiting: wait for their review.",
    proposal_invalid:
      "The proposal fails the catalogue checks (invalid fields).",
    proposal_conflict:
      "Records changed since the proposal: resolve the conflicts.",
    proposal_meta_types: "A proposal changes items and relations only.",
    own_proposal: "Nobody reviews their own proposals.",
  },
};

export default en;
