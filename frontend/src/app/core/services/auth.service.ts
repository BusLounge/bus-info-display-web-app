import { Inject, Injectable, PLATFORM_ID } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';
import { Observable, tap } from 'rxjs';
import { ApiService } from './api.service';

export interface LoginResponse {
  token: string;
  token_type: string;
  expires_in: number;
  username: string;
  role: string;
  authenticatedAt: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly tokenKey = 'auth_token';
  private readonly userKey = 'auth_user';

  constructor(
    private api: ApiService,
    @Inject(PLATFORM_ID) private platformId: object
  ) {}

  login(username: string, password: string): Observable<LoginResponse> {
    return this.api.post<LoginResponse>('/auth/login', { username, password }).pipe(
      tap((response) => {
        if (!this.isBrowser()) {
          return;
        }

        localStorage.setItem(this.tokenKey, response.token);
        localStorage.setItem(this.userKey, JSON.stringify({
          username: response.username,
          role: response.role,
          authenticatedAt: response.authenticatedAt,
        }));
      })
    );
  }

  logout(): void {
    if (!this.isBrowser()) {
      return;
    }

    localStorage.removeItem(this.tokenKey);
    localStorage.removeItem(this.userKey);
  }

  isAuthenticated(): boolean {
    if (!this.isBrowser()) {
      return false;
    }

    return !!localStorage.getItem(this.tokenKey);
  }

  private isBrowser(): boolean {
    return isPlatformBrowser(this.platformId);
  }
}