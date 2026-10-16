# Multi-architecture (CI : linux/amd64 et linux/arm64) : le build Vite produit des fichiers
# statiques, identiques pour toutes les cibles : il tourne sur la plateforme de build (sans émulation).
FROM --platform=$BUILDPLATFORM node:22-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
# Les binaires natifs (rolldown, @tailwindcss/oxide, lightningcss) sont des dépendances optionnelles :
# npm ignore silencieusement un téléchargement raté. On vérifie donc leur présence tout de suite,
# pour échouer ici avec un message clair plutôt qu'au `vite build`. Remède : `docker compose build --no-cache web`.
RUN npm ci --no-audit --no-fund \
 && node --input-type=module -e "await import('rolldown'); await import('@tailwindcss/oxide'); await import('lightningcss')" \
 || { echo "Binaire natif manquant (dépendance optionnelle npm non téléchargée) : relancer le build avec --no-cache"; exit 1; }
COPY . .
RUN npm run build

# Image nginx non-root (uid 101), écoute sur 8080.
FROM nginxinc/nginx-unprivileged:1.28-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY nginx-security-headers.conf /etc/nginx/conf.d/security-headers.inc
EXPOSE 8080
