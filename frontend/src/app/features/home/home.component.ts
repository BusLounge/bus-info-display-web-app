import { CommonModule, isPlatformBrowser } from '@angular/common';
import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { PLATFORM_ID } from '@angular/core';
import { Router } from '@angular/router';

@Component({
	selector: 'app-home',
	standalone: true,
	imports: [CommonModule],
	templateUrl: './home.component.html',
	styleUrl: './home.component.scss'
})
export class HomeComponent implements OnInit, OnDestroy {
	private readonly platformId = inject(PLATFORM_ID);
	private readonly backgroundUrls: string[] = [
		// Use multiple online images; these are stable Unsplash endpoints.
		'https://images.unsplash.com/photo-1544620347-c4fd4a3d5957?auto=format&fit=crop&w=1800&q=80',
		
	];

	showBgA = true;
	backgroundCssA = '';
	backgroundCssB = '';
	private backgroundIndex = 0;
	private timerId: ReturnType<typeof setInterval> | null = null;
	private readonly fadeMs = 900;

	constructor(private router: Router) {}

	ngOnInit(): void {
		// Pick an initial image based on the current time so it changes “by session”.
		const seed = Math.floor(Date.now() / 1000);
		this.backgroundIndex = seed % this.backgroundUrls.length;
		this.backgroundCssA = this.toCss(this.backgroundUrls[this.backgroundIndex]);
		this.backgroundCssB = this.backgroundCssA;
		this.showBgA = true;

		// Server-side rendering: don't start timers (no `window`).
		if (!isPlatformBrowser(this.platformId)) {
			return;
		}

		// Rotate every 12 seconds.
		this.timerId = globalThis.setInterval(() => {
			this.backgroundIndex = (this.backgroundIndex + 1) % this.backgroundUrls.length;
			this.crossfadeTo(this.backgroundUrls[this.backgroundIndex]);
		}, 12_000);
	}

	ngOnDestroy(): void {
		if (this.timerId !== null) {
			globalThis.clearInterval(this.timerId);
			this.timerId = null;
		}
	}

	navigateToLogin(): void {
		this.router.navigate(['/login']);
	}

	private crossfadeTo(url: string): void {
		if (!isPlatformBrowser(this.platformId)) {
			// Extra safety: SSR should never animate.
			this.backgroundCssA = this.toCss(url);
			this.backgroundCssB = this.backgroundCssA;
			this.showBgA = true;
			return;
		}

		const nextCss = this.toCss(url);
		if (this.showBgA) {
			this.backgroundCssB = nextCss;
		} else {
			this.backgroundCssA = nextCss;
		}

		// Next tick: flip the visible layer so CSS opacity transition runs.
		globalThis.setTimeout(() => {
			this.showBgA = !this.showBgA;
		}, 0);

		// After fade completes, keep both layers aligned to avoid flashing on resize.
		globalThis.setTimeout(() => {
			if (this.showBgA) {
				this.backgroundCssB = this.backgroundCssA;
			} else {
				this.backgroundCssA = this.backgroundCssB;
			}
		}, this.fadeMs + 50);
	}

	private toCss(url: string): string {
		return `url('${url}')`;
	}
}
