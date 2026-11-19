import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  Input,
  OnChanges,
  OnDestroy,
  ViewChild,
  inject
} from '@angular/core';

/**
 * Single-line text that slides sideways (and back) when it is too wide for
 * its box, pausing at each end. Text that fits stays still.
 *
 * The host element takes the cell's place in the layout, so the parent's
 * classes (font, colour, grid placement) go directly on <kiosk-marquee>.
 */
@Component({
  selector: 'kiosk-marquee',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<span class="marquee-track" #track>{{ text }}</span>`,
  styles: [`
    :host {
      display: block;
      min-width: 0;
      max-width: 100%;
      overflow: hidden;
      white-space: nowrap;
    }

    .marquee-track {
      display: inline-block;
      white-space: nowrap;
      will-change: transform;
    }

    :host(.is-scrolling) {
      text-align: left !important;
    }

    :host(.is-scrolling) .marquee-track {
      animation: kiosk-marquee var(--marquee-duration, 8s) ease-in-out infinite alternate;
    }

    /* Hold at the start, slide, hold at the end; 'alternate' slides back. */
    @keyframes kiosk-marquee {
      0%, 18% { transform: translateX(0); }
      82%, 100% { transform: translateX(var(--marquee-shift, 0)); }
    }

    @media (prefers-reduced-motion: reduce) {
      :host(.is-scrolling) .marquee-track {
        display: inline;
        animation: none;
      }
      :host(.is-scrolling) {
        text-overflow: ellipsis;
      }
    }
  `]
})
export class MarqueeComponent implements AfterViewInit, OnChanges, OnDestroy {
  @Input() text: string | number | null | undefined = '';

  @ViewChild('track', { static: true }) private track?: ElementRef<HTMLElement>;

  private readonly host: HTMLElement = inject(ElementRef<HTMLElement>).nativeElement;
  private resizeObserver?: ResizeObserver;
  private frame = 0;
  private appliedShiftPx = 0;

  /** Slide speed, in multiples of the font size per second. */
  private readonly speedEmPerSecond = 2.2;
  /** Share of each pass spent moving (the rest is the pause at the ends). */
  private readonly movingShare = 0.64;

  ngAfterViewInit(): void {
    if (typeof ResizeObserver !== 'undefined') {
      // Re-measure when the box resizes or the text width changes (new text,
      // web font finishing loading, language switch).
      this.resizeObserver = new ResizeObserver(() => this.scheduleMeasure());
      this.resizeObserver.observe(this.host);
      if (this.track) {
        this.resizeObserver.observe(this.track.nativeElement);
      }
    }
    this.scheduleMeasure();
  }

  ngOnChanges(): void {
    this.scheduleMeasure();
  }

  ngOnDestroy(): void {
    cancelAnimationFrame(this.frame);
    this.resizeObserver?.disconnect();
  }

  private scheduleMeasure(): void {
    if (typeof requestAnimationFrame === 'undefined') {
      return;
    }
    cancelAnimationFrame(this.frame);
    this.frame = requestAnimationFrame(() => this.measure());
  }

  private measure(): void {
    const track = this.track?.nativeElement;
    if (!track) {
      return;
    }

    const overflowPx = Math.ceil(track.offsetWidth - this.host.clientWidth);
    if (overflowPx <= 1) {
      if (this.appliedShiftPx !== 0) {
        this.appliedShiftPx = 0;
        this.host.classList.remove('is-scrolling');
        this.host.style.removeProperty('--marquee-shift');
        this.host.style.removeProperty('--marquee-duration');
      }
      return;
    }

    // Leave the animation alone unless the distance actually changed, so
    // routine re-measures don't restart it mid-slide.
    if (overflowPx === this.appliedShiftPx) {
      return;
    }
    this.appliedShiftPx = overflowPx;

    const fontSizePx = parseFloat(getComputedStyle(this.host).fontSize) || 16;
    const travelSeconds = overflowPx / (fontSizePx * this.speedEmPerSecond);
    const passSeconds = Math.max(3, travelSeconds / this.movingShare);

    this.host.style.setProperty('--marquee-shift', `-${overflowPx}px`);
    this.host.style.setProperty('--marquee-duration', `${passSeconds.toFixed(2)}s`);
    this.host.classList.add('is-scrolling');
  }
}
