import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { forkJoin } from 'rxjs';
import {
  Lounge,
  LoungeService,
  LoungeRouteValidationResponse,
  LoungeRouteSegmentValidation
} from '../../core/services/lounge.service';
import { HeaderComponent } from '../../shared/components/header/header.component';

@Component({
  selector: 'app-lounge',
  standalone: true,
  imports: [CommonModule, HeaderComponent],
  templateUrl: './lounge.component.html',
  styleUrls: ['./lounge.component.scss']
})
export class LoungeComponent implements OnInit {
  lounges: Lounge[] = [];
  isLoadingLounges = false;
  isValidatingAll = false;
  error: string | null = null;

  private validationState: Record<string, LoungeRouteValidationResponse> = {};
  private validatingState: Record<string, boolean> = {};

  constructor(private loungeService: LoungeService) {}

  ngOnInit(): void {
    this.loadLounges();
  }

  loadLounges(): void {
    this.isLoadingLounges = true;
    this.error = null;

    this.loungeService.getAllLounges().subscribe({
      next: (lounges) => {
        this.lounges = lounges;
        this.isLoadingLounges = false;
      },
      error: (err) => {
        console.error('Failed to load lounges', err);
        this.error = 'Failed to load lounges.';
        this.isLoadingLounges = false;
      }
    });
  }

  validateLounge(loungeId: string): void {
    this.validatingState[loungeId] = true;

    this.loungeService.getLoungeRouteSegmentValidation(loungeId, true).subscribe({
      next: (result) => {
        this.validationState[loungeId] = result;
        this.validatingState[loungeId] = false;
      },
      error: (err) => {
        console.error(`Failed to validate lounge route segments for ${loungeId}`, err);
        this.validatingState[loungeId] = false;
      }
    });
  }

  validateAllLounges(): void {
    if (!this.lounges.length) {
      return;
    }

    this.isValidatingAll = true;
    const requests = this.lounges.map((lounge) => this.loungeService.getLoungeRouteSegmentValidation(lounge.id, true));

    forkJoin(requests).subscribe({
      next: (results) => {
        const map: Record<string, LoungeRouteValidationResponse> = {};
        results.forEach((result) => {
          map[result.loungeId] = result;
        });
        this.validationState = { ...this.validationState, ...map };
        this.isValidatingAll = false;
      },
      error: (err) => {
        console.error('Failed to validate all lounges', err);
        this.isValidatingAll = false;
      }
    });
  }

  getValidationResult(loungeId: string): LoungeRouteValidationResponse | null {
    return this.validationState[loungeId] ?? null;
  }

  isValidating(loungeId: string): boolean {
    return this.validatingState[loungeId] ?? false;
  }

  getInvalidSegments(loungeId: string): LoungeRouteSegmentValidation[] {
    const result = this.getValidationResult(loungeId);
    if (!result) {
      return [];
    }

    return result.segments.filter((segment) => !segment.isValid);
  }

  getInvalidLoungeCount(): number {
    return Object.values(this.validationState).filter((result) => !result.isValid).length;
  }

  getValidatedLoungeCount(): number {
    return Object.keys(this.validationState).length;
  }
}
