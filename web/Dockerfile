# Multi-architecture (CI : linux/amd64 et linux/arm64) : le build Vite produit des fichiers
# statiques, identiques pour toutes les cibles : il tourne sur la plateforme de build (sans émulation).
FROM --platform=$BUILDPLATFORM node:26-alpine@sha256:c6ad9e9c1d80f6eb607c26b4b0a132ff7a2344fad1c78ea3baa2a7d4c8de6d94 AS builder

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
FROM nginxinc/nginx-unprivileged:1.31-alpine@sha256:b5047b9b76845eac124358373c6852377c52e4b05834a47647e955dd91cb8f1e
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY nginx-security-headers.conf /etc/nginx/conf.d/security-headers.inc
# Rendu au démarrage (envsubst de l'entrypoint) en /etc/nginx/conf.d/real-ip.conf : adresse réelle du
# client derrière un reverse-proxy (REAL_IP_FROM = adresse ou réseau du proxy ; défaut : rien n'est relu).
COPY nginx-real-ip.conf.template /etc/nginx/templates/real-ip.conf.template
ENV REAL_IP_FROM=127.0.0.1
EXPOSE 8080
