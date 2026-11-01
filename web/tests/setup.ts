// Environnement jsdom : certains composants Element Plus interrogent
// window.matchMedia, absent de jsdom.
import { vi } from "vitest";

if (typeof window !== "undefined" && !window.matchMedia) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
}

// jsdom n'implémente pas le canvas 2D : il renvoie null en le signalant sur la console à chaque
// appel (« Not implemented: HTMLCanvasElement's getContext() »). Même résultat, sans le bruit :
// lib/images.ts traite déjà l'absence de contexte.
if (typeof HTMLCanvasElement !== "undefined") {
  HTMLCanvasElement.prototype.getContext = vi
    .fn()
    .mockReturnValue(
      null,
    ) as unknown as typeof HTMLCanvasElement.prototype.getContext;
}

// ElTable / ElSelect observent la taille de leurs conteneurs.
if (typeof window !== "undefined" && !("ResizeObserver" in window)) {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  Object.defineProperty(window, "ResizeObserver", {
    writable: true,
    value: ResizeObserverStub,
  });
}

// vue-i18n : installé sur tous les montages de composants (langue par défaut : fr).
import { config } from "@vue/test-utils";
import { i18n } from "@/i18n";

config.global.plugins = [...(config.global.plugins ?? []), i18n];

// jsdom annonce `en-US` : on fige la langue des tests sur le français (useLang lit
// d'abord la préférence mémorisée).
try {
  localStorage.setItem("colec-lang", "fr");
} catch {
  /* stockage indisponible */
}
