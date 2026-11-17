import { Injectable, signal, inject } from '@angular/core';
import { Device, DeviceStatus, DisplayMode, LayoutMode, DeviceConfig } from '../models';
import { ApiService, AgentStatus } from './api.service';

@Injectable({
  providedIn: 'root'
})
export class DeviceService {
  private apiService = inject(ApiService);
  private devices = signal<Device[]>([]);
  private currentDevice = signal<Device | null>(null);

  readonly devices$ = this.devices.asReadonly();
  readonly currentDevice$ = this.currentDevice.asReadonly();

  constructor() {
    // Load real device data from the backend
    this.loadDevices();
  }

  private loadDevices(): void {
    // Fetch real data from TV Sync Agent Go backend
    this.apiService.getAgentStatus().subscribe({
      next: (status: AgentStatus) => {
        const device = this.mapAgentStatusToDevice(status);
        this.devices.set([device]);

        // Auto-select the first (and only) device
        if (device) {
          this.currentDevice.set(device);
        }
      },
      error: (err) => {
        console.error('Failed to load device data from backend:', err);
        console.log('Make sure tv-sync-agent-go is running on port 3001');
        // Fall back to mock data if the backend is not available
        this.loadMockDevices();
      }
    });
  }

  private mapAgentStatusToDevice(status: AgentStatus): Device {
    // Determine display mode based on tvPurpose
    let displayMode = DisplayMode.BOTH;
    if (status.tvPurpose === 'schedule') {
      displayMode = DisplayMode.SCHEDULES_ONLY;
    } else if (status.tvPurpose === 'ads') {
      displayMode = DisplayMode.ADS_ONLY;
    }

    let layoutMode = LayoutMode.SPLIT_SCREEN;
    if (status.layoutMode === 'alternate') {
      layoutMode = LayoutMode.FULL_SCREEN_ALTERNATE;
    }

    // Determine device status
    let deviceStatus = DeviceStatus.ONLINE;
    if (status.lastScheduleError || status.lastAdsError) {
      deviceStatus = DeviceStatus.ERROR;
    }

    // Check if recently synced (within last 10 minutes)
    const now = new Date().getTime();
    const lastSync = new Date(status.lastScheduleSync || status.lastAdsSync).getTime();
    const diffMins = Math.floor((now - lastSync) / 60000);
    if (diffMins > 10) {
      deviceStatus = DeviceStatus.OFFLINE;
    }

    return {
      id: '1', // Single device for now
      name: 'TV-Display-Agent',
      loungeName: 'Colombo Premium LoungeLK', // From config.json
      status: deviceStatus,
      location: 'Configured from config.json',
      lastSyncTime: new Date(status.lastScheduleSync || status.lastAdsSync),
      ipAddress: 'localhost',
      macAddress: 'N/A',
      displayMode: displayMode,
      layoutMode: layoutMode,
      language: status.language,
      broadcastsEnabled: status.broadcastsEnabled,
      createdAt: new Date(status.startedAt),
      updatedAt: new Date()
    };
  }

  private loadMockDevices(): void {
    // Fallback mock data if backend is unavailable
    const mockDevices: Device[] = [
      {
        id: '1',
        name: 'TV-Display-001',
        loungeName: 'Premium Lounge A',
        status: DeviceStatus.ONLINE,
        location: 'Terminal 1, Gate 5',
        lastSyncTime: new Date(),
        ipAddress: '192.168.1.100',
        macAddress: '00:1B:44:11:3A:B7',
        displayMode: DisplayMode.BOTH,
        layoutMode: LayoutMode.SPLIT_SCREEN,
        language: 'en',
        broadcastsEnabled: true,
        createdAt: new Date('2024-01-01'),
        updatedAt: new Date()
      }
    ];

    this.devices.set(mockDevices);
  }

  getDevices(): Device[] {
    return this.devices();
  }

  getDevice(id: string): Device | undefined {
    return this.devices().find(d => d.id === id);
  }

  setCurrentDevice(device: Device): void {
    this.currentDevice.set(device);
  }

  updateDeviceConfig(deviceId: string, config: Partial<DeviceConfig>): void {
    // Update language via API if changed
    if (config.language) {
      this.apiService.updateLanguage(config.language).subscribe({
        next: (response) => {
          console.log('Language updated successfully:', response);
          this.updateLocalDeviceConfig(deviceId, config);
        },
        error: (err) => {
          console.error('Failed to update language:', err);
          // Still update locally even if API call fails
          this.updateLocalDeviceConfig(deviceId, config);
        }
      });
    } else if (config.displayMode) {
      // Map Angular enum to backend format
      const backendDisplayMode = this.mapDisplayModeToBackend(config.displayMode);
      this.apiService.updateDisplayMode(backendDisplayMode).subscribe({
        next: (response: { displayMode: string; message: string }) => {
          console.log('Display mode updated successfully:', response);
          this.updateLocalDeviceConfig(deviceId, config);
        },
        error: (err: unknown) => {
          console.error('Failed to update display mode:', err);
          this.updateLocalDeviceConfig(deviceId, config);
        }
      });
    } else if (config.layoutMode) {
      // Map Angular enum to backend format
      const backendLayoutMode = this.mapLayoutModeToBackend(config.layoutMode);
      this.apiService.updateLayoutMode(backendLayoutMode).subscribe({
        next: (response: { layoutMode: string; message: string }) => {
          console.log('Layout mode updated successfully:', response);
          this.updateLocalDeviceConfig(deviceId, config);
        },
        error: (err: unknown) => {
          console.error('Failed to update layout mode:', err);
          this.updateLocalDeviceConfig(deviceId, config);
        }
      });
    } else if (typeof config.broadcastsEnabled === 'boolean') {
      this.apiService.updateBroadcastsEnabled(config.broadcastsEnabled).subscribe({
        next: (response: { enabled: boolean; message: string }) => {
          console.log('Broadcast display setting updated successfully:', response);
          this.updateLocalDeviceConfig(deviceId, config);
        },
        error: (err: unknown) => {
          console.error('Failed to update broadcast display setting:', err);
          this.updateLocalDeviceConfig(deviceId, config);
        }
      });
    } else {
      // For other config changes, just update locally for now
      this.updateLocalDeviceConfig(deviceId, config);
    }
  }

  private mapDisplayModeToBackend(displayMode: DisplayMode): string {
    switch (displayMode) {
      case DisplayMode.SCHEDULES_ONLY:
        return 'schedules';
      case DisplayMode.ADS_ONLY:
        return 'ads';
      case DisplayMode.BOTH:
      default:
        return 'both';
    }
  }

  private mapLayoutModeToBackend(layoutMode: LayoutMode): string {
    switch (layoutMode) {
      case LayoutMode.SPLIT_SCREEN:
        return 'split';
      case LayoutMode.FULL_SCREEN_ALTERNATE:
        return 'alternate';
      default:
        return 'split';
    }
  }

  private updateLocalDeviceConfig(deviceId: string, config: Partial<DeviceConfig>): void {
    const devices = this.devices();
    const index = devices.findIndex(d => d.id === deviceId);

    if (index !== -1) {
      const updatedDevice = {
        ...devices[index],
        ...config,
        updatedAt: new Date()
      };

      const newDevices = [...devices];
      newDevices[index] = updatedDevice;
      this.devices.set(newDevices);

      if (this.currentDevice()?.id === deviceId) {
        this.currentDevice.set(updatedDevice);
      }
    }
  }

  refreshDeviceStatus(deviceId: string): void {
    // Fetch fresh status from the backend
    this.apiService.getAgentStatus().subscribe({
      next: (status: AgentStatus) => {
        const device = this.mapAgentStatusToDevice(status);
        const devices = this.devices();
        const index = devices.findIndex(d => d.id === deviceId);

        if (index !== -1) {
          const newDevices = [...devices];
          newDevices[index] = device;
          this.devices.set(newDevices);

          if (this.currentDevice()?.id === deviceId) {
            this.currentDevice.set(device);
          }
        }
      },
      error: (err) => {
        console.error('Failed to refresh device status:', err);
      }
    });
  }
}
