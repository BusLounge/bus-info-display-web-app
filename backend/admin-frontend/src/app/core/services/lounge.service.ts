import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from './api.service';

export interface Lounge {
  id: string;
  loungeName: string;
  description?: string;
  address?: string;
  city?: string;
  district?: string;
  state?: string;
  country?: string;
  contactPhone?: string;
  capacity?: number;
  amenities?: any;
  images?: any[];
  status?: string;
  isOperational?: boolean;
  averageRating?: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface LoungeRouteSegmentValidation {
  loungeRouteId: string;
  masterRouteId: string;
  routeNumber?: string;
  stopBeforeId: string;
  stopBeforeName?: string;
  stopBeforeOrder?: number;
  stopAfterId: string;
  stopAfterName?: string;
  stopAfterOrder?: number;
  isValid: boolean;
  reason: string;
}

export interface LoungeRouteValidationResponse {
  loungeId: string;
  isValid: boolean;
  segments: LoungeRouteSegmentValidation[];
}

@Injectable({ providedIn: 'root' })
export class LoungeService {
  constructor(private api: ApiService) {}

  getAllLounges(suppressLoader: boolean = false): Observable<Lounge[]> {
    return this.api.get<Lounge[]>('/lounges', { suppressLoader });
  }

  getLoungeById(id: string): Observable<Lounge> {
    return this.api.get<Lounge>(`/lounges/${id}`);
  }

  getLoungeRouteSegmentValidation(id: string, forceRefresh: boolean = true): Observable<LoungeRouteValidationResponse> {
    return this.api.get<LoungeRouteValidationResponse>(`/lounges/${id}/route-segment-validation`, { forceRefresh });
  }
}
