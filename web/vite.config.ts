/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    // Application installable (lot E5a) : le service worker garde l'application (JS, CSS, polices
    // latines, icônes) pour qu'elle s'ouvre sans réseau. Aucune réponse de l'API n'est mise en cache
    // ici : les données hors ligne vivent dans IndexedDB (lib/offline). Mise à jour proposée à
    // l'utilisateur (registerType « prompt », src/pwa.ts).
    VitePWA({
      registerType: "prompt",
      injectRegister: false,
      includeAssets: ["favicon.svg", "apple-touch-icon.png"],
      manifest: {
        name: "Collectionnard",
        short_name: "Collectionnard",
        description: "Gestion de collections (séries, BD, manga, DVD…)",
        lang: "fr",
        start_url: "/",
        scope: "/",
        display: "standalone",
        background_color: "#eef0eb",
        theme_color: "#f7f8f5",
        icons: [
          { src: "pwa-192x192.png", sizes: "192x192", type: "image/png" },
          { src: "pwa-512x512.png", sizes: "512x512", type: "image/png" },
          {
            src: "maskable-512x512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable",
          },
        ],
      },
      workbox: {
        globPatterns: [
          "**/*.{js,css,html,svg,png}",
          "**/*-latin-*.woff2",
          "**/*-latin-ext-*.woff2",
          "**/primeicons-*.woff2",
          // Lecteur de codes-barres (lot M2) : le wasm de zxing est servi et précaché ici, pas par un CDN.
          "**/*.wasm",
        ],
        navigateFallback: "/index.html",
        navigateFallbackDenylist: [/^\/api\//],
        cleanupOutdatedCaches: true,
      },
    }),
  ],
  // Drapeaux de compilation de vue-i18n : API Composition seule (pas de legacy), pas de devtools en prod.
  define: {
    __VUE_I18N_FULL_INSTALL__: true,
    __VUE_I18N_LEGACY_API__: false,
    __INTLIFY_PROD_DEVTOOLS__: false,
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts"],
    setupFiles: ["tests/setup.ts"],
  },
});
