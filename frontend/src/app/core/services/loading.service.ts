import { Injectable } from '@angular/core';
import { BehaviorSubject, combineLatest, map, distinctUntilChanged } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class LoadingService {
  private requestCount = 0;
  private routeCount = 0;

  private requestCount$ = new BehaviorSubject<number>(0);
  private routeCount$ = new BehaviorSubject<number>(0);

  readonly isLoading$ = combineLatest([this.requestCount$, this.routeCount$]).pipe(
    map(([requests, routes]) => requests > 0 || routes > 0),
    distinctUntilChanged()
  );

  startRequest(): void {
    this.requestCount += 1;
    this.requestCount$.next(this.requestCount);
  }

  endRequest(): void {
    this.requestCount = Math.max(0, this.requestCount - 1);
    this.requestCount$.next(this.requestCount);
  }

  startRouteTransition(): void {
    this.routeCount += 1;
    this.routeCount$.next(this.routeCount);
  }

  endRouteTransition(): void {
    this.routeCount = Math.max(0, this.routeCount - 1);
    this.routeCount$.next(this.routeCount);
  }
}
