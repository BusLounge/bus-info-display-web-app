import { Injectable } from '@angular/core';
import { from, Observable, of } from 'rxjs';
import { HttpClient } from '@angular/common/http';
import { catchError } from 'rxjs/operators';

// Define the shape of the Electron API that will be exposed on the window object
interface IElectronAPI {
  invoke: (channel: string, ...args: any[]) => Promise<any>;
  getAds: () => Promise<any>;
  getBroadcasts: () => Promise<any>;
  getLoungeAds: () => Promise<any>;
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
    if (this.ipc) {
      // Electron IPC path
      console.log(`[BridgeService] Using IPC for endpoint: ${endpoint}`);
      switch (endpoint) {
        case 'ads':
          return from(this.ipc.getAds() as Promise<T>);
        case 'broadcasts':
          return from(this.ipc.getBroadcasts() as Promise<T>);
        case 'lounge-ads':
          return from(this.ipc.getLoungeAds() as Promise<T>);
      }
      return from(this.ipc.invoke('bridge:get', endpoint) as Promise<T>);
    }
    
    // Fallback: regular HTTP (for browser / dev mode)
    console.log(`[BridgeService] Using HTTP for endpoint: ${endpoint}`);
    const url = `/local/${endpoint}`;
    return this.http.get<T>(url).pipe(
      catchError(error => {
        console.error(`[BridgeService] HTTP fallback failed for ${url}:`, error);
        return of({} as T); // Return an empty object on error to prevent crashes
      })
    );
  }
}
