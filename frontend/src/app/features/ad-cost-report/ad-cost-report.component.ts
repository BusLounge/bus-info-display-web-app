import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HeaderComponent } from '../../shared/components/header/header.component';
import {
  AdvertisementService,
  AdvertisementCalculationRate,
  AdvertisementPlaybackLog,
  AdvertisementCostReportRow,
  AdvertisementPlaybackLogRequest,
} from '../../core/services';

@Component({
  selector: 'app-ad-cost-report',
  standalone: true,
  imports: [CommonModule, FormsModule, HeaderComponent],
  templateUrl: './ad-cost-report.component.html',
  styleUrl: './ad-cost-report.component.scss',
})
export class AdCostReportComponent implements OnInit {
  rates: AdvertisementCalculationRate[] = [];
  reportRows: AdvertisementCostReportRow[] = [];
  playbackLogs: AdvertisementPlaybackLog[] = [];
  reportSearchTerm = '';
  reportTrafficFilter: 'All' | 'Peak' | 'Moderate' | 'Off-Peak' = 'All';
  logsSearchTerm = '';
  logsTrafficFilter: 'All' | 'Peak' | 'Moderate' | 'Off-Peak' = 'All';
  logsAdvertisementId = '';
  logsLimit = 200;

  startDate = '';
  endDate = '';

  isLoadingRates = false;
  isLoadingReport = false;
  isLoadingLogs = false;
  isRecordingLog = false;

  successMessage = '';
  errorMessage = '';

  rateDrafts: Record<string, number> = {};

  logForm: AdvertisementPlaybackLogRequest = {
    advertisementId: '',
    advertisementName: '',
    durationSeconds: 10,
    playedAt: '',
  };

  constructor(private advertisementService: AdvertisementService) {}

  private isTrafficLevel(value: string): value is 'Peak' | 'Moderate' | 'Off-Peak' {
    return value === 'Peak' || value === 'Moderate' || value === 'Off-Peak';
  }

  ngOnInit(): void {
    const today = new Date();
    const weekAgo = new Date(today);
    weekAgo.setDate(today.getDate() - 7);

    this.startDate = weekAgo.toISOString().split('T')[0];
    this.endDate = today.toISOString().split('T')[0];

    this.loadRates();
    this.loadReport();
    this.loadPlaybackLogs();
  }

  refreshAll(): void {
    this.loadRates();
    this.loadReport();
    this.loadPlaybackLogs();
  }

  clearDateFilters(): void {
    this.startDate = '';
    this.endDate = '';
    this.loadReport();
    this.loadPlaybackLogs();
  }

  applyQuickRange(days: number): void {
    const today = new Date();
    const start = new Date(today);
    start.setDate(today.getDate() - days);

    this.startDate = start.toISOString().split('T')[0];
    this.endDate = today.toISOString().split('T')[0];
    this.loadReport();
    this.loadPlaybackLogs();
  }

  private extractErrorMessage(error: any, fallback: string): string {
    const apiMessage = error?.error?.error;
    if (apiMessage) {
      return apiMessage;
    }

    if (error?.status === 0) {
      return 'Cannot reach backend API. Start backend server on port 8083 and try again.';
    }

    if (error?.status === 404) {
      return 'Cost report API not found. Restart backend so latest routes are loaded.';
    }

    return fallback;
  }

  loadRates(): void {
    this.isLoadingRates = true;
    this.errorMessage = '';

    this.advertisementService.getCalculationRates().subscribe({
      next: (rates) => {
        this.rates = rates;
        this.rateDrafts = {};
        for (const rate of rates) {
          this.rateDrafts[rate.trafficLevel] = rate.costPerSecond;
        }
        this.isLoadingRates = false;
      },
      error: (error) => {
        this.isLoadingRates = false;
        this.errorMessage = this.extractErrorMessage(error, 'Failed to load cost rates.');
      },
    });
  }

  saveRate(trafficLevel: string): void {
    if (!this.isTrafficLevel(trafficLevel)) {
      this.errorMessage = 'Invalid traffic level selected.';
      return;
    }

    const value = Number(this.rateDrafts[trafficLevel]);
    if (Number.isNaN(value) || value < 0) {
      this.errorMessage = 'Cost per second must be a number greater than or equal to 0.';
      return;
    }

    this.errorMessage = '';
    this.successMessage = '';

    this.advertisementService.upsertCalculationRate(trafficLevel, value).subscribe({
      next: () => {
        this.successMessage = `${trafficLevel} rate saved.`;
        this.loadRates();
        this.loadReport();
      },
      error: (error) => {
        this.errorMessage = this.extractErrorMessage(error, 'Failed to save rate.');
      },
    });
  }

  loadReport(): void {
    this.isLoadingReport = true;
    this.errorMessage = '';

    this.advertisementService.getCostReport(this.startDate, this.endDate).subscribe({
      next: (response) => {
        this.reportRows = response?.rows || [];
        this.isLoadingReport = false;
      },
      error: (error) => {
        this.isLoadingReport = false;
        this.errorMessage = this.extractErrorMessage(error, 'Failed to load cost report.');
      },
    });
  }

  loadPlaybackLogs(): void {
    this.isLoadingLogs = true;
    this.errorMessage = '';

    const limitValue = Number(this.logsLimit);
    const safeLimit = Number.isFinite(limitValue) && limitValue > 0 ? Math.min(Math.floor(limitValue), 1000) : 200;
    this.logsLimit = safeLimit;

    this.advertisementService
      .getPlaybackLogs({
        startDate: this.startDate || undefined,
        endDate: this.endDate || undefined,
        advertisementId: this.logsAdvertisementId.trim() || undefined,
        trafficLevel: this.logsTrafficFilter,
        limit: safeLimit,
      })
      .subscribe({
        next: (response) => {
          this.playbackLogs = response?.rows || [];
          this.isLoadingLogs = false;
        },
        error: (error) => {
          this.isLoadingLogs = false;
          this.errorMessage = this.extractErrorMessage(error, 'Failed to load playback logs.');
        },
      });
  }

  recordPlaybackLog(): void {
    if (!this.logForm.advertisementId.trim() || !this.logForm.advertisementName.trim()) {
      this.errorMessage = 'Advertisement ID and Advertisement Name are required for log entry.';
      return;
    }

    if (!this.logForm.durationSeconds || this.logForm.durationSeconds <= 0) {
      this.errorMessage = 'Duration must be greater than 0.';
      return;
    }

    this.isRecordingLog = true;
    this.errorMessage = '';
    this.successMessage = '';

    const payload: AdvertisementPlaybackLogRequest = {
      ...this.logForm,
      advertisementId: this.logForm.advertisementId.trim(),
      advertisementName: this.logForm.advertisementName.trim(),
      durationSeconds: Number(this.logForm.durationSeconds),
      trafficLevel: this.logForm.trafficLevel,
      playedAt: this.logForm.playedAt?.trim() || undefined,
    };

    this.advertisementService.createPlaybackLog(payload).subscribe({
      next: () => {
        this.isRecordingLog = false;
        this.successMessage = 'Playback log added successfully.';
        this.loadReport();
        this.loadPlaybackLogs();
      },
      error: (error) => {
        this.isRecordingLog = false;
        this.errorMessage = this.extractErrorMessage(error, 'Failed to record playback log.');
      },
    });
  }

  exportCsv(): void {
    if (!this.reportRows.length) {
      this.errorMessage = 'No report rows to export.';
      return;
    }

    const headers = [
      'Advertisement ID',
      'Advertisement Name',
      'Traffic Level',
      'Play Count',
      'Total Seconds',
      'Cost Per Second',
      'Total Cost',
    ];

    const rows = this.reportRows.map((row) => [
      row.advertisementId,
      row.advertisementName,
      row.trafficLevel,
      row.playCount,
      row.totalSeconds,
      row.costPerSecond,
      row.totalCost,
    ]);

    const csv = [headers, ...rows]
      .map((line) =>
        line
          .map((field) => `"${String(field ?? '').replace(/"/g, '""')}"`)
          .join(',')
      )
      .join('\n');

    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `ad-cost-report-${this.startDate || 'all'}-to-${this.endDate || 'all'}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  exportLogsCsv(): void {
    if (!this.filteredPlaybackLogs.length) {
      this.errorMessage = 'No playback logs to export.';
      return;
    }

    const headers = [
      'Log ID',
      'Advertisement ID',
      'Advertisement Name',
      'Traffic Level',
      'Duration Seconds',
      'Played At',
      'Created At',
    ];

    const rows = this.filteredPlaybackLogs.map((row) => [
      row.id,
      row.advertisementId,
      row.advertisementName,
      row.trafficLevel,
      row.durationSeconds,
      row.playedAt,
      row.createdAt,
    ]);

    const csv = [headers, ...rows]
      .map((line) =>
        line
          .map((field) => `"${String(field ?? '').replace(/"/g, '""')}"`)
          .join(',')
      )
      .join('\n');

    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `ad-playback-logs-${this.startDate || 'all'}-to-${this.endDate || 'all'}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  get filteredReportRows(): AdvertisementCostReportRow[] {
    const term = this.reportSearchTerm.trim().toLowerCase();
    return this.reportRows.filter((row) => {
      const matchTraffic = this.reportTrafficFilter === 'All' || row.trafficLevel === this.reportTrafficFilter;
      if (!matchTraffic) {
        return false;
      }
      if (!term) {
        return true;
      }
      return (
        row.advertisementName.toLowerCase().includes(term) ||
        row.advertisementId.toLowerCase().includes(term)
      );
    });
  }

  get totalReportCost(): number {
    return this.filteredReportRows.reduce((sum, row) => sum + (Number(row.totalCost) || 0), 0);
  }

  get totalReportSeconds(): number {
    return this.filteredReportRows.reduce((sum, row) => sum + (Number(row.totalSeconds) || 0), 0);
  }

  get totalPlayCount(): number {
    return this.filteredReportRows.reduce((sum, row) => sum + (Number(row.playCount) || 0), 0);
  }

  get filteredPlaybackLogs(): AdvertisementPlaybackLog[] {
    const term = this.logsSearchTerm.trim().toLowerCase();
    return this.playbackLogs.filter((row) => {
      if (this.logsTrafficFilter !== 'All' && row.trafficLevel !== this.logsTrafficFilter) {
        return false;
      }

      if (term) {
        const haystack = `${row.advertisementName} ${row.advertisementId}`.toLowerCase();
        if (!haystack.includes(term)) {
          return false;
        }
      }

      return true;
    });
  }

  get totalPlaybackSeconds(): number {
    return this.filteredPlaybackLogs.reduce((sum, row) => sum + (Number(row.durationSeconds) || 0), 0);
  }

  get totalPlaybackRows(): number {
    return this.filteredPlaybackLogs.length;
  }

  formatCurrency(value: number): string {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(value || 0);
  }
}
