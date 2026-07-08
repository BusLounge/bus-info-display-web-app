import { Component, OnInit, OnDestroy, AfterViewInit, PLATFORM_ID, Inject } from '@angular/core';
import { CommonModule, isPlatformBrowser } from '@angular/common';
import { RouterModule, ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { HeaderComponent } from '../../shared/components/header/header.component';
import { RouteService, MasterRoute } from '../../core/services/route.service';
import { decodePolyline, encodePolyline, LatLng, distanceToSegment, calculateDistance } from '../../core/utils/polyline.utils';

type EditModeType = 'add' | 'select' | 'insert' | 'move';

@Component({
  selector: 'app-route-editor',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule, HeaderComponent],
  templateUrl: './route-editor.component.html',
  styleUrl: './route-editor.component.scss'
})
export class RouteEditorComponent implements OnInit, AfterViewInit, OnDestroy {
  // Route Data
  routeId: string | null = null;
  isEditMode: boolean = false;
  routeName: string = '';
  originCity: string = '';
  destinationCity: string = '';
  totalDistance: string = '0';
  isActive: boolean = true;

  // Map Points
  points: LatLng[] = [];
  highlightedPointIndex: number | null = null;
  selectedPoints: Set<number> = new Set();
  
  // Edit Mode
  editModeType: EditModeType = 'add';
  isDragging: boolean = false;
  draggingPointIndex: number | null = null;

  // Map
  private map: any = null;
  private markers: any[] = [];
  private polyline: any = null;
  private midpointMarkers: any[] = [];
  private L: any = null;

  // UI State
  isLoading: boolean = false;
  isSaving: boolean = false;
  error: string | null = null;
  isBrowser: boolean = false;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private routeService: RouteService,
    @Inject(PLATFORM_ID) private platformId: Object
  ) {
    this.isBrowser = isPlatformBrowser(this.platformId);
  }

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.routeId = id;
      this.isEditMode = true;
      this.loadRoute(id);
    }
  }

  async ngAfterViewInit(): Promise<void> {
    if (this.isBrowser) {
      await this.initializeMap();
    }
  }

  ngOnDestroy(): void {
    if (this.map) {
      this.map.remove();
    }
  }

  private async initializeMap(): Promise<void> {
    // Dynamically import Leaflet only in browser
    const L = await import('leaflet');
    this.L = L.default || L;

    // Import icon fix
    await import('../../core/utils/leaflet-icon-fix');

    // Initialize map centered on Sri Lanka (default)
    this.map = this.L.map('map').setView([6.9271, 79.8612], 8);

    // Add OpenStreetMap tile layer
    this.L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19,
      attribution: '© OpenStreetMap contributors'
    }).addTo(this.map);

    // Add map click handler
    this.map.on('click', (e: any) => this.handleMapClick(e));
  }

  private loadRoute(id: string): void {
    this.isLoading = true;
    this.routeService.getRouteById(id).subscribe({
      next: (route: MasterRoute) => {
        this.routeName = route.route_name;
        this.originCity = route.origin_city;
        this.destinationCity = route.destination_city;
        this.totalDistance = route.total_distance_km;
        this.isActive = route.is_active;
        
        // Decode polyline to points
        if (route.encoded_polyline) {
          try {
            this.points = decodePolyline(route.encoded_polyline);
            this.updateMapDisplay();
          } catch (e) {
            console.error('Error decoding polyline:', e);
            this.error = 'Failed to decode route polyline';
          }
        }
        
        this.isLoading = false;
      },
      error: (err: any) => {
        console.error('Error loading route:', err);
        this.error = 'Failed to load route';
        this.isLoading = false;
      }
    });
  }

  private handleMapClick(e: any): void {
    const lat = e.latlng.lat;
    const lng = e.latlng.lng;

    if (this.editModeType === 'add') {
      this.addPoint(lat, lng);
    } else if (this.editModeType === 'insert') {
      this.insertPointNearSegment(lat, lng);
    }
  }

  private addPoint(lat: number, lng: number): void {
    this.points.push({ lat, lng });
    this.updateMapDisplay();
    this.calculateTotalDistance();
  }

  private insertPointNearSegment(lat: number, lng: number): void {
    if (this.points.length < 2) {
      this.addPoint(lat, lng);
      return;
    }

    const clickedPoint: LatLng = { lat, lng };
    let minDistance = Infinity;
    let insertIndex = -1;

    // Find the closest segment
    for (let i = 0; i < this.points.length - 1; i++) {
      const distance = distanceToSegment(clickedPoint, this.points[i], this.points[i + 1]);
      if (distance < minDistance) {
        minDistance = distance;
        insertIndex = i + 1;
      }
    }

    if (insertIndex !== -1) {
      this.points.splice(insertIndex, 0, clickedPoint);
      this.highlightedPointIndex = insertIndex;
      this.updateMapDisplay();
      this.calculateTotalDistance();
    }
  }

  deletePoint(index: number): void {
    this.points.splice(index, 1);
    this.highlightedPointIndex = null;
    this.selectedPoints.delete(index);
    this.updateMapDisplay();
    this.calculateTotalDistance();
  }

  setEditMode(mode: EditModeType): void {
    this.editModeType = mode;
    this.highlightedPointIndex = null;
    if (mode !== 'select') {
      this.selectedPoints.clear();
    }
    this.updateMapDisplay();
  }

  deleteSelectedPoints(): void {
    if (this.selectedPoints.size === 0) return;
    
    if (!confirm(`Delete ${this.selectedPoints.size} selected point(s)?`)) return;

    const sortedIndices = Array.from(this.selectedPoints).sort((a, b) => b - a);
    sortedIndices.forEach(index => {
      this.points.splice(index, 1);
    });

    this.selectedPoints.clear();
    this.highlightedPointIndex = null;
    this.updateMapDisplay();
    this.calculateTotalDistance();
  }

  clearAllPoints(): void {
    if (!confirm('Are you sure you want to clear all points?')) return;
    
    this.points = [];
    this.highlightedPointIndex = null;
    this.selectedPoints.clear();
    this.updateMapDisplay();
    this.calculateTotalDistance();
  }

  highlightPoint(index: number): void {
    this.highlightedPointIndex = index;
    this.updateMapDisplay();

    // Center map on highlighted point
    if (this.map) {
      this.map.flyTo([this.points[index].lat, this.points[index].lng], 15);
    }
  }

  togglePointSelection(index: number): void {
    if (this.selectedPoints.has(index)) {
      this.selectedPoints.delete(index);
    } else {
      this.selectedPoints.add(index);
    }
    this.updateMapDisplay();
  }

  private updateMapDisplay(): void {
    if (!this.map || !this.L) return;

    // Clear existing markers and polyline
    this.markers.forEach(m => m.remove());
    this.midpointMarkers.forEach(m => m.remove());
    this.markers = [];
    this.midpointMarkers = [];
    
    if (this.polyline) {
      this.polyline.remove();
      this.polyline = null;
    }

    if (this.points.length === 0) return;

    // Draw polyline
    const latLngs: any[] = this.points.map(p => [p.lat, p.lng]);
    this.polyline = this.L.polyline(latLngs, {
      color: '#8B5CF6',
      weight: 3,
      opacity: 0.7
    }).addTo(this.map);

    // Draw point markers
    this.points.forEach((point, index) => {
      const isHighlighted = this.highlightedPointIndex === index;
      const isSelected = this.selectedPoints.has(index);
      
      const color = isHighlighted ? '#FCD34D' : isSelected ? '#3B82F6' : '#8B5CF6';
      const radius = isHighlighted ? 8 : isSelected ? 7 : 6;

      const marker = this.L.circleMarker([point.lat, point.lng], {
        radius: radius,
        fillColor: color,
        color: 'white',
        weight: 2,
        fillOpacity: 1
      }).addTo(this.map!);

      marker.on('click', (e: any) => {
        this.L.DomEvent.stopPropagation(e);
        if (this.editModeType === 'select') {
          this.togglePointSelection(index);
        } else if (this.editModeType === 'add') {
          this.deletePoint(index);
        } else {
          this.highlightPoint(index);
        }
      });

      // Enable dragging in move mode using mousedown/mousemove/mouseup
      if (this.editModeType === 'move') {
        let isDragging = false;
        
        marker.on('mousedown', () => {
          isDragging = true;
          this.draggingPointIndex = index;
        });

        this.map!.on('mousemove', (e: any) => {
          if (isDragging && this.draggingPointIndex === index) {
            this.points[index] = { lat: e.latlng.lat, lng: e.latlng.lng };
            marker.setLatLng(e.latlng);
            this.updatePolylineOnly();
          }
        });

        this.map!.on('mouseup', () => {
          if (isDragging && this.draggingPointIndex === index) {
            isDragging = false;
            this.draggingPointIndex = null;
            this.calculateTotalDistance();
          }
        });
      }

      this.markers.push(marker);
    });

    // Draw midpoint markers in insert mode
    if (this.editModeType === 'insert' && this.points.length > 1) {
      for (let i = 0; i < this.points.length - 1; i++) {
        const midLat = (this.points[i].lat + this.points[i + 1].lat) / 2;
        const midLng = (this.points[i].lng + this.points[i + 1].lng) / 2;

        const midMarker = this.L.circleMarker([midLat, midLng], {
          radius: 6,
          fillColor: '#10B981',
          color: 'white',
          weight: 2,
          fillOpacity: 0.6,
          dashArray: '5, 5'
        }).addTo(this.map!);

        this.midpointMarkers.push(midMarker);
      }
    }

    // Fit bounds if we have points
    if (this.points.length > 0) {
      const bounds = this.L.latLngBounds(latLngs);
      this.map.fitBounds(bounds, { padding: [50, 50] });
    }
  }

  private updatePolylineOnly(): void {
    if (!this.polyline) return;
    
    const latLngs: any[] = this.points.map(p => [p.lat, p.lng]);
    this.polyline.setLatLngs(latLngs);
  }

  private calculateTotalDistance(): void {
    if (this.points.length < 2) {
      this.totalDistance = '0';
      return;
    }

    let total = 0;
    for (let i = 0; i < this.points.length - 1; i++) {
      total += calculateDistance(this.points[i], this.points[i + 1]);
    }

    this.totalDistance = total.toFixed(2);
  }

  saveRoute(): void {
    if (!this.validateForm()) return;

    this.isSaving = true;
    this.error = null;

    const encodedPolyline = encodePolyline(this.points);

    const routeData = {
      route_number: this.routeId ? undefined : this.getNextRouteNumber(),
      route_name: this.routeName,
      origin_city: this.originCity,
      destination_city: this.destinationCity,
      total_distance_km: this.totalDistance,
      encoded_polyline: encodedPolyline,
      is_active: this.isActive
    };

    const request = this.routeId
      ? this.routeService.updateRoute(this.routeId, routeData)
      : this.routeService.createRoute({
          route_number: routeData.route_number!,
          route_name: routeData.route_name,
          origin_city: routeData.origin_city,
          destination_city: routeData.destination_city,
          total_distance_km: routeData.total_distance_km,
          encoded_polyline: routeData.encoded_polyline,
          is_active: routeData.is_active
        });

    request.subscribe({
      next: () => {
        this.isSaving = false;
        alert(this.routeId ? 'Route updated successfully!' : 'Route created successfully!');
        this.router.navigate(['/route-management']);
      },
      error: (err: any) => {
        console.error('Error saving route:', err);
        this.error = 'Failed to save route. Please try again.';
        this.isSaving = false;
      }
    });
  }

  private validateForm(): boolean {
    if (!this.routeName.trim()) {
      alert('Please enter a route name');
      return false;
    }
    if (!this.originCity.trim()) {
      alert('Please enter origin city');
      return false;
    }
    if (!this.destinationCity.trim()) {
      alert('Please enter destination city');
      return false;
    }
    if (this.points.length < 2) {
      alert('Please add at least 2 points to the route');
      return false;
    }
    return true;
  }

  private getNextRouteNumber(): string {
    // This would ideally come from the backend or a service
    return new Date().getTime().toString().slice(-6);
  }

  goBack(): void {
    if (confirm('Discard unsaved changes?')) {
      this.router.navigate(['/route-management']);
    }
  }

  focusAllPoints(): void {
    if (!this.map || !this.L || this.points.length === 0) return;

    const latLngs: any[] = this.points.map(p => [p.lat, p.lng]);
    const bounds = this.L.latLngBounds(latLngs);
    this.map.fitBounds(bounds, { padding: [50, 50] });
  }
}
