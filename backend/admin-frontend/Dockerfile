# --- build ---
FROM node:22-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npx ng build --configuration docker

# --- run ---
FROM nginx:1.27-alpine
# The nginx image renders templates with environment variables at startup.
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
ENV PORT=80 \
    API_UPSTREAM=http://api:8083 \
    NGINX_ENTRYPOINT_LOCAL_RESOLVERS=1
COPY --from=build /src/dist/bus-schedule-lounge/browser /usr/share/nginx/html
EXPOSE 80
