import { Injectable } from '@angular/core';
import { HttpClient, HttpContext } from '@angular/common/http';
import { environment } from '../../../environments/environment';
import { Observable, of } from 'rxjs';
import { finalize, shareReplay, tap } from 'rxjs/operators';
import { SKIP_GLOBAL_LOADING } from '../interceptors/loading.interceptor';

export interface ApiRequestOptions {
  forceRefresh?: boolean;
  ttlMs?: number;
  suppressLoader?: boolean;
  responseType?: 'json' | 'text' | 'blob' | 'arraybuffer';
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private baseUrl = environment.apiUrl;
  private readonly defaultCacheTtlMs = 30000;
  private readonly storagePrefix = 'api-cache:';
  private responseCache = new Map<string, { expiresAt: number; value: unknown }>();
  private inflightRequests = new Map<string, Observable<unknown>>();
  
  constructor(private http: HttpClient) {}

  getApiUrl(): string {
    return this.baseUrl;
  }

  get<T>(path: string, options?: ApiRequestOptions): Observable<T> {
    const url = `${this.baseUrl}${path}`;
    const cacheKey = `GET:${url}`;
    const now = Date.now();
    const ttlMs = options?.ttlMs ?? this.defaultCacheTtlMs;

    if (!options?.forceRefresh) {
      const cached = this.responseCache.get(cacheKey);
      if (cached && cached.expiresAt > now) {
        return of(cached.value as T);
      }

      const stored = this.readStorageCache<T>(cacheKey, now);
      if (stored.found) {
        this.responseCache.set(cacheKey, {
          value: stored.value as T,
          expiresAt: stored.expiresAt
        });
        return of(stored.value as T);
      }

      const inflight = this.inflightRequests.get(cacheKey);
      if (inflight) {
        return inflight as Observable<T>;
      }
    }

    const request$ = this.http.get<T>(url, {
      context: this.buildContext(options)
    }).pipe(
      tap((value) => {
        const expiresAt = Date.now() + ttlMs;
        this.responseCache.set(cacheKey, {
          value,
          expiresAt
        });
        this.writeStorageCache(cacheKey, value, expiresAt);
      }),
      finalize(() => {
        this.inflightRequests.delete(cacheKey);
      }),
      shareReplay(1)
    );

    this.inflightRequests.set(cacheKey, request$ as Observable<unknown>);
    return request$;
  }

  post<T>(path: string, body: unknown, options?: ApiRequestOptions) {
    const url = `${this.baseUrl}${path}`;
    console.log('[ApiService] POST request to:', url);
    console.log('[ApiService] Request body:', body);
    this.clearGetCache();
    return this.http.post<T>(url, body, {
      context: this.buildContext(options)
    });
  }

  put<T>(path: string, body: unknown, options?: ApiRequestOptions) {
    this.clearGetCache();
    return this.http.put<T>(`${this.baseUrl}${path}`, body, {
      context: this.buildContext(options)
    });
  }

  delete<T>(path: string, options?: ApiRequestOptions) {
    this.clearGetCache();
    return this.http.delete<T>(`${this.baseUrl}${path}`, {
      context: this.buildContext(options)
    });
  }

  // Download blob (for files like PDFs, CSVs, Excel)
  getBlob(path: string, options?: ApiRequestOptions): Observable<Blob> {
    const url = `${this.baseUrl}${path}`;
    return this.http.get(url, {
      responseType: 'blob',
      context: this.buildContext(options)
    });
  }

  private buildContext(options?: ApiRequestOptions): HttpContext {
    if (!options?.suppressLoader) {
      return new HttpContext();
    }

    return new HttpContext().set(SKIP_GLOBAL_LOADING, true);
  }

  private clearGetCache() {
    this.responseCache.clear();
    this.inflightRequests.clear();
    this.clearStorageCache();
  }

  private readStorageCache<T>(cacheKey: string, now: number): { found: boolean; value?: T; expiresAt: number } {
    if (!this.isBrowserStorageAvailable()) {
      return { found: false, expiresAt: 0 };
    }

    const raw = window.sessionStorage.getItem(this.storagePrefix + cacheKey);
    if (!raw) {
      return { found: false, expiresAt: 0 };
    }

    try {
      const parsed = JSON.parse(raw) as { expiresAt: number; value: T };
      if (!parsed || typeof parsed.expiresAt !== 'number' || parsed.expiresAt <= now) {
        window.sessionStorage.removeItem(this.storagePrefix + cacheKey);
        return { found: false, expiresAt: 0 };
      }
      return { found: true, value: parsed.value, expiresAt: parsed.expiresAt };
    } catch {
      window.sessionStorage.removeItem(this.storagePrefix + cacheKey);
      return { found: false, expiresAt: 0 };
    }
  }

  private writeStorageCache(cacheKey: string, value: unknown, expiresAt: number) {
    if (!this.isBrowserStorageAvailable()) {
      return;
    }

    try {
      window.sessionStorage.setItem(
        this.storagePrefix + cacheKey,
        JSON.stringify({ value, expiresAt })
      );
    } catch {
      // Ignore storage quota/serialization errors and keep in-memory cache only.
    }
  }

  private clearStorageCache() {
    if (!this.isBrowserStorageAvailable()) {
      return;
    }

    const keysToDelete: string[] = [];
    for (let i = 0; i < window.sessionStorage.length; i++) {
      const key = window.sessionStorage.key(i);
      if (key && key.startsWith(this.storagePrefix)) {
        keysToDelete.push(key);
      }
    }

    keysToDelete.forEach((key) => window.sessionStorage.removeItem(key));
  }

  private isBrowserStorageAvailable(): boolean {
    return typeof window !== 'undefined' && typeof window.sessionStorage !== 'undefined';
  }
}
