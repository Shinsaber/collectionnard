import { AxiosError, type AxiosResponse } from "axios";
import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "@/i18n";
import { extractApiError, filenameFromDisposition } from "./client";

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
  afterEach(() => {
    i18n.global.locale.value = "fr";
  });

  it("traduit le code d'erreur et renvoie les erreurs par champ (sans le / initial)", () => {
    const e = axiosError(422, {
      error: "validation_error",
      message: "Attributes do not match the schema",
      details: { fields: { "/titre": "required", "/annee": "minimum" } },
    });
    expect(extractApiError(e)).toEqual({
      code: "validation_error",
      message: "Données invalides",
      fieldErrors: { titre: "required", annee: "minimum" },
    });
  });

  it("suit la langue courante et n'a pas de fieldErrors sans details", () => {
    i18n.global.locale.value = "en";
    const e = axiosError(404, {
      error: "not_found",
      message: "Item not found",
    });
    const res = extractApiError(e);
    expect(res.message).toBe("Not found");
    expect(res.fieldErrors).toBeUndefined();
  });

  it("garde le message du serveur pour un code inconnu", () => {
    const e = axiosError(418, { error: "teapot", message: "I'm a teapot" });
    expect(extractApiError(e).message).toBe("I'm a teapot");
  });

  it("retombe sur le message axios si le corps n'a ni code ni message", () => {
    const e = axiosError(500, { oops: true });
    expect(extractApiError(e).message).toBe("Request failed");
  });

  it("gère une Error classique et une valeur inconnue", () => {
    expect(extractApiError(new Error("boom")).message).toBe("boom");
    expect(extractApiError("???").message).toBe("Erreur inconnue");
  });
});

describe("filenameFromDisposition", () => {
  it("lit le nom entre guillemets ou non", () => {
    expect(
      filenameFromDisposition('attachment; filename="items.csv"', "x"),
    ).toBe("items.csv");
    expect(filenameFromDisposition("attachment; filename=a.json", "x")).toBe(
      "a.json",
    );
  });

  it("décode la forme filename* (RFC 5987)", () => {
    expect(
      filenameFromDisposition(
        "attachment; filename*=UTF-8''s%C3%A9ries.csv",
        "x",
      ),
    ).toBe("séries.csv");
  });

  it("repli sans en-tête ou sans nom", () => {
    expect(filenameFromDisposition(undefined, "fallback.csv")).toBe(
      "fallback.csv",
    );
    expect(filenameFromDisposition("attachment", "fallback.csv")).toBe(
      "fallback.csv",
    );
  });
});
