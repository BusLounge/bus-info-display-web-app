import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from './api.service';

export interface Lounge {
  id: string;
  loungeName: string;
  loungeOwnerId?: string;
  description?: string;
  address?: string;
  city?: string;
  district?: string;
  state?: string;
  country?: string;
  postalCode?: string;
  latitude?: number;
  longitude?: number;
  contactPhone?: string;
  price1Hour?: number;
  price2Hours?: number;
  price3Hours?: number;
  priceUntilBus?: number;
  capacity?: number;
  totalStaff?: number;
  /** JSON array as text from the API, e.g. '["parking","ac"]' */
  amenities?: string | string[] | null;
  /** JSON array of image URLs as text from the API */
  images?: string | string[] | null;
  status?: string;
  isOperational?: boolean;
  averageRating?: number;
  verificationNote?: string;
  marketplaceCategoryId?: string;
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

  getLoungeRouteSegmentValidation(
    id: string,
    forceRefresh: boolean = true,
    suppressLoader: boolean = false
  ): Observable<LoungeRouteValidationResponse> {
    return this.api.get<LoungeRouteValidationResponse>(`/lounges/${id}/route-segment-validation`, {
      forceRefresh,
      suppressLoader
    });
  }
}
