import { Component, CUSTOM_ELEMENTS_SCHEMA, OnDestroy, PLATFORM_ID, inject } from '@angular/core';
import { AsyncPipe, NgIf, isPlatformBrowser } from '@angular/common';
import {
  NavigationCancel,
  NavigationEnd,
  NavigationError,
  NavigationStart,
  Router,
  RouterOutlet
} from '@angular/router';
import { Subscription, filter } from 'rxjs';
import { LoadingService } from './core/services/loading.service';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, NgIf, AsyncPipe],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnDestroy {
  private readonly platformId = inject(PLATFORM_ID);
  readonly isLoading$;
  hideGlobalLoaderForCurrentRoute: boolean = false;
  private navigationSubscription: Subscription;
  private readonly onWindowResize = () => this.syncHeaderOffset();

  constructor(
    private router: Router,
    private loadingService: LoadingService
  ) {
    this.isLoading$ = this.loadingService.isLoading$;
    this.hideGlobalLoaderForCurrentRoute = this.isLoaderSuppressedForUrl(this.router.url);

    if (isPlatformBrowser(this.platformId)) {
      window.addEventListener('resize', this.onWindowResize);
      this.syncHeaderOffset();
    }

    this.navigationSubscription = this.router.events.pipe(
      filter((event) =>
        event instanceof NavigationStart ||
        event instanceof NavigationEnd ||
        event instanceof NavigationCancel ||
        event instanceof NavigationError
      )
    ).subscribe((event) => {
      if (event instanceof NavigationStart) {
        this.hideGlobalLoaderForCurrentRoute = this.isLoaderSuppressedForUrl(event.url);
        if (!this.hideGlobalLoaderForCurrentRoute) {
          this.loadingService.startRouteTransition();
        }
      } else {
        const resolvedUrl = event instanceof NavigationEnd
          ? event.urlAfterRedirects
          : ('url' in event ? event.url : this.router.url);
        this.hideGlobalLoaderForCurrentRoute = this.isLoaderSuppressedForUrl(resolvedUrl);
        this.loadingService.endRouteTransition();
        this.syncHeaderOffset();
      }
    });
  }

  ngOnDestroy(): void {
    if (isPlatformBrowser(this.platformId)) {
      window.removeEventListener('resize', this.onWindowResize);
    }

    this.navigationSubscription.unsubscribe();
  }

  private syncHeaderOffset(): void {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    // Delay one tick so routed header layout settles before reading height.
    setTimeout(() => {
      const header = document.querySelector('.app-header') as HTMLElement | null;
      const headerHeight = header ? header.offsetHeight : 0;
      document.documentElement.style.setProperty('--loader-nav-offset', `${headerHeight}px`);
    }, 0);
  }

  private isLoaderSuppressedForUrl(url: string | undefined): boolean {
    if (!url) {
      return false;
    }

    const normalized = url.toLowerCase();
    return normalized.startsWith('/bids-display');
  }
}
