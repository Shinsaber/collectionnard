FROM node:22-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Image nginx non-root (uid 101), écoute sur 8080.
FROM nginxinc/nginx-unprivileged:1.28-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY nginx-security-headers.conf /etc/nginx/conf.d/security-headers.inc
EXPOSE 8080
