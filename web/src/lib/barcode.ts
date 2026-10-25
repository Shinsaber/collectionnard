// Lecture des codes-barres (lot M2) : `BarcodeDetector` natif quand le navigateur l'offre et sait
// lire l'EAN-13 (Chrome Android), sinon le ponyfill barcode-detector (MIT) sur zxing-wasm (MIT),
// chargé à la demande avec le `.wasm` servi par l'application (jamais un CDN ; précaché par le
// service worker, donc disponible hors ligne).

/** Formats lus : codes commerciaux (ISBN = EAN-13, EAN-8, UPC), codes industriels courants, QR. */
export const FORMATS = [
  "ean_13",
  "ean_8",
  "upc_a",
  "upc_e",
  "code_128",
  "code_39",
  "itf",
  "qr_code",
] as const;

export type BarcodeFormat = (typeof FORMATS)[number];

/** Résultat d'une détection (sous-ensemble de l'API Shape Detection, absente des libs TS). */
export interface DetectedBarcode {
  rawValue: string;
  format: string;
}

export interface BarcodeReader {
  detect(source: ImageBitmapSource): Promise<DetectedBarcode[]>;
}

interface NativeDetectorCtor {
  new (options?: { formats?: string[] }): BarcodeReader;
  getSupportedFormats(): Promise<string[]>;
}

let cached: Promise<BarcodeReader> | null = null;

/** Lecteur partagé (natif ou ponyfill) ; la première création charge le ponyfill si besoin. */
export function createDetector(): Promise<BarcodeReader> {
  cached ??= build().catch((e: unknown) => {
    cached = null; // la prochaine tentative réessaie (réseau revenu, par exemple)
    throw e;
  });
  return cached;
}

async function build(): Promise<BarcodeReader> {
  const native = (globalThis as { BarcodeDetector?: NativeDetectorCtor })
    .BarcodeDetector;
  if (native) {
    try {
      const supported = await native.getSupportedFormats();
      if (supported.includes("ean_13")) {
        return new native({
          formats: FORMATS.filter((f) => supported.includes(f)),
        });
      }
    } catch {
      // Implémentation native inutilisable (Chrome sans Google Play Services) : ponyfill.
    }
  }
  const [{ BarcodeDetector, prepareZXingModule }, { default: wasmUrl }] =
    await Promise.all([
      import("barcode-detector/ponyfill"),
      import("zxing-wasm/reader/zxing_reader.wasm?url"),
    ]);
  prepareZXingModule({
    overrides: {
      locateFile: (path: string, prefix: string) =>
        path.endsWith(".wasm") ? wasmUrl : prefix + path,
    },
  });
  return new BarcodeDetector({ formats: [...FORMATS] });
}

/** Réinitialise le lecteur partagé (tests). */
export function resetDetector() {
  cached = null;
}

/**
 * Confirme une lecture : un code 1D n'est retenu que lu deux fois de suite (une image floue
 * peut donner une valeur fausse malgré la clé de contrôle), un QR dès la première lecture.
 * `cooldown` : un code déjà émis est ignoré pendant `cooldownMs` (mode continu).
 */
export class ScanConfirmer {
  private last = "";
  private emitted = new Map<string, number>();

  constructor(private readonly cooldownMs = 1500) {}

  /** Vrai si `value` est confirmée et peut être émise maintenant. */
  push(value: string, format: string, now = Date.now()): boolean {
    const v = value.trim();
    if (!v) return false;
    const confirmed = format === "qr_code" || v === this.last;
    this.last = v;
    if (!confirmed) return false;
    const at = this.emitted.get(v);
    if (at !== undefined && now - at < this.cooldownMs) return false;
    this.emitted.set(v, now);
    this.last = "";
    return true;
  }

  reset() {
    this.last = "";
    this.emitted.clear();
  }
}
