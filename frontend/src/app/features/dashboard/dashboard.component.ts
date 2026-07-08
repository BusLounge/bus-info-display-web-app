import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HeaderComponent } from '../../shared/components/header/header.component';
import { DashboardService } from '../../core/services/dashboard.service';
import type {
  DashboardStats,
  RouteStats,
  AdvertisementCategoryStats,
  AdvertisementStatusStats,
  MediaCategoryStats,
  BroadcastMessage,
  BroadcastMessageRequest,
} from '../../core/services/dashboard.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, HeaderComponent, FormsModule],
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent implements OnInit, OnDestroy {
  loading: boolean = true;
  error: string = '';
  
  // Stats data
  stats: DashboardStats = {
    todayDepartures: 0,
    todayArrivals: 0,
    delayedArrivals: 0,
    delayedDepartures: 0
  };

  // Chart data
  routesPerLounges: RouteStats[] = [];
  advertisementCategories: AdvertisementCategoryStats[] = [];
  advertisementStatus: AdvertisementStatusStats[] = [];
  mediaCategories: MediaCategoryStats[] = [];
  broadcastMessages: BroadcastMessage[] = [];

  broadcastForm = {
    message: '',
    priority: 'normal',
    displayDurationSeconds: 10,
    frequencySeconds: 60,
    startAt: '',
    endAt: '',
    isActive: true,
    showOnLoungeTV: true,
  };

  broadcastsEnabled: boolean = true;
  broadcastError: string = '';
  broadcastSaving: boolean = false;
  private refreshInterval: any;
  private readonly refreshIntervalMs = 30000;

  constructor(private dashboardService: DashboardService) {}

  ngOnInit() {
    this.loadDashboardData();
    this.loadBroadcastMessages();
    this.loadBroadcastsEnabledState();
    this.refreshInterval = setInterval(() => this.loadDashboardData(true), this.refreshIntervalMs);
  }

  ngOnDestroy(): void {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }
  }

  loadBroadcastMessages() {
    this.broadcastError = '';
    this.dashboardService.getBroadcastMessages().subscribe({
      next: (items) => {
        this.broadcastMessages = items || [];
      },
      error: (err) => {
        console.error('Error loading broadcast messages:', err);
        this.broadcastError = 'Failed to load broadcast messages.';
      },
    });
  }

  createBroadcastMessage() {
    const payload: BroadcastMessageRequest = {
      message: this.broadcastForm.message.trim(),
      priority: this.broadcastForm.priority,
      displayDurationSeconds: Number(this.broadcastForm.displayDurationSeconds),
      frequencySeconds: Number(this.broadcastForm.frequencySeconds),
      isActive: this.broadcastForm.isActive,
      showOnLoungeTV: this.broadcastForm.showOnLoungeTV,
    };

    if (!payload.message) {
      this.broadcastError = 'Message is required.';
      return;
    }
    if (payload.displayDurationSeconds <= 0 || payload.frequencySeconds <= 0) {
      this.broadcastError = 'Duration and frequency must be greater than zero.';
      return;
    }

    if (this.broadcastForm.startAt) {
      payload.startAt = new Date(this.broadcastForm.startAt).toISOString();
    }
    if (this.broadcastForm.endAt) {
      payload.endAt = new Date(this.broadcastForm.endAt).toISOString();
    }

    this.broadcastSaving = true;
    this.broadcastError = '';
    this.dashboardService.createBroadcastMessage(payload).subscribe({
      next: () => {
        this.broadcastSaving = false;
        this.broadcastForm.message = '';
        this.broadcastForm.endAt = '';
        this.loadBroadcastMessages();
      },
      error: (err) => {
        console.error('Error creating broadcast message:', err);
        this.broadcastSaving = false;
        this.broadcastError = err?.error?.error || 'Failed to create broadcast message.';
      },
    });
  }

  toggleBroadcastMessage(item: BroadcastMessage) {
    const payload: BroadcastMessageRequest = {
      message: item.message,
      priority: item.priority,
      displayDurationSeconds: item.displayDurationSeconds,
      frequencySeconds: item.frequencySeconds,
      startAt: item.startAt,
      endAt: item.endAt,
      isActive: !item.isActive,
      showOnLoungeTV: item.showOnLoungeTV,
    };

    this.dashboardService.updateBroadcastMessage(item.id, payload).subscribe({
      next: () => this.loadBroadcastMessages(),
      error: (err) => {
        console.error('Error toggling broadcast message:', err);
        this.broadcastError = err?.error?.error || 'Failed to update broadcast message.';
      },
    });
  }

  deleteBroadcastMessage(item: BroadcastMessage) {
    if (!confirm('Delete this broadcast message?')) {
      return;
    }

    this.dashboardService.deleteBroadcastMessage(item.id).subscribe({
      next: () => this.loadBroadcastMessages(),
      error: (err) => {
        console.error('Error deleting broadcast message:', err);
        this.broadcastError = err?.error?.error || 'Failed to delete broadcast message.';
      },
    });
  }

  toggleBroadcasts() {
    console.log('[DEBUG] Toggling broadcasts to:', this.broadcastsEnabled);
    this.dashboardService.setBroadcastsEnabled(this.broadcastsEnabled).subscribe({
      next: () => {
        console.log('[DEBUG] Broadcasts toggled successfully');
      },
      error: (err) => {
        console.error('[ERROR] Failed to toggle broadcasts:', err);
        this.broadcastsEnabled = !this.broadcastsEnabled; // Revert on error
        this.broadcastError = 'Failed to update broadcast setting.';
      },
    });
  }

  private loadBroadcastsEnabledState() {
    this.dashboardService.getBroadcastsEnabledState().subscribe({
      next: (status) => {
        this.broadcastsEnabled = status.enabled;
        console.log('[DEBUG] Loaded broadcasts enabled state:', this.broadcastsEnabled);
      },
      error: (err) => {
        console.error('[WARNING] Could not load broadcasts enabled state:', err);
        // Keep default value (true)
      },
    });
  }

  loadDashboardData(silentRefresh: boolean = false) {
    if (!silentRefresh) {
      this.loading = true;
    }
    this.error = '';
    
    this.dashboardService.getDashboardData().subscribe({
      next: (data) => {
        this.stats = data.stats;
        this.routesPerLounges = data.routesPerLounges || [];
        this.advertisementCategories = data.advertisementCategories || [];
        this.advertisementStatus = data.advertisementStatus || [];
        this.mediaCategories = data.mediaCategories || [];
        this.loading = false;
      },
      error: (err) => {
        console.error('Error loading dashboard data:', err);
        this.error = 'Live dashboard refresh failed. Showing last available data.';
        this.loading = false;
      }
    });
  }

  // Helper methods for chart rendering
  getMaxValue(data: any[], key: string): number {
    if (!data || data.length === 0) return 1;
    return Math.max(...data.map(item => item[key]));
  }

  getBarHeight(value: number, maxValue: number): number {
    if (maxValue === 0) return 0;
    return (value / maxValue) * 100;
  }

  // Pie chart helper methods
  getPieSlicePath(index: number, data: any[]): string {
    if (!data || data.length === 0) return '';
    
    const centerX = 100;
    const centerY = 100;
    const radius = 80;
    
    let startAngle = -90; // Start from top
    for (let i = 0; i < index; i++) {
      startAngle += (data[i].percentage / 100) * 360;
    }
    
    const angle = (data[index].percentage / 100) * 360;
    
    // Handle full circle (100%)
    if (angle >= 359.99) {
      return `M ${centerX} ${centerY} m -${radius}, 0 a ${radius},${radius} 0 1,1 ${radius * 2},0 a ${radius},${radius} 0 1,1 -${radius * 2},0`;
    }
    
    const endAngle = startAngle + angle;
    
    const startRad = (startAngle * Math.PI) / 180;
    const endRad = (endAngle * Math.PI) / 180;
    
    const x1 = centerX + radius * Math.cos(startRad);
    const y1 = centerY + radius * Math.sin(startRad);
    const x2 = centerX + radius * Math.cos(endRad);
    const y2 = centerY + radius * Math.sin(endRad);
    
    const largeArc = angle > 180 ? 1 : 0;
    
    return `M ${centerX} ${centerY} L ${x1} ${y1} A ${radius} ${radius} 0 ${largeArc} 1 ${x2} ${y2} Z`;
  }

  getStatusPieSlicePath(index: number): string {
    return this.getPieSlicePath(index, this.advertisementStatus);
  }

  getMediaPieSlicePath(index: number): string {
    return this.getPieSlicePath(index, this.mediaCategories);
  }
}
