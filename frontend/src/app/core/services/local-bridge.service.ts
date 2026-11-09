import { Injectable } from '@angular/core';
import { from, Observable, of } from 'rxjs';
import { HttpClient } from '@angular/common/http';
import { catchError } from 'rxjs/operators';

export const LOCAL_BRIDGE_BASE_URL = 'http://localhost:3001/local';

// Define the shape of the Electron API that will be exposed on the window object
interface IElectronAPI {
  invoke: (channel: string, ...args: any[]) => Promise<any>;
  getAds: () => Promise<any>;
  getBroadcasts: () => Promise<any>;
  getLoungeAds: () => Promise<any>;
  onScheduleSnapshotUpdated?: (callback: () => void) => () => void;
}

declare global {
  interface Window {
    electronAPI?: IElectronAPI;
  }
}

@Injectable({
  providedIn: 'root',
})
export class LocalBridgeService {
  private ipc?: IElectronAPI;

  constructor(private http: HttpClient) {
    if (typeof window !== 'undefined') {
      this.ipc = window.electronAPI;
    }
  }

  get<T>(endpoint: string): Observable<T> {
    const url = `${LOCAL_BRIDGE_BASE_URL}/${endpoint}`;

    if (this.ipc) {
      // Electron IPC path
      console.log(`[BridgeService] Using IPC for endpoint: ${endpoint}`);
      let request: Promise<T>;
      switch (endpoint) {
        case 'ads':
          request = this.ipc.getAds() as Promise<T>;
          break;
        case 'broadcasts':
          request = this.ipc.getBroadcasts() as Promise<T>;
          break;
        case 'lounge-ads':
          request = this.ipc.getLoungeAds() as Promise<T>;
          break;
        default:
          request = this.ipc.invoke('bridge:get', endpoint) as Promise<T>;
      }

      return from(request).pipe(
        catchError(error => {
          console.error(`[BridgeService] IPC fallback failed for ${endpoint}; trying ${url}:`, error);
          return this.getOverHttp<T>(url);
        })
      );
    }

    // Fallback: regular HTTP (for browser / dev mode)
    console.log(`[BridgeService] Using HTTP for endpoint: ${endpoint}`);
    return this.getOverHttp<T>(url);
  }

  private getOverHttp<T>(url: string): Observable<T> {
    return this.http.get<T>(url).pipe(
      catchError(error => {
        console.error(`[BridgeService] HTTP fallback failed for ${url}:`, error);
        return of({} as T); // Return an empty object on error to prevent crashes
      })
    );
  }
}
