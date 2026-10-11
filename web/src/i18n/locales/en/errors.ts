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
  },
};

export default en;
