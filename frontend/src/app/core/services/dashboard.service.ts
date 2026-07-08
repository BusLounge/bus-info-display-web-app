import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { HttpClient } from '@angular/common/http';
import { ApiService } from './api.service';

export interface DashboardStats {
  todayDepartures: number;
  todayArrivals: number;
  delayedArrivals: number;
  delayedDepartures: number;
}

export interface RouteStats {
  route: string;
  count?: number;
  lounges: number;
}

export interface AdvertisementCategoryStats {
  category: string;
  count: number;
}

export interface AdvertisementStatusStats {
  label: string;
  value: number;
  percentage: number;
  color: string;
}

export interface MediaCategoryStats {
  label: string;
  value: number;
  percentage: number;
  color: string;
}

export interface DashboardResponse {
  stats: DashboardStats;
  routesPerLounges: RouteStats[];
  advertisementCategories: AdvertisementCategoryStats[];
  advertisementStatus: AdvertisementStatusStats[];
  mediaCategories: MediaCategoryStats[];
}

export interface BroadcastMessage {
  id: string;
  message: string;
  priority: string;
  displayDurationSeconds: number;
  frequencySeconds: number;
  startAt: string;
  endAt?: string;
  isActive: boolean;
  showOnLoungeTV: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface BroadcastMessageRequest {
  message: string;
  priority: string;
  displayDurationSeconds: number;
  frequencySeconds: number;
  startAt?: string;
  endAt?: string;
  isActive: boolean;
  showOnLoungeTV: boolean;
}

@Injectable({ providedIn: 'root' })
export class DashboardService {
  private readonly localBridgeBaseUrl = 'http://localhost:3001/local';

  constructor(
    private api: ApiService,
    private http: HttpClient
  ) {}

  getDashboardData(): Observable<DashboardResponse> {
    return this.api.get<DashboardResponse>('/dashboard');
  }

  getBroadcastMessages(): Observable<BroadcastMessage[]> {
    return this.api.get<BroadcastMessage[]>('/broadcast-messages', { forceRefresh: true });
  }

  createBroadcastMessage(payload: BroadcastMessageRequest): Observable<BroadcastMessage> {
    return this.api.post<BroadcastMessage>('/broadcast-messages', payload);
  }

  updateBroadcastMessage(id: string, payload: BroadcastMessageRequest): Observable<BroadcastMessage> {
    return this.api.put<BroadcastMessage>(`/broadcast-messages/${id}`, payload);
  }

  deleteBroadcastMessage(id: string): Observable<{ message: string }> {
    return this.api.delete<{ message: string }>(`/broadcast-messages/${id}`);
  }

  setBroadcastsEnabled(enabled: boolean): Observable<{ enabled: boolean }> {
    console.log('[DEBUG] Calling local bridge to set broadcasts enabled:', enabled);
    return this.http.post<{ enabled: boolean }>(
      `${this.localBridgeBaseUrl}/broadcasts-enabled`,
      { enabled }
    );
  }

  getBroadcastsEnabledState(): Observable<{ enabled: boolean }> {
    console.log('[DEBUG] Fetching broadcasts enabled state from local bridge');
    return this.http.get<{ enabled: boolean }>(
      `${this.localBridgeBaseUrl}/broadcasts-enabled`
    );
  }
}
