import { Injectable, OnDestroy, PLATFORM_ID, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { catchError, tap } from 'rxjs/operators';
import { environment } from '../../../environments/environment';
import { isPlatformBrowser } from '@angular/common';

export interface AdPlaybackLog {
  advertisementId: string;
  advertisementName: string;
  trafficLevel: string;
  durationSeconds: number;
  playedAt?: string;
}

@Injectable({ providedIn: 'root' })
export class AdPlaybackLoggerService implements OnDestroy {
  private readonly platformId = inject(PLATFORM_ID);
  private readonly apiBaseUrl = environment.apiUrl;
  private readonly batchLogsMaxSize = 10;
  private logBatch: AdPlaybackLog[] = [];
  private batchSendInterval: any;
  private destroyed = false;

  constructor(private http: HttpClient) {
    if (isPlatformBrowser(this.platformId)) {
      this.initBatchSender();
    }
  }

  /**
   * Log an advertisement playback event
   * @param ad The advertisement being played
   * @param durationMs Duration in milliseconds
   * @param trafficLevel Current traffic level (Peak, Moderate, Off-Peak)
   */
  logAdPlayback(ad: {
    id?: string;
    name?: string;
    advertisementName?: string;
  }, durationMs: number, trafficLevel: string = 'Moderate'): void {
    if (this.destroyed || !isPlatformBrowser(this.platformId)) {
      return;
    }

    try {
      const durationSeconds = Math.round(durationMs / 1000);
      const log: AdPlaybackLog = {
        advertisementId: ad.id || 'unknown',
        advertisementName: ad.name || ad.advertisementName || 'Unknown Ad',
        trafficLevel: this.normalizeTrafficLevel(trafficLevel),
        durationSeconds: Math.max(1, durationSeconds),
        playedAt: new Date().toISOString()
      };

      console.log('[AD_LOG] Recording playback:', log);
      this.logBatch.push(log);

      // Send immediately if batch is full, otherwise wait for interval
      if (this.logBatch.length >= this.batchLogsMaxSize) {
        this.flushBatch();
      }
    } catch (err) {
      console.error('[AD_LOG_ERROR] Failed to log playback:', err);
    }
  }

  /**
   * Manually flush the batch to backend
   */
  flushBatch(): void {
    if (this.destroyed || !isPlatformBrowser(this.platformId) || !this.logBatch.length) {
      return;
    }

    const batch = [...this.logBatch];
    this.logBatch = [];

    console.log('[AD_LOG] Flushing batch with', batch.length, 'logs');
    
    // Send each log individually to the backend
    batch.forEach(log => {
      this.recordPlaybackLog(log).subscribe({
        next: () => console.log('[AD_LOG] Recorded:', log.advertisementName),
        error: (err) => console.error('[AD_LOG_ERROR] Failed to record:', err)
      });
    });
  }

  /**
   * Record a single playback log to the backend
   */
  private recordPlaybackLog(log: AdPlaybackLog): Observable<any> {
    if (this.destroyed || !isPlatformBrowser(this.platformId)) {
      return of(null);
    }

    return this.http.post(
      `${this.apiBaseUrl}/advertisement-calculation/logs`,
      log
    ).pipe(
      tap(() => {
        console.log('[AD_LOG] Successfully recorded:', log.advertisementName);
      }),
      catchError(err => {
        console.error('[AD_LOG_ERROR] Failed to record playback:', err);
        // Return success to prevent error cascading
        return of(null);
      })
    );
  }

  /**
   * Normalize traffic level to standard format
   */
  private normalizeTrafficLevel(level: string): string {
    const normalized = (level || '').toLowerCase().trim();
    if (normalized.includes('peak')) return 'Peak';
    if (normalized.includes('moderate')) return 'Moderate';
    if (normalized.includes('off')) return 'Off-Peak';
    return 'Moderate'; // Default
  }

  /**
   * Initialize batch sender to flush logs every 10 seconds
   */
  private initBatchSender(): void {
    this.batchSendInterval = setInterval(() => {
      if (this.logBatch.length > 0) {
        this.flushBatch();
      }
    }, 10000); // Send every 10 seconds
  }

  /**
   * Cleanup
   */
  ngOnDestroy(): void {
    this.destroyed = true;

    if (this.batchSendInterval) {
      clearInterval(this.batchSendInterval);
      this.batchSendInterval = null;
    }

    // Avoid HTTP calls while injector is tearing down.
    this.logBatch = [];
  }
}
