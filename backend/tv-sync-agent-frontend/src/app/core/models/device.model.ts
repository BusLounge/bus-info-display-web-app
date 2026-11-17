export interface Device {
  id: string;
  name: string;
  loungeName: string;
  status: DeviceStatus;
  location: string;
  lastSyncTime: Date;
  ipAddress: string;
  macAddress?: string;
  displayMode: DisplayMode;
  layoutMode: LayoutMode;
  language: string;
  broadcastsEnabled?: boolean;
  emergencyMessage?: EmergencyMessage;
  createdAt: Date;
  updatedAt: Date;
}

export enum DeviceStatus {
  ONLINE = 'online',
  OFFLINE = 'offline',
  SYNCING = 'syncing',
  ERROR = 'error'
}

export enum DisplayMode {
  SCHEDULES_ONLY = 'schedules_only',
  ADS_ONLY = 'ads_only',
  BOTH = 'both'
}

export enum LayoutMode {
  SPLIT_SCREEN = 'split_screen',
  FULL_SCREEN_ALTERNATE = 'full_screen_alternate'
}

export interface EmergencyMessage {
  id: string;
  message: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  isActive: boolean;
  startTime?: Date;
  endTime?: Date;
  backgroundColor?: string;
  textColor?: string;
}

export interface DeviceConfig {
  displayMode: DisplayMode;
  layoutMode: LayoutMode;
  language: string;
  broadcastsEnabled?: boolean;
  scheduleRefreshInterval: number;
  adRefreshInterval: number;
  emergencyMessageEnabled: boolean;
}
