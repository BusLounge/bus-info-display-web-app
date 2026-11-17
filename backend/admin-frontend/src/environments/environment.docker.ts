// Used by the Docker image (ng build --configuration docker).
// nginx serves the app and proxies /api and /media to the API container.
export const environment = {
  production: true,
  apiUrl: '/api',
  // Where the kiosk display app (kiosk/kiosk-frontend) is served
  kioskDisplayUrl: 'http://localhost:4200/bids-display'
};
