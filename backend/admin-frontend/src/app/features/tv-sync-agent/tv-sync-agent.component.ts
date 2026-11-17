import { Component, OnInit, OnDestroy, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subject, interval } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { TVSyncAgentService } from '../../core/services/tv-sync-agent.service';
import { LoungeService } from '../../core/services/lounge.service';
import { AdvertisementService } from '../../core/services/advertisement.service';
import {
  TVSyncAgentConfig,
  DisplayMode,
  LayoutOption,
  DeviceStatus,
  EmergencyMessage,
  SUPPORTED_LANGUAGES,
  AdPlaybackTracking,
  BillingReport,
  LoungeSpecificAd,
  LoungeSpecificAdRequest,
  LoungeAdSlotSummary,
  LoungeAdTimeSlot,
} from '../../core/models/tv-sync-agent.model';
import { Lounge } from '../../core/models';

@Component({
  selector: 'app-tv-sync-agent',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './tv-sync-agent.component.html',
  styleUrls: ['./tv-sync-agent.component.scss'],
})
export class TVSyncAgentComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();

  // Signals for reactive state management
  lounges = signal<Lounge[]>([]);
  devices = signal<DeviceStatus[]>([]);
  selectedLounge = signal<Lounge | null>(null);
  config = signal<TVSyncAgentConfig | null>(null);
  emergencyMessage = signal<EmergencyMessage | null>(null);
  adPlaybackHistory = signal<AdPlaybackTracking[]>([]);
  billingReport = signal<BillingReport | null>(null);
  loungeAds = signal<LoungeSpecificAd[]>([]);
  loungeAdSlots = signal<LoungeAdSlotSummary | null>(null);
  loungeAdSlotsLoadError = signal<string | null>(null);

  // UI State
  activeTab = signal<string>('overview');
  isLoading = signal<boolean>(false);
  isSaving = signal<boolean>(false);
  showEmergencyDialog = signal<boolean>(false);
  showBillingDialog = signal<boolean>(false);
  lastSyncTime = signal<Date | null>(null);
  syncProgress = signal<number>(0);

  // Configuration Options
  displayModes: { value: DisplayMode; label: string; icon: string }[] = [
    { value: 'schedules-only', label: 'Schedules Only', icon: '📅' },
    { value: 'ads-only', label: 'Ads Only', icon: '📺' },
    { value: 'both', label: 'Schedules & Ads', icon: '🔀' },
  ];

  layoutOptions: { value: LayoutOption; label: string; icon: string; description: string }[] = [
    {
      value: 'split-screen',
      label: 'Split Screen',
      icon: '⬌',
      description: 'Show schedules and ads side by side',
    },
    {
      value: 'full-screen-alternate',
      label: 'Full Screen Alternate',
      icon: '⇄',
      description: 'Alternate between full screen schedules and ads',
    },
  ];

  languages = SUPPORTED_LANGUAGES;

  // Emergency Message Form
  emergencyMessageForm = {
    enabled: false,
    message: '',
    priority: 'medium' as 'low' | 'medium' | 'high' | 'critical',
    backgroundColor: '#ff0000',
    textColor: '#ffffff',
    displayDuration: 10,
  };

  // Billing Report Form
  billingReportForm = {
    startDate: '',
    endDate: '',
  };

  loungeAdForm = {
    advertisementName: '',
    mediaUrl: '',
    mediaType: 'image' as 'image' | 'video',
    durationSeconds: 4,
    priority: 'normal',
    isActive: true,
    isDefaultForAll: false,
  };

  selectedLoungeAdFile: File | null = null;
  loungeAdUploadError = signal<string | null>(null);
  isUploadingLoungeAdMedia = signal<boolean>(false);

  // Computed values
  onlineDevicesCount = computed(() => {
    return this.devices().filter((d) => d.status === 'online').length;
  });

  offlineDevicesCount = computed(() => {
    return this.devices().filter((d) => d.status === 'offline').length;
  });

  totalAdsPlayed = computed(() => {
    return this.adPlaybackHistory().length;
  });

  loungeAdRemainingSeconds = computed(() => {
    const slots = this.loungeAdSlots();
    if (!slots) {
      return 0;
    }
    const remainingFromCompany = Math.max(0, slots.remainingSeconds);
    return Math.max(0, Math.min(remainingFromCompany, 24 - this.loungeAdForm.durationSeconds));
  });

  loungeAdOverflowSeconds = computed(() => {
    const slots = this.loungeAdSlots();
    if (!slots) {
      return 0;
    }
    return Math.max(0, this.loungeAdForm.durationSeconds - slots.remainingSeconds);
  });

  interactiveAvailableSlots = computed((): LoungeAdTimeSlot[] => {
    const slots = this.loungeAdSlots();
    if (!slots) {
      return [];
    }

    const available = (slots.availableSlots || []).filter(
      (slot) => slot.interactive && slot.durationSeconds > 0
    );

    if (available.length > 0) {
      return available;
    }

    if (slots.remainingSeconds > 0) {
      return [
        {
          type: 'available',
          label: 'Available for lounge/default ads',
          startSecond: 6 + slots.bookedSeconds,
          endSecond: 30,
          durationSeconds: slots.remainingSeconds,
          interactive: true,
        },
      ];
    }

    return [];
  });

  interactiveDurationSlots = computed((): LoungeAdTimeSlot[] => {
    const baseAvailable = this.interactiveAvailableSlots();
    if (!baseAvailable.length) {
      return [];
    }

    const primaryWindow = baseAvailable[0];
    const maxSeconds = Math.max(0, Math.floor(primaryWindow.durationSeconds || 0));
    if (maxSeconds <= 0) {
      return [];
    }

    const options: LoungeAdTimeSlot[] = [];
    for (let seconds = 1; seconds <= maxSeconds; seconds++) {
      options.push({
        type: 'available',
        label: `${seconds}s`,
        startSecond: primaryWindow.startSecond,
        endSecond: primaryWindow.startSecond + seconds,
        durationSeconds: seconds,
        interactive: true,
      });
    }

    return options;
  });

  selectedDurationEndSecond = computed((): number => {
    const slots = this.loungeAdSlots();
    if (!slots) {
      return 6;
    }

    const adStart = 6 + Math.max(0, Math.min(24, slots.bookedSeconds || 0));
    const requested = Math.max(0, Math.floor(Number(this.loungeAdForm.durationSeconds) || 0));
    const allowed = Math.max(0, Math.min(requested, slots.remainingSeconds || 0));
    return adStart + allowed;
  });

  loungeSlotStatusText = computed((): string => {
    if (this.loungeAdSlotsLoadError()) {
      return 'Unable to load slot summary. Check backend API and lounge mapping, then refresh.';
    }

    const slots = this.loungeAdSlots();
    if (!slots) {
      return 'Loading slot summary...';
    }

    if (slots.remainingSeconds <= 0) {
      return 'No available ad seconds right now. Company ads are using the full 24-second ad window.';
    }

    if ((slots.bookedSeconds || 0) === 0) {
      return 'No company ads are scheduled. Full 24-second ad window (6s-30s) is available for lounge/default ads.';
    }

    return `Available now: ${slots.remainingSeconds}s for lounge/default ads.`;
  });

  cyclePreviewSegments = computed(() => {
    const slots = this.loungeAdSlots();
    const scheduleSeconds = 6;
    const adWindow = 24;
    const companyBooked = Math.min(adWindow, Math.max(0, slots?.bookedSeconds || 0));
    const requestedLoungeSeconds = Math.min(adWindow, Math.max(0, Number(this.loungeAdForm.durationSeconds) || 0));
    const availableForLounge = Math.max(0, adWindow - companyBooked);
    const appliedLoungeSeconds = Math.min(requestedLoungeSeconds, availableForLounge);
    const fallbackSeconds = Math.max(0, adWindow - companyBooked - appliedLoungeSeconds);

    return [
      {
        key: 'schedule',
        label: `Schedules ${scheduleSeconds}s`,
        seconds: scheduleSeconds,
      },
      {
        key: 'company',
        label: `Company ads ${companyBooked}s`,
        seconds: companyBooked,
      },
      {
        key: 'lounge',
        label: `This lounge ad ${appliedLoungeSeconds}s`,
        seconds: appliedLoungeSeconds,
      },
      {
        key: 'fallback',
        label: `Fallback ads ${fallbackSeconds}s`,
        seconds: fallbackSeconds,
      },
    ];
  });

  constructor(
    private tvSyncService: TVSyncAgentService,
    private loungeService: LoungeService,
    private adService: AdvertisementService
  ) {}

  ngOnInit(): void {
    this.loadLounges();
    this.setupAutoRefresh();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  // Data Loading
  loadLounges(): void {
    this.isLoading.set(true);
    this.loungeService
      .getAllLounges()
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (lounges) => {
          this.lounges.set(lounges);
          if (lounges.length > 0 && !this.selectedLounge()) {
            this.selectLounge(lounges[0]);
          }
          this.isLoading.set(false);
        },
        error: (error) => {
          console.error('Error loading lounges:', error);
          this.isLoading.set(false);
        },
      });
  }

  selectLounge(lounge: Lounge): void {
    this.selectedLounge.set(lounge);
    this.loadLoungeData(lounge.id);
  }

  loadLoungeData(loungeId: string): void {
    this.isLoading.set(true);

    // Load TV Sync Config
    this.tvSyncService
      .getAgentConfig(loungeId)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (config) => {
          this.config.set(config);
        },
        error: (error) => {
          console.error('Error loading config:', error);
          // Initialize default config if not found
          this.initializeDefaultConfig(loungeId);
        },
      });

    // Load Devices
    this.tvSyncService
      .getAllDevices(loungeId)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (devices) => {
          this.devices.set(devices);
        },
        error: (error) => {
          console.error('Error loading devices:', error);
        },
      });

    // Load Emergency Message
    this.tvSyncService
      .getEmergencyMessage(loungeId)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (message) => {
          this.emergencyMessage.set(message);
        },
        error: (error) => {
          console.error('Error loading emergency message:', error);
        },
      });

    // Load Ad Playback History (last 7 days)
    const endDate = new Date().toISOString().split('T')[0];
    const startDate = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0];

    this.tvSyncService
      .getAdPlaybackHistory(loungeId, startDate, endDate)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (history) => {
          this.adPlaybackHistory.set(history);
          this.isLoading.set(false);
        },
        error: (error) => {
          console.error('Error loading ad playback history:', error);
          this.isLoading.set(false);
        },
      });

    this.tvSyncService
      .getLoungeAds(loungeId)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (ads) => {
          this.loungeAds.set(ads || []);
        },
        error: (error) => {
          console.error('Error loading lounge ads:', error);
          this.loungeAds.set([]);
        },
      });

    this.tvSyncService
      .getLoungeAdSlots(loungeId)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (slots) => {
          this.loungeAdSlots.set(slots);
          this.loungeAdSlotsLoadError.set(null);
        },
        error: (error) => {
          console.error('Error loading lounge ad slot summary:', error);
          this.loungeAdSlots.set(null);
          this.loungeAdSlotsLoadError.set(error?.error?.error || 'Slot summary API request failed');
        },
      });
  }

  initializeDefaultConfig(loungeId: string): void {
    const lounge = this.selectedLounge();
    if (!lounge) return;

    const defaultConfig: TVSyncAgentConfig = {
      loungeId: loungeId,
      loungeName: lounge.loungeName,
      displayMode: 'both',
      language: 'en',
      layoutOption: 'split-screen',
      adPlaybackTracking: [],
      deviceStatus: {
        deviceId: 'default',
        status: 'offline',
      },
    };

    this.config.set(defaultConfig);
  }

  // Configuration Updates
  updateDisplayMode(mode: DisplayMode): void {
    const currentConfig = this.config();
    if (!currentConfig) return;

    const updatedConfig = { ...currentConfig, displayMode: mode };
    this.config.set(updatedConfig);
    this.saveConfiguration();
  }

  updateLayoutOption(layout: LayoutOption): void {
    const currentConfig = this.config();
    if (!currentConfig) return;

    const updatedConfig = { ...currentConfig, layoutOption: layout };
    this.config.set(updatedConfig);
    this.saveConfiguration();
  }

  updateLanguage(languageCode: string): void {
    const currentConfig = this.config();
    if (!currentConfig) return;

    const updatedConfig = { ...currentConfig, language: languageCode };
    this.config.set(updatedConfig);
    this.saveConfiguration();
  }

  saveConfiguration(): void {
    const currentConfig = this.config();
    if (!currentConfig) return;

    this.isSaving.set(true);
    this.tvSyncService
      .saveAgentConfig(currentConfig)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (savedConfig) => {
          this.config.set(savedConfig);
          this.isSaving.set(false);
          this.showSuccessNotification('Configuration saved successfully');
        },
        error: (error) => {
          console.error('Error saving configuration:', error);
          this.isSaving.set(false);
          this.showErrorNotification('Failed to save configuration');
        },
      });
  }

  // Emergency Message
  openEmergencyDialog(): void {
    const currentMessage = this.emergencyMessage();
    if (currentMessage) {
      this.emergencyMessageForm = {
        enabled: currentMessage.enabled,
        message: currentMessage.message,
        priority: currentMessage.priority,
        backgroundColor: currentMessage.backgroundColor ?? '#ff0000',
        textColor: currentMessage.textColor ?? '#ffffff',
        displayDuration: currentMessage.displayDuration ?? 10,
      };
    }
    this.showEmergencyDialog.set(true);
  }

  closeEmergencyDialog(): void {
    this.showEmergencyDialog.set(false);
  }

  saveEmergencyMessage(): void {
    const lounge = this.selectedLounge();
    if (!lounge) return;

    const message: EmergencyMessage = {
      ...this.emergencyMessageForm,
      createdAt: new Date().toISOString(),
    };

    this.tvSyncService
      .setEmergencyMessage(lounge.id, message)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.emergencyMessage.set(message);
          this.closeEmergencyDialog();
          this.showSuccessNotification('Emergency message activated');
        },
        error: (error) => {
          console.error('Error setting emergency message:', error);
          this.showErrorNotification('Failed to activate emergency message');
        },
      });
  }

  clearEmergencyMessage(): void {
    const lounge = this.selectedLounge();
    if (!lounge) return;

    this.tvSyncService
      .clearEmergencyMessage(lounge.id)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.emergencyMessage.set(null);
          this.showSuccessNotification('Emergency message cleared');
        },
        error: (error) => {
          console.error('Error clearing emergency message:', error);
          this.showErrorNotification('Failed to clear emergency message');
        },
      });
  }

  // Billing Reports
  openBillingDialog(): void {
    const endDate = new Date();
    const startDate = new Date();
    startDate.setMonth(startDate.getMonth() - 1);

    this.billingReportForm = {
      startDate: startDate.toISOString().split('T')[0],
      endDate: endDate.toISOString().split('T')[0],
    };

    this.showBillingDialog.set(true);
  }

  closeBillingDialog(): void {
    this.showBillingDialog.set(false);
  }

  generateBillingReport(): void {
    const lounge = this.selectedLounge();
    if (!lounge) return;

    this.isLoading.set(true);
    this.tvSyncService
      .generateBillingReport(
        lounge.id,
        this.billingReportForm.startDate,
        this.billingReportForm.endDate
      )
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (report) => {
          this.billingReport.set(report);
          this.isLoading.set(false);
          this.showSuccessNotification('Billing report generated');
        },
        error: (error) => {
          console.error('Error generating billing report:', error);
          this.isLoading.set(false);
          this.showErrorNotification('Failed to generate billing report');
        },
      });
  }

  downloadBillingReport(format: 'pdf' | 'csv' | 'excel'): void {
    const lounge = this.selectedLounge();
    if (!lounge) return;

    this.tvSyncService
      .downloadBillingReport(
        lounge.id,
        this.billingReportForm.startDate,
        this.billingReportForm.endDate,
        format
      )
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (blob) => {
          const url = window.URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `billing-report-${lounge.loungeName}-${new Date().toISOString()}.${format}`;
          a.click();
          window.URL.revokeObjectURL(url);
          this.showSuccessNotification(`Report downloaded as ${format.toUpperCase()}`);
        },
        error: (error) => {
          console.error('Error downloading billing report:', error);
          this.showErrorNotification('Failed to download report');
        },
      });
  }

  // Device Management
  syncNow(): void {
    const lounge = this.selectedLounge();
    if (!lounge) return;

    this.syncProgress.set(0);
    const progressInterval = setInterval(() => {
      this.syncProgress.update((p) => Math.min(p + 10, 90));
    }, 100);

    this.tvSyncService
      .syncNow(lounge.id)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          clearInterval(progressInterval);
          this.syncProgress.set(100);
          this.lastSyncTime.set(new Date());
          setTimeout(() => this.syncProgress.set(0), 2000);
          this.showSuccessNotification('Sync completed successfully');
          this.loadLoungeData(lounge.id);
        },
        error: (error) => {
          clearInterval(progressInterval);
          this.syncProgress.set(0);
          console.error('Error syncing:', error);
          this.showErrorNotification('Sync failed');
        },
      });
  }

  rebootDevice(device: DeviceStatus): void {
    if (!confirm(`Are you sure you want to reboot device ${device.deviceId}?`)) return;

    this.tvSyncService
      .sendRemoteCommand(device.deviceId, 'reboot')
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.showSuccessNotification('Reboot command sent');
        },
        error: (error) => {
          console.error('Error rebooting device:', error);
          this.showErrorNotification('Failed to reboot device');
        },
      });
  }

  refreshDevice(device: DeviceStatus): void {
    this.tvSyncService
      .sendRemoteCommand(device.deviceId, 'refresh')
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.showSuccessNotification('Refresh command sent');
        },
        error: (error) => {
          console.error('Error refreshing device:', error);
          this.showErrorNotification('Failed to refresh device');
        },
      });
  }

  // Auto Refresh
  setupAutoRefresh(): void {
    interval(30000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        const lounge = this.selectedLounge();
        if (lounge) {
          this.loadLoungeData(lounge.id);
        }
      });
  }

  // Utility Methods
  getStatusColor(status: string): string {
    switch (status) {
      case 'online':
        return '#10b981';
      case 'offline':
        return '#ef4444';
      case 'syncing':
        return '#f59e0b';
      case 'error':
        return '#dc2626';
      default:
        return '#6b7280';
    }
  }

  getStatusIcon(status: string): string {
    switch (status) {
      case 'online':
        return '✓';
      case 'offline':
        return '✕';
      case 'syncing':
        return '↻';
      case 'error':
        return '⚠';
      default:
        return '○';
    }
  }

  formatDate(date: string | undefined): string {
    if (!date) return 'N/A';
    return new Date(date).toLocaleString();
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    } else {
      return `${secs}s`;
    }
  }

  showSuccessNotification(message: string): void {
    // Implement your notification system here
    console.log('✓', message);
  }

  showErrorNotification(message: string): void {
    // Implement your notification system here
    console.error('✕', message);
  }

  setActiveTab(tab: string): void {
    this.activeTab.set(tab);
  }

  createLoungeAd(): void {
    const lounge = this.selectedLounge();
    if (!lounge) {
      return;
    }

    const payload: LoungeSpecificAdRequest = {
      loungeId: this.loungeAdForm.isDefaultForAll ? undefined : lounge.id,
      advertisementName: this.loungeAdForm.advertisementName.trim(),
      mediaUrl: this.loungeAdForm.mediaUrl.trim(),
      mediaType: this.loungeAdForm.mediaType,
      durationSeconds: Number(this.loungeAdForm.durationSeconds),
      priority: this.loungeAdForm.priority,
      isActive: this.loungeAdForm.isActive,
      isDefaultForAll: this.loungeAdForm.isDefaultForAll,
    };

    if (!payload.advertisementName || !payload.mediaUrl) {
      this.showErrorNotification('Name and media URL are required for lounge ads');
      return;
    }

    if (payload.durationSeconds <= 0 || payload.durationSeconds > 24) {
      this.showErrorNotification('Lounge ad duration must be between 1 and 24 seconds');
      return;
    }

    const slots = this.loungeAdSlots();
    if (slots && payload.durationSeconds > slots.remainingSeconds) {
      this.showErrorNotification(
        `Duration exceeds remaining ad window (${slots.remainingSeconds}s available)`
      );
      return;
    }

    this.tvSyncService
      .createLoungeAd(payload)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.showSuccessNotification('Lounge-specific advertisement created');
          this.resetLoungeAdForm();
          this.loadLoungeData(lounge.id);
        },
        error: (error) => {
          console.error('Error creating lounge ad:', error);
          this.showErrorNotification(error?.error?.error || 'Failed to create lounge ad');
        },
      });
  }

  onLoungeAdFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement | null;
    const file = input?.files?.[0] || null;
    this.selectedLoungeAdFile = file;
    this.loungeAdUploadError.set(null);

    if (!file) {
      return;
    }

    const mime = (file.type || '').toLowerCase();
    if (mime.startsWith('video/')) {
      this.loungeAdForm.mediaType = 'video';
    } else if (mime.startsWith('image/')) {
      this.loungeAdForm.mediaType = 'image';
    }
  }

  uploadSelectedLoungeAdMedia(): void {
    if (!this.selectedLoungeAdFile) {
      this.loungeAdUploadError.set('Select a media file first');
      return;
    }

    this.isUploadingLoungeAdMedia.set(true);
    this.loungeAdUploadError.set(null);

    this.adService
      .uploadMedia(this.selectedLoungeAdFile)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (response) => {
          this.loungeAdForm.mediaUrl = response.mediaUrl;
          if (response.mediaType === 'video' || response.mediaType === 'image') {
            this.loungeAdForm.mediaType = response.mediaType;
          }
          this.isUploadingLoungeAdMedia.set(false);
          this.showSuccessNotification('Media uploaded and linked to lounge ad form');
        },
        error: (error) => {
          console.error('Error uploading lounge ad media:', error);
          this.loungeAdUploadError.set(error?.error?.error || 'Media upload failed');
          this.isUploadingLoungeAdMedia.set(false);
        },
      });
  }

  scheduleTypeEntries(): { key: string; seconds: number }[] {
    const breakdown = this.loungeAdSlots()?.bookedByScheduleType;
    if (!breakdown) {
      return [];
    }

    return Object.entries(breakdown)
      .map(([key, seconds]) => ({ key, seconds: Number(seconds) || 0 }))
      .filter((item) => item.seconds > 0)
      .sort((a, b) => b.seconds - a.seconds);
  }

  cycleSegmentWidth(seconds: number): number {
    return Math.max(0, Math.min(100, (seconds / 30) * 100));
  }

  pickLoungeAdDuration(seconds: number): void {
    const availableSeconds = Math.max(1, Math.min(24, this.loungeAdSlots()?.remainingSeconds || 24));
    const normalized = Math.max(1, Math.min(availableSeconds, Math.floor(seconds || 0)));
    this.loungeAdForm.durationSeconds = normalized;
  }

  refreshLoungeAdSlots(): void {
    const lounge = this.selectedLounge();
    if (!lounge) {
      return;
    }
    this.loungeAdSlots.set(null);
    this.loungeAdSlotsLoadError.set(null);
    this.tvSyncService
      .getLoungeAdSlots(lounge.id)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (slots) => {
          this.loungeAdSlots.set(slots);
          this.loungeAdSlotsLoadError.set(null);
        },
        error: (error) => {
          console.error('Error refreshing lounge ad slot summary:', error);
          this.loungeAdSlots.set(null);
          this.loungeAdSlotsLoadError.set(error?.error?.error || 'Slot summary API request failed');
        },
      });
  }

  private resetLoungeAdForm(): void {
    this.loungeAdForm.advertisementName = '';
    this.loungeAdForm.mediaUrl = '';
    this.loungeAdForm.mediaType = 'image';
    this.loungeAdForm.durationSeconds = 4;
    this.loungeAdForm.priority = 'normal';
    this.loungeAdForm.isActive = true;
    this.loungeAdForm.isDefaultForAll = false;
    this.selectedLoungeAdFile = null;
    this.loungeAdUploadError.set(null);
  }

  deleteLoungeAd(ad: LoungeSpecificAd): void {
    const lounge = this.selectedLounge();
    if (!lounge) {
      return;
    }

    if (!confirm(`Delete lounge ad: ${ad.advertisementName}?`)) {
      return;
    }

    this.tvSyncService
      .deleteLoungeAd(ad.id)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.showSuccessNotification('Lounge ad deleted');
          this.loadLoungeData(lounge.id);
        },
        error: (error) => {
          console.error('Error deleting lounge ad:', error);
          this.showErrorNotification(error?.error?.error || 'Failed to delete lounge ad');
        },
      });
  }
}
