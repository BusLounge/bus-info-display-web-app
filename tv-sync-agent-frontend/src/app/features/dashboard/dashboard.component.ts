import { Component, computed, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DeviceService, EmergencyMessageService, AdTrackingService } from '../../core/services';
import { Device, DisplayMode, LayoutMode, DeviceStatus } from '../../core/models';
import {
  ApiService,
  BroadcastMessage,
  LocalLoungeAd,
  LocalLoungeAdCreateRequest,
  LocalLoungeAdUpdateRequest,
  LocalLoungeAdSlotSummary,
} from '../../core/services/api.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent {
  private deviceService = inject(DeviceService);
  private emergencyService = inject(EmergencyMessageService);
  private adTrackingService = inject(AdTrackingService);
  private apiService = inject(ApiService);

  devices = this.deviceService.devices$;
  selectedDevice = signal<Device | null>(null);

  displayModes = [
    { value: DisplayMode.SCHEDULES_ONLY, label: 'Schedules Only', icon: '📅' },
    { value: DisplayMode.ADS_ONLY, label: 'Advertisements Only', icon: '📺' },
    { value: DisplayMode.BOTH, label: 'Both (Hybrid)', icon: '🔀' }
  ];

  layoutModes = [
    { value: LayoutMode.SPLIT_SCREEN, label: 'Split Screen', icon: '⬜⬜', description: 'Show schedules and ads side by side' },
    { value: LayoutMode.FULL_SCREEN_ALTERNATE, label: 'Full Screen Alternate', icon: '🔄', description: 'Alternate between full screen schedules and ads' }
  ];

  languages = [
    { code: 'en', name: 'English', nativeName: 'English', flag: '🇱🇰' },
    { code: 'si', name: 'Sinhala', nativeName: 'සිංහල', flag: '🇱🇰' },
    { code: 'ta', name: 'Tamil', nativeName: 'தமிழ்', flag: '🇱🇰' }
  ];

  emergencyMessage = signal({
    text: '',
    priority: 'medium' as 'low' | 'medium' | 'high' | 'critical',
    isActive: false
  });

  displayTiming = {
    scheduleDuration: 6,
    scheduleInterval: 30,
    adDuration: 24
  };

  isRefreshing = signal(false);
  showTimingConfig = signal(false);
  showQuickActions = signal(false);
  broadcastsEnabled = signal(true);
  broadcasts = signal<BroadcastMessage[]>([]);
  activePanel = signal<'controls' | 'lounge-ads'>('controls');

  loungeAds = signal<LocalLoungeAd[]>([]);
  loungeAdSlots = signal<LocalLoungeAdSlotSummary | null>(null);
  loungeAdError = signal<string | null>(null);
  isCreatingLoungeAd = signal(false);
  editingLoungeAdId = signal<string | null>(null);
  selectedLoungeAdMediaFile: File | null = null;

  loungeAdForm = {
    advertisementName: '',
    isActive: true,
    isDefaultForAll: false,
  };

  stats = computed(() => {
    const devices = this.devices();
    return {
      total: devices.length,
      online: devices.filter(d => d.status === DeviceStatus.ONLINE).length,
      offline: devices.filter(d => d.status === DeviceStatus.OFFLINE).length,
      syncing: devices.filter(d => d.status === DeviceStatus.SYNCING).length
    };
  });

  constructor() {
    const devices = this.deviceService.getDevices();
    if (devices.length > 0) {
      this.selectedDevice.set(devices[0]);
    }
    this.loadBroadcasts();
    this.refreshLoungeAdPanel();
  }

  setActivePanel(panel: 'controls' | 'lounge-ads'): void {
    this.activePanel.set(panel);
    if (panel === 'lounge-ads') {
      this.refreshLoungeAdPanel();
    }
  }

  selectDevice(device: Device): void {
    this.selectedDevice.set(device);
    this.deviceService.setCurrentDevice(device);
  }

  updateDisplayMode(mode: DisplayMode): void {
    const device = this.selectedDevice();
    if (device) {
      this.deviceService.updateDeviceConfig(device.id, { displayMode: mode });
      this.selectedDevice.set({ ...device, displayMode: mode });
    }
  }

  updateLayoutMode(mode: LayoutMode): void {
    const device = this.selectedDevice();
    if (device) {
      this.deviceService.updateDeviceConfig(device.id, { layoutMode: mode });
      this.selectedDevice.set({ ...device, layoutMode: mode });
    }
  }

  updateLanguage(langCode: string): void {
    const device = this.selectedDevice();
    if (device) {
      this.deviceService.updateDeviceConfig(device.id, { language: langCode });
      this.selectedDevice.set({ ...device, language: langCode });
    }
  }

  updateBroadcastDisplay(enabled: boolean): void {
    const device = this.selectedDevice();
    if (device) {
      this.deviceService.updateDeviceConfig(device.id, { broadcastsEnabled: enabled });
      this.selectedDevice.set({ ...device, broadcastsEnabled: enabled });
    }
    this.broadcastsEnabled.set(enabled);
  }

  loadBroadcasts(): void {
    this.apiService.getBroadcastsEnabled().subscribe({
      next: (state: { enabled: boolean }) => {
        this.broadcastsEnabled.set(state.enabled);
      },
      error: (err: unknown) => {
        console.error('Failed to load broadcast display setting:', err);
      }
    });

    this.apiService.getBroadcasts().subscribe({
      next: (snapshot) => {
        this.broadcasts.set(snapshot.items || []);
      },
      error: (err: unknown) => {
        console.error('Failed to load local broadcasts:', err);
        this.broadcasts.set([]);
      }
    });
  }

  syncBroadcastsNow(): void {
    this.apiService.syncBroadcastsNow().subscribe({
      next: () => {
        this.loadBroadcasts();
      },
      error: (err: unknown) => {
        console.error('Failed to sync broadcast messages:', err);
      }
    });
  }

  refreshLoungeAdPanel(): void {
    this.loungeAdError.set(null);

    this.apiService.getLocalLoungeAds().subscribe({
      next: (snapshot: any) => {
        this.loungeAds.set(snapshot?.items || []);
      },
      error: (err: unknown) => {
        console.error('Failed to load local lounge ads:', err);
        this.loungeAds.set([]);
        this.loungeAdError.set('Failed to load local lounge ads.');
      }
    });

    this.apiService.getLocalLoungeAdSlots().subscribe({
      next: (slots) => {
        this.loungeAdSlots.set(slots);
      },
      error: (err: any) => {
        console.error('Failed to load local lounge ad slots:', err);
        this.loungeAdSlots.set(null);
        this.loungeAdError.set(this.resolveApiErrorMessage(err, 'Failed to load lounge ad slot summary.'));
      }
    });
  }

  createLocalLoungeAd(): void {
    if (this.editingLoungeAdId()) {
      this.updateLocalLoungeAd();
      return;
    }

    const payload: LocalLoungeAdCreateRequest = {
      advertisementName: this.loungeAdForm.advertisementName.trim(),
      isActive: this.loungeAdForm.isActive,
      isDefaultForAll: this.loungeAdForm.isDefaultForAll,
    };

    if (!payload.advertisementName) {
      this.loungeAdError.set('Advertisement name is required.');
      return;
    }
    if (!this.selectedLoungeAdMediaFile) {
      this.loungeAdError.set('Please choose a video/image file.');
      return;
    }

    this.isCreatingLoungeAd.set(true);
    this.loungeAdError.set(null);

    this.apiService.createLocalLoungeAd(payload, this.selectedLoungeAdMediaFile).subscribe({
      next: () => {
        this.isCreatingLoungeAd.set(false);
        this.resetLoungeAdForm();
        this.refreshLoungeAdPanel();
      },
      error: (err: any) => {
        this.isCreatingLoungeAd.set(false);
        this.loungeAdError.set(this.resolveApiErrorMessage(err, 'Failed to create local lounge ad.'));
      }
    });
  }

  updateLocalLoungeAd(): void {
    const id = this.editingLoungeAdId();
    if (!id) {
      return;
    }

    const payload: LocalLoungeAdUpdateRequest = {
      advertisementName: this.loungeAdForm.advertisementName.trim(),
      isActive: this.loungeAdForm.isActive,
      isDefaultForAll: this.loungeAdForm.isDefaultForAll,
    };

    if (!payload.advertisementName) {
      this.loungeAdError.set('Advertisement name is required.');
      return;
    }

    this.isCreatingLoungeAd.set(true);
    this.loungeAdError.set(null);

    this.apiService.updateLocalLoungeAd(id, payload, this.selectedLoungeAdMediaFile || undefined).subscribe({
      next: () => {
        this.isCreatingLoungeAd.set(false);
        this.resetLoungeAdForm();
        this.refreshLoungeAdPanel();
      },
      error: (err: any) => {
        this.isCreatingLoungeAd.set(false);
        this.loungeAdError.set(this.resolveApiErrorMessage(err, 'Failed to update local lounge ad.'));
      }
    });
  }

  startEditLoungeAd(ad: LocalLoungeAd): void {
    this.editingLoungeAdId.set(ad.id);
    this.loungeAdForm = {
      advertisementName: ad.advertisementName,
      isActive: ad.isActive,
      isDefaultForAll: ad.isDefaultForAll,
    };
    this.selectedLoungeAdMediaFile = null;
    this.loungeAdError.set(null);
  }

  cancelEditLoungeAd(): void {
    this.resetLoungeAdForm();
  }

  onLoungeAdFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement | null;
    this.selectedLoungeAdMediaFile = input?.files?.[0] || null;
  }

  deleteLoungeAd(ad: LocalLoungeAd): void {
    if (!confirm(`Delete lounge ad \"${ad.advertisementName}\"?`)) {
      return;
    }

    this.apiService.deleteLocalLoungeAd(ad.id).subscribe({
      next: () => {
        if (this.editingLoungeAdId() === ad.id) {
          this.resetLoungeAdForm();
        }
        this.refreshLoungeAdPanel();
      },
      error: (err: any) => {
        this.loungeAdError.set(this.resolveApiErrorMessage(err, 'Failed to delete lounge ad.'));
      }
    });
  }

  private resetLoungeAdForm(): void {
    this.editingLoungeAdId.set(null);
    this.selectedLoungeAdMediaFile = null;
    this.loungeAdForm = {
      advertisementName: '',
      isActive: true,
      isDefaultForAll: false,
    };
  }

  private resolveApiErrorMessage(err: any, fallback: string): string {
    const direct = err?.error;
    if (typeof direct === 'string' && direct.trim()) {
      return direct;
    }
    if (direct?.error && typeof direct.error === 'string' && direct.error.trim()) {
      return direct.error;
    }
    const status = err?.status;
    if (typeof status === 'number' && status > 0) {
      return `${fallback} (HTTP ${status})`;
    }
    return fallback;
  }

  sendEmergencyMessage(): void {
    const msg = this.emergencyMessage();
    if (msg.text.trim()) {
      this.emergencyService.createEmergencyMessage({
        message: msg.text,
        priority: msg.priority,
        isActive: true,
        backgroundColor: this.getEmergencyColor(msg.priority),
        textColor: '#ffffff'
      });
      this.emergencyMessage.set({ ...msg, isActive: true });
    }
  }

  clearEmergencyMessage(): void {
    this.emergencyService.clearEmergencyMessage();
    this.emergencyMessage.set({ text: '', priority: 'medium', isActive: false });
  }

  refreshDevice(deviceId: string): void {
    this.isRefreshing.set(true);
    window.location.reload();
  }

  forceSyncDevice(deviceId: string): void {
    this.isRefreshing.set(true);

    this.apiService.syncBroadcastsNow().subscribe({
      next: () => {
        this.deviceService.refreshDeviceStatus(deviceId);
        this.loadBroadcasts();

        setTimeout(() => {
          this.isRefreshing.set(false);
        }, 800);
      },
      error: (err: unknown) => {
        console.error('Failed to force sync broadcasts:', err);
        this.isRefreshing.set(false);
      }
    });
  }

  toggleQuickActions(): void {
    this.showQuickActions.update(val => !val);
  }

  getConnectionQuality(device: Device): { label: string; color: string; icon: string } {
    const now = new Date().getTime();
    const lastSync = new Date(device.lastSyncTime).getTime();
    const diffMins = Math.floor((now - lastSync) / 60000);

    if (diffMins < 1) return { label: 'Excellent', color: '#10b981', icon: '●●●●' };
    if (diffMins < 5) return { label: 'Good', color: '#22c55e', icon: '●●●○' };
    if (diffMins < 15) return { label: 'Fair', color: '#f59e0b', icon: '●●○○' };
    if (diffMins < 30) return { label: 'Poor', color: '#f97316', icon: '●○○○' };
    return { label: 'Disconnected', color: '#ef4444', icon: '○○○○' };
  }

  getDeviceUptime(device: Device): string {
    const now = new Date().getTime();
    const created = new Date(device.createdAt).getTime();
    const diffDays = Math.floor((now - created) / (1000 * 60 * 60 * 24));

    if (diffDays < 1) return 'Less than a day';
    if (diffDays < 30) return `${diffDays} days`;

    const months = Math.floor(diffDays / 30);
    if (months < 12) return `${months} ${months === 1 ? 'month' : 'months'}`;

    const years = Math.floor(months / 12);
    return `${years} ${years === 1 ? 'year' : 'years'}`;
  }

  restartDevice(deviceId: string): void {
    if (confirm('Are you sure you want to restart this device? This will briefly interrupt the display.')) {
      console.log('Restarting device:', deviceId);
    }
  }

  viewDeviceLogs(deviceId: string): void {
    console.log('Viewing logs for device:', deviceId);
  }

  getStatusColor(status: DeviceStatus): string {
    switch (status) {
      case DeviceStatus.ONLINE: return '#10b981';
      case DeviceStatus.OFFLINE: return '#ef4444';
      case DeviceStatus.SYNCING: return '#f59e0b';
      case DeviceStatus.ERROR: return '#dc2626';
      default: return '#6b7280';
    }
  }

  getEmergencyColor(priority: string): string {
    switch (priority) {
      case 'low': return '#3b82f6';
      case 'medium': return '#f59e0b';
      case 'high': return '#f97316';
      case 'critical': return '#dc2626';
      default: return '#6b7280';
    }
  }

  getTimeSinceSync(device: Device): string {
    const now = new Date().getTime();
    const lastSync = new Date(device.lastSyncTime).getTime();
    const diffMs = now - lastSync;
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;

    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ago`;
  }
}
