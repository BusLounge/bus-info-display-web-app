import { Injectable, signal } from '@angular/core';
import { AdPlayback, AdBillingReport, PlaybackStatus } from '../models';

@Injectable({
  providedIn: 'root'
})
export class AdTrackingService {
  private playbacks = signal<AdPlayback[]>([]);

  readonly playbacks$ = this.playbacks.asReadonly();

  constructor() {
    this.loadMockData();
  }

  private loadMockData(): void {
    const mockPlaybacks: AdPlayback[] = [
      {
        id: '1',
        deviceId: '1',
        adId: 'ad_001',
        adName: 'Coffee Shop Special',
        adDuration: 30,
        playbackStartTime: new Date(Date.now() - 3600000),
        playbackEndTime: new Date(Date.now() - 3570000),
        status: PlaybackStatus.COMPLETED,
        impressions: 45,
        completionRate: 98,
        billingAmount: 12.50
      },
      {
        id: '2',
        deviceId: '1',
        adId: 'ad_002',
        adName: 'Travel Insurance',
        adDuration: 45,
        playbackStartTime: new Date(Date.now() - 1800000),
        playbackEndTime: new Date(Date.now() - 1755000),
        status: PlaybackStatus.COMPLETED,
        impressions: 32,
        completionRate: 95,
        billingAmount: 18.75
      }
    ];

    this.playbacks.set(mockPlaybacks);
  }

  getPlaybacksByDevice(deviceId: string): AdPlayback[] {
    return this.playbacks().filter(p => p.deviceId === deviceId);
  }

  generateBillingReport(deviceId: string, startDate: Date, endDate: Date): AdBillingReport {
    const devicePlaybacks = this.getPlaybacksByDevice(deviceId).filter(
      p => p.playbackStartTime >= startDate && p.playbackStartTime <= endDate
    );

    const adsMap = new Map<string, {
      adName: string;
      impressions: number;
      playbackTime: number;
      completionRates: number[];
      billing: number;
    }>();

    devicePlaybacks.forEach(playback => {
      if (!adsMap.has(playback.adId)) {
        adsMap.set(playback.adId, {
          adName: playback.adName,
          impressions: 0,
          playbackTime: 0,
          completionRates: [],
          billing: 0
        });
      }

      const adData = adsMap.get(playback.adId)!;
      adData.impressions += playback.impressions;
      adData.playbackTime += playback.adDuration;
      adData.completionRates.push(playback.completionRate);
      adData.billing += playback.billingAmount || 0;
    });

    const ads = Array.from(adsMap.entries()).map(([adId, data]) => ({
      adId,
      adName: data.adName,
      totalImpressions: data.impressions,
      totalPlaybackTime: data.playbackTime,
      averageCompletionRate: data.completionRates.reduce((a, b) => a + b, 0) / data.completionRates.length,
      billingAmount: data.billing
    }));

    return {
      deviceId,
      deviceName: 'Device Name', // Would be fetched from device service
      totalImpressions: ads.reduce((sum, ad) => sum + ad.totalImpressions, 0),
      totalPlaybackTime: ads.reduce((sum, ad) => sum + ad.totalPlaybackTime, 0),
      totalBillingAmount: ads.reduce((sum, ad) => sum + ad.billingAmount, 0),
      reportPeriod: { startDate, endDate },
      ads
    };
  }
}
