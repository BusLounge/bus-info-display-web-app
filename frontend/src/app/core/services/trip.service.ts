import { Injectable } from '@angular/core';
import { Observable, from, map, of } from 'rxjs';
import { finalize, shareReplay, tap } from 'rxjs/operators';
import { supabase, isSupabaseConfigured } from '../config/supabase.config';
import { MasterRoute } from './route.service';
import { RouteSegment } from '../utils/polyline.utils';

export interface Trip {
  trip_id: string;
  route_id: string;
  route_name: string;
  current_point_index: number;
  current_latitude: number;
  current_longitude: number;
  total_points: number;
  speed_multiplier: number;
  is_animating: boolean;
  progress_percentage: number;
  created_at: string;
  updated_at: string;
}

export interface TripWithRoute {
  trip: Trip;
  decodedPoints: Array<{ lat: number; lng: number }>;
  routeData: MasterRoute | null;
}

export interface RouteWithTrips extends MasterRoute {
  activeTrips?: Trip[];
  tripCount?: number;
  segments?: RouteSegment[];
  decodedPointCount?: number;
  segmentError?: string;
}

@Injectable({
  providedIn: 'root'
})
export class TripService {
  private readonly tripsCacheTtlMs = 15000;
  private tripsCache: { expiresAt: number; value: Trip[] } | null = null;
  private tripsInFlight$: Observable<Trip[]> | null = null;
  
  constructor() {}

  // Get all trips from current_locations table
  getAllTrips(): Observable<Trip[]> {
    if (!isSupabaseConfigured || !supabase) {
      console.warn('Supabase not configured. Returning empty trips array.');
      return of([]);
    }

    const now = Date.now();
    if (this.tripsCache && this.tripsCache.expiresAt > now) {
      return of(this.tripsCache.value);
    }

    if (this.tripsInFlight$) {
      return this.tripsInFlight$;
    }
    
    this.tripsInFlight$ = from(
      supabase
        .from('current_locations')
        .select('*')
        .order('created_at', { ascending: false })
    ).pipe(
      map((response: any) => {
        if (response.error) {
          throw new Error(response.error.message);
        }
        return response.data || [];
      }),
      tap((trips) => {
        this.tripsCache = {
          value: trips,
          expiresAt: Date.now() + this.tripsCacheTtlMs
        };
      }),
      finalize(() => {
        this.tripsInFlight$ = null;
      }),
      shareReplay(1)
    );

    return this.tripsInFlight$;
  }

  // Get trips for a specific route
  getTripsByRouteId(routeId: string): Observable<Trip[]> {
    return this.getAllTrips().pipe(
      map((trips) => trips.filter((trip) => trip.route_id === routeId))
    );
  }

  // Get active (animating) trips
  getActiveTrips(): Observable<Trip[]> {
    return this.getAllTrips().pipe(
      map((trips) => trips.filter((trip) => trip.is_animating))
    );
  }

  // Get trip by ID
  getTripById(tripId: string): Observable<Trip | null> {
    if (!isSupabaseConfigured || !supabase) {
      return of(null);
    }
    
    return from(
      supabase
        .from('current_locations')
        .select('*')
        .eq('trip_id', tripId)
        .single()
    ).pipe(
      map((response: any) => {
        if (response.error) {
          throw new Error(response.error.message);
        }
        return response.data;
      })
    );
  }

  // Create new trip
  createTrip(trip: Partial<Trip>): Observable<Trip> {
    if (!isSupabaseConfigured || !supabase) {
      throw new Error('Supabase not configured. Cannot create trip.');
    }
    
    return from(
      supabase
        .from('current_locations')
        .insert([trip])
        .select()
        .single()
    ).pipe(
      map((response: any) => {
        if (response.error) {
          throw new Error(response.error.message);
        }
        this.clearTripsCache();
        return response.data;
      })
    );
  }

  // Update trip
  updateTrip(tripId: string, updates: Partial<Trip>): Observable<Trip> {
    if (!isSupabaseConfigured || !supabase) {
      throw new Error('Supabase not configured. Cannot update trip.');
    }
    
    return from(
      supabase
        .from('current_locations')
        .update(updates)
        .eq('trip_id', tripId)
        .select()
        .single()
    ).pipe(
      map((response: any) => {
        if (response.error) {
          throw new Error(response.error.message);
        }
        this.clearTripsCache();
        return response.data;
      })
    );
  }

  // Delete trip
  deleteTrip(tripId: string): Observable<void> {
    if (!isSupabaseConfigured || !supabase) {
      throw new Error('Supabase not configured. Cannot delete trip.');
    }
    
    return from(
      supabase
        .from('current_locations')
        .delete()
        .eq('trip_id', tripId)
    ).pipe(
      map((response: any) => {
        if (response.error) {
          throw new Error(response.error.message);
        }
        this.clearTripsCache();
      })
    );
  }

  // Get trip count for a route
  getTripCountByRouteId(routeId: string): Observable<number> {
    return this.getTripsByRouteId(routeId).pipe(map((trips) => trips.length));
  }

  private clearTripsCache(): void {
    this.tripsCache = null;
    this.tripsInFlight$ = null;
  }
}
