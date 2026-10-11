import { AxiosError, type AxiosResponse } from "axios";
import { describe, expect, it } from "vitest";
import { extractApiError } from "./client";

function axiosError(status: number, data: unknown): AxiosError {
  return new AxiosError(
    "Request failed",
    "ERR_BAD_REQUEST",
    undefined,
    undefined,
    {
      status,
      data,
      statusText: "",
      headers: {},
      config: { headers: {} },
    } as AxiosResponse,
  );
}

describe("extractApiError", () => {
  it("renvoie le message APIError et les erreurs par champ (sans le / initial)", () => {
    const e = axiosError(422, {
      error: "validation_error",
      message: "Les attributs ne respectent pas le schéma",
      details: { fields: { "/titre": "required", "/annee": "minimum" } },
    });
    expect(extractApiError(e)).toEqual({
      message: "Les attributs ne respectent pas le schéma",
      fieldErrors: { titre: "required", annee: "minimum" },
    });
  });

  it("n'a pas de fieldErrors sans details", () => {
    const e = axiosError(404, {
      error: "not_found",
      message: "Ressource non trouvée",
    });
    const res = extractApiError(e);
    expect(res.message).toBe("Ressource non trouvée");
    expect(res.fieldErrors).toBeUndefined();
  });

  it("retombe sur le message axios si le corps n'a pas de message", () => {
    const e = axiosError(500, { oops: true });
    expect(extractApiError(e).message).toBe("Request failed");
  });

  it("gère une Error classique et une valeur inconnue", () => {
    expect(extractApiError(new Error("boom")).message).toBe("boom");
    expect(extractApiError("???").message).toBe("Erreur inconnue");
  });
});
