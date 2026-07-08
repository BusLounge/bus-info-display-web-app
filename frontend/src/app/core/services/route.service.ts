import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from './api.service';

export interface MasterRoute {
  id?: string;
  route_number: string;
  route_name: string;
  origin_city: string;
  destination_city: string;
  total_distance_km: string;
  estimated_duration_minutes: number;
  encoded_polyline: string | null;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface RouteSegment {
  index: number;
  from: { lat: number; lng: number };
  to: { lat: number; lng: number };
  distance: number;
  bearing: number;
  distanceFormatted: string;
  bearingFormatted: string;
}

export interface RouteCreateRequest {
  route_number: string;
  route_name: string;
  origin_city: string;
  destination_city: string;
  total_distance_km: string;
  estimated_duration_minutes?: number;
  encoded_polyline: string | null;
  is_active: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class RouteService {
  constructor(private apiService: ApiService) {}

  // Get all routes
  getAllRoutes(): Observable<MasterRoute[]> {
    return this.apiService.get<MasterRoute[]>('/routes');
  }

  // Get single route by ID
  getRouteById(id: string): Observable<MasterRoute> {
    return this.apiService.get<MasterRoute>(`/routes/${id}`);
  }

  // Create new route
  createRoute(route: RouteCreateRequest): Observable<MasterRoute> {
    return this.apiService.post<MasterRoute>('/routes', route);
  }

  // Update existing route
  updateRoute(id: string, route: Partial<RouteCreateRequest>): Observable<MasterRoute> {
    return this.apiService.put<MasterRoute>(`/routes/${id}`, route);
  }

  // Delete route
  deleteRoute(id: string): Observable<void> {
    return this.apiService.delete<void>(`/routes/${id}`);
  }

  // Get next route number
  getNextRouteNumber(routes: MasterRoute[]): string {
    if (!routes || routes.length === 0) {
      return '1';
    }
    
    const maxNumber = routes.reduce((max, route) => {
      const num = parseInt(route.route_number) || 0;
      return num > max ? num : max;
    }, 0);
    
    return String(maxNumber + 1);
  }
}
