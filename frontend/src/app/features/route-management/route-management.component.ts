import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { forkJoin } from 'rxjs';
import { HeaderComponent } from '../../shared/components/header/header.component';
import { RouteService, MasterRoute } from '../../core/services/route.service';
import { TripService, Trip, RouteWithTrips } from '../../core/services/trip.service';
import { generateSegments, RouteSegment, decodePolyline } from '../../core/utils/polyline.utils';

@Component({
  selector: 'app-route-management',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule, HeaderComponent],
  templateUrl: './route-management.component.html',
  styleUrl: './route-management.component.scss'
})
export class RouteManagementComponent implements OnInit {
  routes: RouteWithTrips[] = [];
  filteredRoutes: RouteWithTrips[] = [];
  searchTerm: string = '';
  isLoading: boolean = false;
  error: string | null = null;
  allTrips: Trip[] = [];
  showTripsPanel: boolean = false;
  selectedRoute: RouteWithTrips | null = null;
  expandedSegments: Set<string> = new Set(); // Track which routes have expanded segments

  constructor(
    private routeService: RouteService,
    private tripService: TripService
  ) {}

  ngOnInit(): void {
    this.loadRoutes();
  }

  loadRoutes(): void {
    this.isLoading = true;
    this.error = null;

    // Fetch both routes and trips simultaneously
    forkJoin({
      routes: this.routeService.getAllRoutes(),
      trips: this.tripService.getAllTrips()
    }).subscribe({
      next: (data: { routes: MasterRoute[], trips: Trip[] }) => {
        this.allTrips = data.trips;
        
        // Merge trip data with route data and decode polylines
        this.routes = data.routes.map((route: MasterRoute) => {
          const routeTrips = data.trips.filter((trip: Trip) => trip.route_id === route.id);
          
          // Decode polyline and generate segments
          let segments: RouteSegment[] = [];
          let decodedPointCount = 0;
          let segmentError = '';

          if (route.encoded_polyline) {
            try {
              const points = decodePolyline(route.encoded_polyline);
              decodedPointCount = points.length;

              if (points.length >= 2) {
                segments = generateSegments(points);
              } else {
                segmentError = 'Polyline has fewer than 2 points.';
              }
            } catch (e) {
              console.error('Error decoding polyline for route', route.id, e);
              segmentError = 'Invalid encoded polyline.';
            }
          } else {
            segmentError = 'No encoded polyline saved for this route.';
          }
          
          return {
            ...route,
            activeTrips: routeTrips,
            tripCount: routeTrips.length,
            segments,
            decodedPointCount,
            segmentError
          } as RouteWithTrips;
        });
        
        this.filteredRoutes = this.routes;
        this.isLoading = false;
      },
      error: (err: any) => {
        console.error('Error loading routes and trips:', err);
        this.error = 'Failed to load routes. Please try again.';
        this.isLoading = false;
      }
    });
  }

  filterRoutes(): void {
    const term = this.searchTerm.toLowerCase();
    this.filteredRoutes = this.routes.filter((route: RouteWithTrips) =>
      route.route_name.toLowerCase().includes(term) ||
      route.route_number.toLowerCase().includes(term) ||
      route.origin_city.toLowerCase().includes(term) ||
      route.destination_city.toLowerCase().includes(term)
    );
  }

  deleteRoute(id: string): void {
    if (!confirm('Are you sure you want to delete this route?')) {
      return;
    }

    this.routeService.deleteRoute(id).subscribe({
      next: () => {
        this.loadRoutes();
      },
      error: (err: any) => {
        console.error('Error deleting route:', err);
        alert('Failed to delete route.');
      }
    });
  }

  toggleRouteStatus(route: RouteWithTrips): void {
    this.routeService.updateRoute(route.id!, {
      is_active: !route.is_active
    }).subscribe({
      next: () => {
        route.is_active = !route.is_active;
      },
      error: (err: any) => {
        console.error('Error updating route status:', err);
        alert('Failed to update route status.');
      }
    });
  }

  showTripsForRoute(route: RouteWithTrips): void {
    this.selectedRoute = route;
    this.showTripsPanel = true;
  }

  closeTripsPanel(): void {
    this.showTripsPanel = false;
    this.selectedRoute = null;
  }

  getActiveTripCount(route: RouteWithTrips): number {
    return route.activeTrips?.filter((trip: Trip) => trip.is_animating).length || 0;
  }

  getTripStatusClass(trip: Trip): string {
    return trip.is_animating ? 'animating' : 'paused';
  }

  formatProgress(progress: number): string {
    return progress.toFixed(1);
  }

  toggleSegments(routeId: string | undefined): void {
    if (!routeId) return;
    if (this.expandedSegments.has(routeId)) {
      this.expandedSegments.delete(routeId);
    } else {
      this.expandedSegments.add(routeId);
    }
  }

  isSegmentsExpanded(routeId: string | undefined): boolean {
    return routeId ? this.expandedSegments.has(routeId) : false;
  }

  getRouteTotalDistance(segments: RouteSegment[]): number {
    return segments.reduce((sum, seg) => sum + seg.distance, 0);
  }

  canShowSegments(route: RouteWithTrips): boolean {
    return !!route.encoded_polyline || !!route.segmentError;
  }
}
