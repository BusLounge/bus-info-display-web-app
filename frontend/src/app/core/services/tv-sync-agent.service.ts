import { Injectable } from '@angular/core';
import { Observable, BehaviorSubject, interval } from 'rxjs';
import { map, switchMap } from 'rxjs/operators';
import { ApiService } from './api.service';
import {
  TVSyncAgentConfig,
  DeviceStatus,
  AdPlaybackTracking,
  ScheduleDisplay,
  AdDisplay,
  SyncStatus,
  BillingReport,
  EmergencyMessage,
  LoungeSpecificAd,
  LoungeSpecificAdRequest,
  LoungeAdSlotSummary,
} from '../models/tv-sync-agent.model';

@Injectable({ providedIn: 'root' })
export class TVSyncAgentService {
  private syncStatus$ = new BehaviorSubject<SyncStatus>({
    isSyncing: false,
  });

  private deviceStatus$ = new BehaviorSubject<DeviceStatus | null>(null);

  constructor(private api: ApiService) {
    // Auto-sync every 30 seconds
    this.initAutoSync();
  }

  // Configuration Management
  getAgentConfig(loungeId: string): Observable<TVSyncAgentConfig> {
    return this.api.get<TVSyncAgentConfig>(`/tv-sync/config/${loungeId}`);
  }

  saveAgentConfig(config: TVSyncAgentConfig): Observable<TVSyncAgentConfig> {
    return this.api.post<TVSyncAgentConfig>('/tv-sync/config', config);
  }

  updateAgentConfig(
    loungeId: string,
    config: Partial<TVSyncAgentConfig>
  ): Observable<TVSyncAgentConfig> {
    return this.api.put<TVSyncAgentConfig>(`/tv-sync/config/${loungeId}`, config);
  }

  // Device Management
  getDeviceStatus(deviceId: string): Observable<DeviceStatus> {
    return this.api.get<DeviceStatus>(`/tv-sync/device/${deviceId}/status`);
  }

  getAllDevices(loungeId?: string): Observable<DeviceStatus[]> {
    const url = loungeId ? `/tv-sync/devices?loungeId=${loungeId}` : '/tv-sync/devices';
    return this.api.get<DeviceStatus[]>(url);
  }

  updateDeviceStatus(deviceId: string, status: Partial<DeviceStatus>): Observable<DeviceStatus> {
    return this.api.put<DeviceStatus>(`/tv-sync/device/${deviceId}/status`, status);
  }

  // Emergency Messaging
  setEmergencyMessage(loungeId: string, message: EmergencyMessage): Observable<any> {
    return this.api.post<any>(`/tv-sync/emergency/${loungeId}`, message);
  }

  clearEmergencyMessage(loungeId: string): Observable<any> {
    return this.api.delete<any>(`/tv-sync/emergency/${loungeId}`);
  }

  getEmergencyMessage(loungeId: string): Observable<EmergencyMessage | null> {
    return this.api.get<EmergencyMessage | null>(`/tv-sync/emergency/${loungeId}`);
  }

  // Schedule Display
  getSchedules(loungeId: string): Observable<ScheduleDisplay[]> {
    return this.api.get<ScheduleDisplay[]>(`/tv-sync/schedules/${loungeId}`);
  }

  // Ad Display
  getAdsForDisplay(loungeId: string): Observable<AdDisplay[]> {
    return this.api.get<AdDisplay[]>(`/tv-sync/ads/${loungeId}`);
  }

  // Lounge-specific Ads
  getLoungeAds(loungeId: string): Observable<LoungeSpecificAd[]> {
    return this.api.get<LoungeSpecificAd[]>(`/lounge-ads/lounge/${loungeId}`);
  }

  getLoungeAdSlots(loungeId: string): Observable<LoungeAdSlotSummary> {
    return this.api.get<LoungeAdSlotSummary>(`/lounge-ads/slots/${loungeId}`);
  }

  createLoungeAd(payload: LoungeSpecificAdRequest): Observable<LoungeSpecificAd> {
    return this.api.post<LoungeSpecificAd>('/lounge-ads', payload);
  }

  deleteLoungeAd(id: string): Observable<{ message: string }> {
    return this.api.delete<{ message: string }>(`/lounge-ads/${id}`);
  }

  // Ad Playback Tracking
  trackAdPlayback(tracking: AdPlaybackTracking): Observable<any> {
    return this.api.post<any>('/tv-sync/tracking/playback', tracking);
  }

  getAdPlaybackHistory(
    loungeId: string,
    startDate?: string,
    endDate?: string
  ): Observable<AdPlaybackTracking[]> {
    let url = `/tv-sync/tracking/playback/${loungeId}`;
    const params = [];
    if (startDate) params.push(`startDate=${startDate}`);
    if (endDate) params.push(`endDate=${endDate}`);
    if (params.length) url += `?${params.join('&')}`;

    return this.api.get<AdPlaybackTracking[]>(url);
  }

  // Billing Reports
  generateBillingReport(
    loungeId: string,
    startDate: string,
    endDate: string
  ): Observable<BillingReport> {
    return this.api.post<BillingReport>('/tv-sync/billing/report', {
      loungeId,
      startDate,
      endDate,
    });
  }

  downloadBillingReport(
    loungeId: string,
    startDate: string,
    endDate: string,
    format: 'pdf' | 'csv' | 'excel' = 'pdf'
  ): Observable<Blob> {
    return this.api.get<Blob>(
      `/tv-sync/billing/report/download?loungeId=${loungeId}&startDate=${startDate}&endDate=${endDate}&format=${format}`,
      { responseType: 'blob' as any }
    );
  }

  // Sync Operations
  syncNow(loungeId: string): Observable<any> {
    this.syncStatus$.next({ ...this.syncStatus$.value, isSyncing: true });
    return this.api.post<any>(`/tv-sync/sync/${loungeId}`, {}).pipe(
      map((response) => {
        this.syncStatus$.next({
          isSyncing: false,
          lastSync: new Date(),
          nextSync: new Date(Date.now() + 30000),
        });
        return response;
      })
    );
  }

  getSyncStatus(): Observable<SyncStatus> {
    return this.syncStatus$.asObservable();
  }

  getDeviceStatusStream(): Observable<DeviceStatus | null> {
    return this.deviceStatus$.asObservable();
  }

  updateDeviceStatusStream(status: DeviceStatus): void {
    this.deviceStatus$.next(status);
  }

  // Auto-sync initialization
  private initAutoSync(): void {
    interval(30000).subscribe(() => {
      const status = this.syncStatus$.value;
      if (!status.isSyncing) {
        const nextSync = new Date(Date.now() + 30000);
        this.syncStatus$.next({ ...status, nextSync });
      }
    });
  }

  // Content validation
  validateContent(loungeId: string): Observable<{
    valid: boolean;
    errors?: string[];
    warnings?: string[];
  }> {
    return this.api.get<any>(`/tv-sync/validate/${loungeId}`);
  }

  // Screenshot/Preview
  captureScreenshot(deviceId: string): Observable<{ imageUrl: string }> {
    return this.api.post<{ imageUrl: string }>(`/tv-sync/device/${deviceId}/screenshot`, {});
  }

  // Remote Control
  sendRemoteCommand(
    deviceId: string,
    command: 'reboot' | 'refresh' | 'clear-cache' | 'update'
  ): Observable<any> {
    return this.api.post<any>(`/tv-sync/device/${deviceId}/command`, { command });
  }
}
