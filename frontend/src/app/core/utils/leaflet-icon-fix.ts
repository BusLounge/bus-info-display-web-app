/* Leaflet icon fix - Required for markers to display correctly */
import { isPlatformBrowser } from '@angular/common';

// Only run in browser environment
if (typeof window !== 'undefined') {
  import('leaflet').then((L) => {
    const leaflet = L.default || L;
    
    // Fix Leaflet default marker icon paths for webpack/Angular
    delete (leaflet.Icon.Default.prototype as any)._getIconUrl;

    leaflet.Icon.Default.mergeOptions({
      iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon-2x.png',
      iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon.png',
      shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-shadow.png',
    });
  });
}
