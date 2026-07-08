import { Injectable, signal, effect, PLATFORM_ID, inject } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';

export type Theme = 'light' | 'dark';

@Injectable({
  providedIn: 'root'
})
export class ThemeService {
  private readonly THEME_KEY = 'app-theme';
  private platformId = inject(PLATFORM_ID);
  
  // Signal for reactive theme state
  theme = signal<Theme>(this.getInitialTheme());

  constructor() {
    // Apply theme when it changes
    effect(() => {
      const currentTheme = this.theme();
      this.applyTheme(currentTheme);
      if (isPlatformBrowser(this.platformId)) {
        localStorage.setItem(this.THEME_KEY, currentTheme);
      }
    });
  }

  private getInitialTheme(): Theme {
    if (isPlatformBrowser(this.platformId)) {
      const stored = localStorage.getItem(this.THEME_KEY) as Theme;
      return stored || 'light';
    }
    return 'light';
  }

  private applyTheme(theme: Theme): void {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }
    
    const root = document.documentElement;
    root.setAttribute('data-theme', theme);
    
    if (theme === 'dark') {
      root.classList.add('dark-theme');
      root.classList.remove('light-theme');
    } else {
      root.classList.add('light-theme');
      root.classList.remove('dark-theme');
    }
  }

  toggleTheme(): void {
    this.theme.update(current => current === 'light' ? 'dark' : 'light');
  }

  setTheme(theme: Theme): void {
    this.theme.set(theme);
  }

  isDark(): boolean {
    return this.theme() === 'dark';
  }
}
