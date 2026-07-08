import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from './api.service';

export interface Advertisement {
  id?: string;
  advertisementName: string;
  description?: string;
  advertisementCategory: string;
  mediaDuration?: number;
  mediaUrl?: string;
  mediaType?: string;
  loungeGroupName?: string;
  priority: string;
  version?: number;
  scheduleType: string;
  frequency?: string;
  recurrenceInterval?: number;
  occursOnceAt?: string;
  occursEveryInterval?: number;
  weeklyDays?: string;
  monthlyDayOfMonth?: number;
  monthlyWeek?: string;
  monthlyDay?: string;
  startDate?: string;
  endDate?: string;
  startTime?: string;
  endTime?: string;
  playTimeSlot?: string;
  playTimeSlots?: string[];
  maxIdleLoopDuration?: number;
  status?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface AdvertisementGroup {
  id: string;
  groupName: string;
  lounges: string;
  noOfAdvertisements: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface AdvertisementGroupCreateRequest {
  groupName: string;
  lounges: string[];
}

export interface TVAdManifestItem {
  id: string;
  advertisementName: string;
  mediaUrl: string;
  mediaType: string;
  mediaDuration?: number;
  priority: string;
  scheduleType?: string;
  frequency?: string;
  startDate?: string;
  endDate?: string;
  startTime?: string;
  endTime?: string;
  playTimeSlot?: string;
  playTimeSlots?: string[];
  nextPlayAt?: string;
  mediaHash: string;
  updatedAt: string;
}

export interface UploadMediaResponse {
  fileName: string;
  mediaUrl: string;
  mediaType: string;
}

export interface LoungeAdSlotSummary {
  scheduleWindowSeconds: number;
  adWindowSeconds: number;
  bookedSeconds: number;
  remainingSeconds: number;
}

export interface AdvertisementCalculationRate {
  trafficLevel: string;
  costPerSecond: number;
  updatedAt: string;
}

export interface AdvertisementPlaybackLogRequest {
  advertisementId: string;
  advertisementName: string;
  trafficLevel?: 'Peak' | 'Moderate' | 'Off-Peak';
  durationSeconds: number;
  playedAt?: string;
}

export interface AdvertisementPlaybackLog {
  id: number;
  advertisementId: string;
  advertisementName: string;
  trafficLevel: 'Peak' | 'Moderate' | 'Off-Peak' | string;
  durationSeconds: number;
  playedAt: string;
  createdAt: string;
}

export interface AdvertisementPlaybackLogResponse {
  startDate: string;
  endDate: string;
  advertisementId: string;
  trafficLevel: string;
  count: number;
  rows: AdvertisementPlaybackLog[];
}

export interface AdvertisementCostReportRow {
  advertisementId: string;
  advertisementName: string;
  trafficLevel: string;
  playCount: number;
  totalSeconds: number;
  costPerSecond: number;
  totalCost: number;
}

export interface AdvertisementCostReportResponse {
  startDate: string;
  endDate: string;
  rows: AdvertisementCostReportRow[];
}

@Injectable({ providedIn: 'root' })
export class AdvertisementService {
  constructor(private api: ApiService) {}

  uploadMedia(file: File): Observable<UploadMediaResponse> {
    const formData = new FormData();
    formData.append('file', file);
    return this.api.post<UploadMediaResponse>('/advertisements/upload-media', formData);
  }

  // Advertisements
  getAllAdvertisements(): Observable<Advertisement[]> {
    return this.api.get<Advertisement[]>('/advertisements');
  }

  checkAdvertisementConflicts(payload: any): Observable<any> {
    return this.api.post<any>('/advertisements/conflicts', payload);
  }

  getAdvertisementById(id: string): Observable<Advertisement> {
    return this.api.get<Advertisement>(`/advertisements/${id}`);
  }

  createAdvertisement(ad: Advertisement): Observable<Advertisement> {
    return this.api.post<Advertisement>('/advertisements', ad);
  }

  updateAdvertisement(id: string, ad: Advertisement): Observable<any> {
    return this.api.put<any>(`/advertisements/${id}`, ad);
  }

  deleteAdvertisement(id: string): Observable<any> {
    return this.api.delete<any>(`/advertisements/${id}`);
  }

  getTVAdsManifest(loungeOrGroup: string): Observable<TVAdManifestItem[]> {
    return this.api.get<TVAdManifestItem[]>(`/tv/ads/${encodeURIComponent(loungeOrGroup)}`, { forceRefresh: true });
  }

  getLoungeAdSlots(loungeId: string): Observable<LoungeAdSlotSummary> {
    return this.api.get<LoungeAdSlotSummary>(`/lounge-ads/slots/${loungeId}`);
  }

  // Advertisement cost calculation
  getCalculationRates(): Observable<AdvertisementCalculationRate[]> {
    return this.api.get<AdvertisementCalculationRate[]>('/advertisement-calculation/rates');
  }

  upsertCalculationRate(
    trafficLevel: 'Peak' | 'Moderate' | 'Off-Peak',
    costPerSecond: number
  ): Observable<AdvertisementCalculationRate> {
    return this.api.put<AdvertisementCalculationRate>(
      `/advertisement-calculation/rates/${encodeURIComponent(trafficLevel)}`,
      { costPerSecond }
    );
  }

  createPlaybackLog(payload: AdvertisementPlaybackLogRequest): Observable<any> {
    return this.api.post<any>('/advertisement-calculation/logs', payload);
  }

  getPlaybackLogs(filters?: {
    startDate?: string;
    endDate?: string;
    advertisementId?: string;
    trafficLevel?: 'Peak' | 'Moderate' | 'Off-Peak' | 'All';
    limit?: number;
  }): Observable<AdvertisementPlaybackLogResponse> {
    const params = new URLSearchParams();
    if (filters?.startDate) {
      params.set('startDate', filters.startDate);
    }
    if (filters?.endDate) {
      params.set('endDate', filters.endDate);
    }
    if (filters?.advertisementId) {
      params.set('advertisementId', filters.advertisementId);
    }
    if (filters?.trafficLevel && filters.trafficLevel !== 'All') {
      params.set('trafficLevel', filters.trafficLevel);
    }
    if (filters?.limit && Number.isFinite(filters.limit) && filters.limit > 0) {
      params.set('limit', String(filters.limit));
    }

    const query = params.toString();
    const url = query ? `/advertisement-calculation/logs?${query}` : '/advertisement-calculation/logs';
    return this.api.get<AdvertisementPlaybackLogResponse>(url);
  }

  getCostReport(startDate?: string, endDate?: string): Observable<AdvertisementCostReportResponse> {
    const params = new URLSearchParams();
    if (startDate) {
      params.set('startDate', startDate);
    }
    if (endDate) {
      params.set('endDate', endDate);
    }
    const query = params.toString();
    const url = query ? `/advertisement-calculation/report?${query}` : '/advertisement-calculation/report';
    return this.api.get<AdvertisementCostReportResponse>(url);
  }

  // Advertisement Groups
  getAllGroups(suppressLoader: boolean = false): Observable<AdvertisementGroup[]> {
    return this.api.get<AdvertisementGroup[]>('/advertisement-groups', { suppressLoader });
  }

  getGroupById(id: string): Observable<AdvertisementGroup> {
    return this.api.get<AdvertisementGroup>(`/advertisement-groups/${id}`);
  }

  createGroup(group: AdvertisementGroupCreateRequest): Observable<AdvertisementGroup> {
    console.log('[AdvertisementService] createGroup called with:', group);
    console.log('[AdvertisementService] API URL:', '/advertisement-groups');
    return this.api.post<AdvertisementGroup>('/advertisement-groups', group);
  }

  updateGroup(id: string, group: AdvertisementGroupCreateRequest): Observable<any> {
    return this.api.put<any>(`/advertisement-groups/${id}`, group);
  }

  deleteGroup(id: string): Observable<any> {
    return this.api.delete<any>(`/advertisement-groups/${id}`);
  }
}
