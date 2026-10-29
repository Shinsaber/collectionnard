import { afterEach, describe, expect, it, vi } from "vitest";

const ponyfill = vi.hoisted(() => ({
  ctor: vi.fn(),
  prepare: vi.fn(),
}));
vi.mock("barcode-detector/ponyfill", () => ({
  BarcodeDetector: class {
    constructor(opts: unknown) {
      ponyfill.ctor(opts);
    }
    detect() {
      return Promise.resolve([]);
    }
  },
  prepareZXingModule: ponyfill.prepare,
}));
vi.mock("zxing-wasm/reader/zxing_reader.wasm?url", () => ({
  default: "/assets/zxing_reader-abc.wasm",
}));

import {
  FORMATS,
  ScanConfirmer,
  createDetector,
  identifierPrefill,
  resetDetector,
} from "./barcode";
import fixtures from "../../tests/fixtures/seed-schemas.json";

type G = typeof globalThis & { BarcodeDetector?: unknown };

afterEach(() => {
  delete (globalThis as G).BarcodeDetector;
  resetDetector();
  vi.clearAllMocks();
});

describe("createDetector", () => {
  it("prend l'implémentation native quand elle lit l'EAN-13, limitée aux formats qu'elle connaît", async () => {
    const seen: unknown[] = [];
    (globalThis as G).BarcodeDetector = class {
      constructor(opts: unknown) {
        seen.push(opts);
      }
      static getSupportedFormats() {
        return Promise.resolve(["ean_13", "qr_code", "pdf417"]);
      }
      detect() {
        return Promise.resolve([]);
      }
    };
    await createDetector();
    expect(seen).toEqual([{ formats: ["ean_13", "qr_code"] }]);
    expect(ponyfill.ctor).not.toHaveBeenCalled();
  });

  it("passe au ponyfill (wasm local) sans implémentation native ou sans EAN-13", async () => {
    (globalThis as G).BarcodeDetector = class {
      static getSupportedFormats() {
        return Promise.resolve(["qr_code"]);
      }
    };
    await createDetector();
    expect(ponyfill.ctor).toHaveBeenCalledWith({ formats: [...FORMATS] });
    const overrides = ponyfill.prepare.mock.calls[0]![0].overrides;
    expect(overrides.locateFile("zxing_reader.wasm", "https://cdn/")).toBe(
      "/assets/zxing_reader-abc.wasm",
    );
    expect(overrides.locateFile("other.js", "https://cdn/")).toBe(
      "https://cdn/other.js",
    );
  });

  it("partage un seul lecteur", async () => {
    const a = await createDetector();
    const b = await createDetector();
    expect(a).toBe(b);
    expect(ponyfill.ctor).toHaveBeenCalledTimes(1);
  });
});

describe("identifierPrefill", () => {
  const schemaOf = (name: string) =>
    fixtures.find((f) => f.name === name)!.schema as Record<string, unknown>;

  it("remplit l'ISBN d'un tome avec un EAN-13 en 978, pas avec un autre code", () => {
    expect(identifierPrefill(schemaOf("Volume"), "9782505000945")).toEqual({
      isbn: "9782505000945",
    });
    expect(identifierPrefill(schemaOf("Volume"), "5099902988216")).toEqual({});
  });

  it("remplit le code-barres d'une édition musicale et l'EAN d'un jeu", () => {
    expect(identifierPrefill(schemaOf("Release"), "5099902988216")).toEqual({
      barcode: "5099902988216",
    });
    expect(identifierPrefill(schemaOf("GameRelease"), "045496590475")).toEqual({
      ean: "045496590475",
    });
  });

  it("ignore les identifiants imbriqués, typés autrement ou avec un format, et les schémas vides", () => {
    const schema = {
      properties: {
        qid: { type: "string", "x-identifier": true, pattern: "^Q[0-9]+$" },
        url: { type: "string", "x-identifier": true, format: "uri" },
        free: { type: "string", "x-identifier": true },
        n: { type: "integer", "x-identifier": true },
        bad: { type: "string", "x-identifier": true, pattern: "(" },
        external_ids: {
          type: "object",
          properties: { wikidata: { type: "string", "x-identifier": true } },
        },
      },
    };
    expect(identifierPrefill(schema, "123")).toEqual({ free: "123" });
    expect(identifierPrefill(null, "123")).toEqual({});
    expect(identifierPrefill(schema, "")).toEqual({});
  });
});

describe("ScanConfirmer", () => {
  it("retient un code 1D lu deux fois de suite seulement", () => {
    const c = new ScanConfirmer();
    expect(c.push("9782505000945", "ean_13", 0)).toBe(false);
    expect(c.push("9782505000946", "ean_13", 10)).toBe(false); // lecture différente
    expect(c.push("9782505000946", "ean_13", 20)).toBe(true);
  });

  it("retient un QR dès la première lecture et ignore les valeurs vides", () => {
    const c = new ScanConfirmer();
    expect(c.push("  ", "qr_code", 0)).toBe(false);
    expect(c.push("https://x", "qr_code", 0)).toBe(true);
  });

  it("n'émet pas deux fois le même code pendant le délai de garde", () => {
    const c = new ScanConfirmer(1000);
    expect(c.push("1", "ean_8", 0)).toBe(false);
    expect(c.push("1", "ean_8", 10)).toBe(true);
    expect(c.push("1", "ean_8", 500)).toBe(false);
    expect(c.push("1", "ean_8", 600)).toBe(false); // confirmé mais en garde
    expect(c.push("1", "ean_8", 1100)).toBe(true); // garde levée, code toujours en vue
    expect(c.push("1", "ean_8", 1200)).toBe(false);
    c.reset();
    expect(c.push("1", "ean_8", 1300)).toBe(false);
    expect(c.push("1", "ean_8", 1400)).toBe(true);
  });
});
