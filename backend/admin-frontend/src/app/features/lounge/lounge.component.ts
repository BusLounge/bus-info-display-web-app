import { ChangeDetectorRef, Component, HostListener, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule, DOCUMENT } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription, from, of } from 'rxjs';
import { catchError, finalize, mergeMap } from 'rxjs/operators';
import {
  Lounge,
  LoungeService,
  LoungeRouteValidationResponse,
  LoungeRouteSegmentValidation
} from '../../core/services/lounge.service';
import { HeaderComponent } from '../../shared/components/header/header.component';

type StatusFilter = 'all' | 'approved' | 'rejected';
type OperationalFilter = 'all' | 'operational' | 'closed';
type ValidationFilter = 'all' | 'valid' | 'invalid' | 'unchecked';
type SortOption = 'name' | 'rating' | 'capacity' | 'updated';

/** Default coordinates many lounges were saved with instead of their real location. */
const DEFAULT_COLOMBO = { lat: 6.9271, lng: 79.8612 };
const PAGE_SIZE = 50;
/** How many validation requests run at the same time. */
const VALIDATION_CONCURRENCY = 6;

const AMENITY_LABELS: Record<string, string> = {
  ac: 'Air conditioning',
  wifi: 'Wi-Fi',
  restrooms: 'Restrooms',
  waiting_area: 'Waiting area',
  parking: 'Parking',
  cafeteria: 'Cafeteria',
  charging_ports: 'Charging ports',
  entertainment: 'Entertainment'
};

@Component({
  selector: 'app-lounge',
  standalone: true,
  imports: [CommonModule, FormsModule, HeaderComponent],
  templateUrl: './lounge.component.html',
  styleUrls: ['./lounge.component.scss']
})
export class LoungeComponent implements OnInit, OnDestroy {
  private readonly document = inject(DOCUMENT);
  // The app runs without zone.js, so async updates must ask for a re-render.
  private readonly changeDetector = inject(ChangeDetectorRef);

  lounges: Lounge[] = [];
  filteredLounges: Lounge[] = [];
  visibleCount = PAGE_SIZE;

  isLoadingLounges = false;
  error: string | null = null;

  // Search and filters
  searchTerm = '';
  statusFilter: StatusFilter = 'all';
  operationalFilter: OperationalFilter = 'all';
  validationFilter: ValidationFilter = 'all';
  sortBy: SortOption = 'name';

  // Bulk validation progress
  isValidatingBulk = false;
  bulkTotal = 0;
  bulkDone = 0;
  private bulkSubscription?: Subscription;

  // Expanded lounge (only one open at a time)
  expandedLoungeId: string | null = null;
  selectedImageIndex = 0;
  copiedLoungeId: string | null = null;
  private copiedTimer?: ReturnType<typeof setTimeout>;

  private validationState: Record<string, LoungeRouteValidationResponse> = {};
  private validatingState: Record<string, boolean> = {};
  private validationErrors: Record<string, boolean> = {};
  private brokenImages = new Set<string>();

  constructor(private loungeService: LoungeService) {}

  ngOnInit(): void {
    this.loadLounges();
  }

  ngOnDestroy(): void {
    this.bulkSubscription?.unsubscribe();
    clearTimeout(this.copiedTimer);
  }

  // ---------------------------------------------------------------- loading

  loadLounges(): void {
    this.isLoadingLounges = true;
    this.error = null;

    this.loungeService.getAllLounges().subscribe({
      next: (lounges) => {
        this.lounges = lounges;
        this.isLoadingLounges = false;
        this.applyFilters();
        this.refresh();
      },
      error: (err) => {
        console.error('Failed to load lounges', err);
        this.error = 'Failed to load lounges. Check that the API is running, then try again.';
        this.isLoadingLounges = false;
        this.refresh();
      }
    });
  }

  // ---------------------------------------------------------------- search, filters, sorting

  applyFilters(): void {
    const terms = this.searchTerm.trim().toLowerCase().split(/\s+/).filter(Boolean);

    const filtered = this.lounges.filter((lounge) => {
      if (terms.length) {
        const haystack = this.searchText(lounge);
        if (!terms.every((term) => haystack.includes(term))) {
          return false;
        }
      }
      if (this.statusFilter !== 'all' && (lounge.status || '').toLowerCase() !== this.statusFilter) {
        return false;
      }
      if (this.operationalFilter === 'operational' && !lounge.isOperational) {
        return false;
      }
      if (this.operationalFilter === 'closed' && lounge.isOperational) {
        return false;
      }
      if (this.validationFilter !== 'all') {
        const result = this.validationState[lounge.id];
        if (this.validationFilter === 'unchecked' && result) return false;
        if (this.validationFilter === 'valid' && !(result && result.isValid)) return false;
        if (this.validationFilter === 'invalid' && !(result && !result.isValid)) return false;
      }
      return true;
    });

    const phrase = terms.join(' ');
    const relevance = (lounge: Lounge): number => {
      if (!terms.length) return 0;
      const name = (lounge.loungeName || '').toLowerCase();
      if (name.includes(phrase)) return 2;
      return terms.every((term) => name.includes(term)) ? 1 : 0;
    };

    this.filteredLounges = filtered.sort(
      (a, b) => relevance(b) - relevance(a) || this.compareLounges(a, b)
    );
    this.visibleCount = PAGE_SIZE;
  }

  clearFilters(): void {
    this.searchTerm = '';
    this.statusFilter = 'all';
    this.operationalFilter = 'all';
    this.validationFilter = 'all';
    this.sortBy = 'name';
    this.applyFilters();
  }

  get hasActiveFilters(): boolean {
    return !!this.searchTerm.trim() ||
      this.statusFilter !== 'all' ||
      this.operationalFilter !== 'all' ||
      this.validationFilter !== 'all';
  }

  get visibleLounges(): Lounge[] {
    return this.filteredLounges.slice(0, this.visibleCount);
  }

  showMore(): void {
    this.visibleCount += PAGE_SIZE;
  }

  trackByLoungeId(_: number, lounge: Lounge): string {
    return lounge.id;
  }

  private searchText(lounge: Lounge): string {
    return [
      lounge.loungeName,
      lounge.address,
      lounge.district,
      lounge.city,
      lounge.state,
      lounge.country,
      lounge.postalCode,
      lounge.contactPhone,
      lounge.id,
      lounge.description,
      ...this.getAmenities(lounge)
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase();
  }

  private compareLounges(a: Lounge, b: Lounge): number {
    switch (this.sortBy) {
      case 'rating':
        return (b.averageRating ?? 0) - (a.averageRating ?? 0) || this.byName(a, b);
      case 'capacity':
        return (b.capacity ?? 0) - (a.capacity ?? 0) || this.byName(a, b);
      case 'updated':
        return (Date.parse(b.updatedAt || '') || 0) - (Date.parse(a.updatedAt || '') || 0);
      default:
        return this.byName(a, b);
    }
  }

  private byName(a: Lounge, b: Lounge): number {
    return (a.loungeName || '').trim().localeCompare((b.loungeName || '').trim(), undefined, { sensitivity: 'base' });
  }

  // ---------------------------------------------------------------- stats

  get operationalCount(): number {
    return this.lounges.filter((lounge) => lounge.isOperational).length;
  }

  get rejectedCount(): number {
    return this.lounges.filter((lounge) => (lounge.status || '').toLowerCase() === 'rejected').length;
  }

  getValidatedLoungeCount(): number {
    return Object.keys(this.validationState).length;
  }

  getInvalidLoungeCount(): number {
    return Object.values(this.validationState).filter((result) => !result.isValid).length;
  }

  // ---------------------------------------------------------------- validation

  validateLounge(loungeId: string, event?: Event): void {
    event?.stopPropagation();
    this.validatingState[loungeId] = true;
    delete this.validationErrors[loungeId];

    this.loungeService.getLoungeRouteSegmentValidation(loungeId, true).subscribe({
      next: (result) => {
        this.validationState[loungeId] = result;
        this.validatingState[loungeId] = false;
        this.refresh();
      },
      error: (err) => {
        console.error(`Failed to validate lounge route segments for ${loungeId}`, err);
        this.validationErrors[loungeId] = true;
        this.validatingState[loungeId] = false;
        this.refresh();
      }
    });
  }

  /** Validates the lounges currently shown by the search and filters. */
  validateFiltered(): void {
    this.runBulkValidation(this.filteredLounges);
  }

  validateAllLounges(): void {
    this.runBulkValidation(this.lounges);
  }

  cancelBulkValidation(): void {
    this.bulkSubscription?.unsubscribe();
    this.isValidatingBulk = false;
    this.applyFilters();
  }

  private runBulkValidation(lounges: Lounge[]): void {
    if (!lounges.length || this.isValidatingBulk) {
      return;
    }
    this.isValidatingBulk = true;
    this.bulkTotal = lounges.length;
    this.bulkDone = 0;

    this.bulkSubscription = from(lounges)
      .pipe(
        mergeMap(
          (lounge) =>
            this.loungeService.getLoungeRouteSegmentValidation(lounge.id, true, true).pipe(
              catchError((err) => {
                console.error(`Failed to validate lounge ${lounge.id}`, err);
                this.validationErrors[lounge.id] = true;
                return of(null);
              }),
              finalize(() => {
                this.bulkDone++;
                this.refresh();
              })
            ),
          VALIDATION_CONCURRENCY
        ),
        finalize(() => {
          this.isValidatingBulk = false;
          if (this.validationFilter !== 'all') {
            this.applyFilters();
          }
          this.refresh();
        })
      )
      .subscribe((result) => {
        if (result) {
          this.validationState[result.loungeId] = result;
          delete this.validationErrors[result.loungeId];
          this.refresh();
        }
      });
  }

  getValidationResult(loungeId: string): LoungeRouteValidationResponse | null {
    return this.validationState[loungeId] ?? null;
  }

  isValidating(loungeId: string): boolean {
    return this.validatingState[loungeId] ?? false;
  }

  hasValidationError(loungeId: string): boolean {
    return this.validationErrors[loungeId] ?? false;
  }

  getInvalidSegments(loungeId: string): LoungeRouteSegmentValidation[] {
    const result = this.getValidationResult(loungeId);
    return result ? result.segments.filter((segment) => !segment.isValid) : [];
  }

  get bulkProgressPercent(): number {
    return this.bulkTotal ? Math.round((this.bulkDone / this.bulkTotal) * 100) : 0;
  }

  // ---------------------------------------------------------------- expandable rows

  isExpanded(lounge: Lounge): boolean {
    return this.expandedLoungeId === lounge.id;
  }

  toggleLounge(lounge: Lounge, row?: HTMLElement): void {
    if (this.isExpanded(lounge)) {
      this.expandedLoungeId = null;
      return;
    }
    this.expandedLoungeId = lounge.id;
    this.selectedImageIndex = 0;
    // Keep the opened row's header in view.
    if (row) {
      setTimeout(() => row.scrollIntoView({ block: 'nearest', behavior: 'smooth' }));
    }
  }

  @HostListener('document:keydown.escape')
  collapseExpanded(): void {
    this.expandedLoungeId = null;
  }

  stepImage(lounge: Lounge, direction: number): void {
    const images = this.getImages(lounge);
    if (images.length < 2) return;
    this.selectedImageIndex = (this.selectedImageIndex + direction + images.length) % images.length;
  }

  copyLoungeId(id: string, event: Event): void {
    event.stopPropagation();
    const done = () => {
      this.copiedLoungeId = id;
      this.refresh();
      clearTimeout(this.copiedTimer);
      this.copiedTimer = setTimeout(() => {
        this.copiedLoungeId = null;
        this.refresh();
      }, 1500);
    };
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(id).then(done).catch(() => this.copyWithFallback(id) && done());
    } else if (this.copyWithFallback(id)) {
      done();
    }
  }

  /** Copies text where the Clipboard API is unavailable (e.g. plain http on another host). */
  private copyWithFallback(text: string): boolean {
    const textarea = this.document.createElement('textarea');
    textarea.value = text;
    textarea.setAttribute('readonly', '');
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    this.document.body.appendChild(textarea);
    textarea.select();
    let ok = false;
    try {
      ok = this.document.execCommand('copy');
    } catch {
      ok = false;
    }
    this.document.body.removeChild(textarea);
    return ok;
  }

  private refresh(): void {
    this.changeDetector.markForCheck();
  }

  // ---------------------------------------------------------------- display helpers

  getImages(lounge: Lounge): string[] {
    return this.parseList(lounge.images).filter((url) => /^https?:\/\//i.test(url) && !this.brokenImages.has(url));
  }

  onImageError(url: string, lounge?: Lounge): void {
    this.brokenImages.add(url);
    if (lounge && this.selectedImageIndex >= this.getImages(lounge).length) {
      this.selectedImageIndex = 0;
    }
  }

  getAmenities(lounge: Lounge): string[] {
    return this.parseList(lounge.amenities);
  }

  amenityLabel(value: string): string {
    const key = value.trim().toLowerCase();
    if (AMENITY_LABELS[key]) {
      return AMENITY_LABELS[key];
    }
    const words = key.replace(/[_-]+/g, ' ');
    return words.charAt(0).toUpperCase() + words.slice(1);
  }

  initials(name: string): string {
    return (name || '?')
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((word) => word.charAt(0).toUpperCase())
      .join('');
  }

  locationLine(lounge: Lounge): string {
    return [lounge.district || lounge.city, lounge.state].filter(Boolean).join(', ');
  }

  hasCoordinates(lounge: Lounge): boolean {
    return typeof lounge.latitude === 'number' && typeof lounge.longitude === 'number';
  }

  usesDefaultCoordinates(lounge: Lounge): boolean {
    return lounge.latitude === DEFAULT_COLOMBO.lat && lounge.longitude === DEFAULT_COLOMBO.lng;
  }

  mapsUrl(lounge: Lounge): string {
    return `https://www.google.com/maps/search/?api=1&query=${lounge.latitude},${lounge.longitude}`;
  }

  hasPricing(lounge: Lounge): boolean {
    return [lounge.price1Hour, lounge.price2Hours, lounge.price3Hours, lounge.priceUntilBus]
      .some((price) => typeof price === 'number' && price > 0);
  }

  formatPrice(price?: number): string {
    if (typeof price !== 'number' || price <= 0) {
      return '—';
    }
    return `LKR ${price.toLocaleString('en-LK', { maximumFractionDigits: 2 })}`;
  }

  /** Ratings are 0 until a lounge gets its first review. */
  hasRating(lounge: Lounge): boolean {
    return typeof lounge.averageRating === 'number' && lounge.averageRating > 0;
  }

  private parseList(value: string | string[] | null | undefined): string[] {
    if (!value) {
      return [];
    }
    if (Array.isArray(value)) {
      return value.map(String).map((item) => item.trim()).filter(Boolean);
    }
    const text = value.trim();
    if (!text) {
      return [];
    }
    if (text.startsWith('[')) {
      try {
        const parsed = JSON.parse(text);
        return Array.isArray(parsed) ? parsed.map(String).map((item) => item.trim()).filter(Boolean) : [];
      } catch {
        // fall through to comma-separated text
      }
    }
    return text.replace(/^[{[]|[}\]]$/g, '').split(',').map((item) => item.replace(/^["']|["']$/g, '').trim()).filter(Boolean);
  }
}
