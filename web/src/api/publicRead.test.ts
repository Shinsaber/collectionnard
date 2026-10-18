import axios, {
  AxiosError,
  type AxiosAdapter,
  type InternalAxiosRequestConfig,
} from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { collectionApi, httpForTests, publicApi } from "./client";
import { getAccessToken, setTokens } from "./tokens";

// Requêtes vues par l'adaptateur simulé : en-tête Authorization de chacune.
let seen: (string | undefined)[] = [];

function respond(config: InternalAxiosRequestConfig, status: number) {
  const response = {
    status,
    statusText: "",
    data: status === 200 ? { id: "c1" } : { error: "unauthorized" },
    headers: {},
    config,
  };
  if (status === 200) return Promise.resolve(response);
  return Promise.reject(
    new AxiosError("fail", "ERR_BAD_REQUEST", config, null, response),
  );
}

/** Refuse tout jeton (401), répond 200 sans jeton. */
const adapter: AxiosAdapter = (config) => {
  const auth = config.headers?.Authorization as string | undefined;
  seen.push(auth);
  return respond(config, auth ? 401 : 200);
};

describe("lectures publiques", () => {
  const original = httpForTests.defaults.adapter;
  const href = window.location.href;

  beforeEach(() => {
    seen = [];
    httpForTests.defaults.adapter = adapter;
    vi.spyOn(axios, "post").mockRejectedValue(new Error("refresh refusé"));
    setTokens("vieux-jeton", "vieux-refresh");
  });

  afterEach(() => {
    httpForTests.defaults.adapter = original;
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("rejoue en anonyme quand la session ne se renouvelle pas, sans renvoyer vers /login", async () => {
    await expect(publicApi.collection("c1")).resolves.toEqual({ id: "c1" });
    expect(seen).toEqual(["Bearer vieux-jeton", undefined]);
    expect(getAccessToken()).toBeNull();
    expect(window.location.href).toBe(href);
  });

  it("une route privée échoue toujours (pas de reprise anonyme)", async () => {
    await expect(collectionApi.get("c1")).rejects.toBeInstanceOf(AxiosError);
    expect(seen).toEqual(["Bearer vieux-jeton"]);
  });
});
