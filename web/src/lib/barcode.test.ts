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
  resetDetector,
} from "./barcode";

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
