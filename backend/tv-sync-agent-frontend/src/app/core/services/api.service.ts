import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment.development';
import { Device, DeviceConfig, AdPlayback, EmergencyMessage } from '../models';

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

export interface BroadcastSnapshot {
  updatedAt: string;
  sourceUrl: string;
  items: BroadcastMessage[];
}

export interface LocalLoungeAd {
  id: string;
  loungeId?: string;
  advertisementName: string;
  mediaUrl: string;
  mediaType: 'image' | 'video';
  durationSeconds: number;
  priority: string;
  isActive: boolean;
  isDefaultForAll: boolean;
}

export interface LocalLoungeAdCreateRequest {
  advertisementName: string;
  isActive?: boolean;
  isDefaultForAll?: boolean;
}

export interface LocalLoungeAdUpdateRequest {
  advertisementName: string;
  isActive?: boolean;
  isDefaultForAll?: boolean;
}

export interface LocalLoungeAdTimeSlot {
  type: string;
  label: string;
  startSecond: number;
  endSecond: number;
  durationSeconds: number;
  interactive: boolean;
}

export interface LocalLoungeAdSlotSummary {
  scheduleWindowSeconds: number;
  adWindowSeconds: number;
  bookedSeconds: number;
  remainingSeconds: number;
  timeSlots?: LocalLoungeAdTimeSlot[];
  availableSlots?: LocalLoungeAdTimeSlot[];
  fallbackRequired: boolean;
  fallbackSeconds: number;
}

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private apiUrl = environment.apiUrl;

  constructor(private http: HttpClient) {}

  // Device APIs
  getDevices(): Observable<Device[]> {
    return this.http.get<Device[]>(`${this.apiUrl}/devices`);
  }

  getDevice(id: string): Observable<Device> {
    return this.http.get<Device>(`${this.apiUrl}/devices/${id}`);
  }

  updateDeviceConfig(deviceId: string, config: Partial<DeviceConfig>): Observable<Device> {
    return this.http.patch<Device>(`${this.apiUrl}/devices/${deviceId}/config`, config);
  }

  refreshDeviceStatus(deviceId: string): Observable<Device> {
    return this.http.post<Device>(`${this.apiUrl}/devices/${deviceId}/refresh`, {});
  }

  // Emergency Message APIs
  sendEmergencyMessage(deviceId: string, message: Omit<EmergencyMessage, 'id'>): Observable<EmergencyMessage> {
    return this.http.post<EmergencyMessage>(`${this.apiUrl}/devices/${deviceId}/emergency-message`, message);
  }

  clearEmergencyMessage(deviceId: string): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/devices/${deviceId}/emergency-message`);
  }

  // Ad Tracking APIs
  getAdPlaybacks(deviceId: string, startDate?: Date, endDate?: Date): Observable<AdPlayback[]> {
    let url = `${this.apiUrl}/devices/${deviceId}/ad-playbacks`;
    const params: any = {};

    if (startDate) params.startDate = startDate.toISOString();
    if (endDate) params.endDate = endDate.toISOString();

    return this.http.get<AdPlayback[]>(url, { params });
  }

  generateBillingReport(deviceId: string, startDate: Date, endDate: Date): Observable<any> {
    return this.http.post<any>(`${this.apiUrl}/devices/${deviceId}/billing-report`, {
      startDate: startDate.toISOString(),
      endDate: endDate.toISOString()
    });
  }

  // Health check
  healthCheck(): Observable<{ status: string; version: string }> {
    return this.http.get<{ status: string; version: string }>(`${this.apiUrl}/health`);
  }

  // Local Bridge APIs (TV Sync Agent Go)
  getAgentStatus(): Observable<AgentStatus> {
    return this.http.get<AgentStatus>(`${this.apiUrl}/status`);
  }

  updateLanguage(language: string): Observable<{ language: string; message: string }> {
    return this.http.post<{ language: string; message: string }>(`${this.apiUrl}/language`, { language });
  }

  updateDisplayMode(displayMode: string): Observable<{ displayMode: string; message: string }> {
    return this.http.post<{ displayMode: string; message: string }>(`${this.apiUrl}/display-mode`, {
      displayMode,
    });
  }

  updateLayoutMode(layoutMode: string): Observable<{ layoutMode: string; message: string }> {
    return this.http.post<{ layoutMode: string; message: string }>(`${this.apiUrl}/layout-mode`, {
      layoutMode,
    });
  }

  getSchedule(): Observable<any> {
    return this.http.get(`${this.apiUrl}/schedule`);
  }

  getAds(): Observable<any> {
    return this.http.get(`${this.apiUrl}/ads`);
  }

  getBroadcasts(): Observable<BroadcastSnapshot> {
    return this.http.get<BroadcastSnapshot>(`${this.apiUrl}/broadcasts?ts=${Date.now()}`);
  }

  syncBroadcastsNow(): Observable<{ message: string; broadcastCount: number; lastBroadcastSync: string }> {
    return this.http.post<{ message: string; broadcastCount: number; lastBroadcastSync: string }>(
      `${this.apiUrl}/broadcasts/sync`,
      {}
    );
  }

  getLocalLoungeAds(): Observable<any> {
    return this.http.get<any>(`${this.apiUrl}/lounge-ads`);
  }

  getLocalLoungeAdSlots(): Observable<LocalLoungeAdSlotSummary> {
    return this.http.get<LocalLoungeAdSlotSummary>(`${this.apiUrl}/lounge-ads/slots`);
  }

  createLocalLoungeAd(payload: LocalLoungeAdCreateRequest, mediaFile: File): Observable<LocalLoungeAd> {
    const formData = new FormData();
    formData.append('advertisementName', payload.advertisementName);
    formData.append('isActive', String(payload.isActive ?? true));
    formData.append('isDefaultForAll', String(payload.isDefaultForAll ?? false));
    formData.append('mediaFile', mediaFile);
    return this.http.post<LocalLoungeAd>(`${this.apiUrl}/lounge-ads`, formData);
  }

  updateLocalLoungeAd(id: string, payload: LocalLoungeAdUpdateRequest, mediaFile?: File): Observable<LocalLoungeAd> {
    const formData = new FormData();
    formData.append('advertisementName', payload.advertisementName);
    formData.append('isActive', String(payload.isActive ?? true));
    formData.append('isDefaultForAll', String(payload.isDefaultForAll ?? false));
    if (mediaFile) {
      formData.append('mediaFile', mediaFile);
    }
    return this.http.put<LocalLoungeAd>(`${this.apiUrl}/lounge-ads/${id}`, formData);
  }

  deleteLocalLoungeAd(id: string): Observable<{ message: string }> {
    return this.http.delete<{ message: string }>(`${this.apiUrl}/lounge-ads/${id}`);
  }

  getBroadcastsEnabled(): Observable<{ enabled: boolean }> {
    return this.http.get<{ enabled: boolean }>(`${this.apiUrl}/broadcasts-enabled`);
  }

  updateBroadcastsEnabled(enabled: boolean): Observable<{ enabled: boolean; message: string }> {
    return this.http.post<{ enabled: boolean; message: string }>(`${this.apiUrl}/broadcasts-enabled`, {
      enabled,
    });
  }
}

// Agent Status interface (from tv-sync-agent-go)
export interface AgentStatus {
  startedAt: Date;
  tvPurpose: string;
  language: string;
  displayMode: string;
  layoutMode: string;
  broadcastsEnabled: boolean;
  scheduleEnabled: boolean;
  adsEnabled: boolean;
  lastScheduleSync: Date;
  lastAdsSync: Date;
  lastBroadcastSync: Date;
  lastScheduleError?: string;
  lastAdsError?: string;
  lastBroadcastError?: string;
  adsCount: number;
  broadcastCount: number;
  schedulePath: string;
  adsManifestPath: string;
  broadcastPath: string;
}
