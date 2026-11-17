export interface TVSyncAgentConfig {
  id?: string;
  loungeId: string;
  loungeName: string;
  deviceId?: string;
  displayMode: DisplayMode;
  emergencyMessage?: EmergencyMessage;
  language: string;
  layoutOption: LayoutOption;
  adPlaybackTracking: AdPlaybackTracking[];
  deviceStatus: DeviceStatus;
  createdAt?: string;
  updatedAt?: string;
}

export type DisplayMode = 'schedules-only' | 'ads-only' | 'both';

export type LayoutOption = 'split-screen' | 'full-screen-alternate';

export interface EmergencyMessage {
  enabled: boolean;
  message: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  backgroundColor?: string;
  textColor?: string;
  displayDuration?: number;
  createdAt?: string;
}

export interface AdPlaybackTracking {
  adId: string;
  adName: string;
  playedAt: string;
  duration: number;
  completed: boolean;
  deviceId: string;
  loungeId: string;
  playbackQuality?: 'hd' | 'sd' | '4k';
}

export interface DeviceStatus {
  deviceId: string;
  status: 'online' | 'offline' | 'syncing' | 'error';
  lastSyncTime?: string;
  location?: DeviceLocation;
  ipAddress?: string;
  firmwareVersion?: string;
  batteryLevel?: number;
  screenResolution?: string;
  uptime?: number;
  memoryUsage?: number;
  cpuUsage?: number;
}

export interface DeviceLocation {
  latitude?: number;
  longitude?: number;
  address?: string;
  city?: string;
  country?: string;
}

export interface ScheduleDisplay {
  id: string;
  route: string;
  destination: string;
  departureTime: string;
  arrivalTime?: string;
  platform?: string;
  status: 'on-time' | 'delayed' | 'cancelled' | 'boarding';
  delay?: number;
}

export interface AdDisplay {
  id: string;
  name: string;
  mediaUrl: string;
  mediaType: 'image' | 'video';
  duration: number;
  priority: 'low' | 'medium' | 'high';
}

export interface SyncStatus {
  isSyncing: boolean;
  lastSync?: Date;
  nextSync?: Date;
  syncProgress?: number;
  errors?: string[];
}

export interface BillingReport {
  loungeId: string;
  loungeName: string;
  period: {
    start: string;
    end: string;
  };
  totalAdsPlayed: number;
  totalDuration: number;
  adBreakdown: {
    adId: string;
    adName: string;
    playCount: number;
    totalDuration: number;
    cost?: number;
  }[];
  totalCost?: number;
}

export interface LoungeSpecificAd {
  id: string;
  loungeId?: string;
  advertisementName: string;
  mediaUrl: string;
  mediaType: 'image' | 'video';
  durationSeconds: number;
  priority: string;
  isActive: boolean;
  isDefaultForAll: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface LoungeSpecificAdRequest {
  loungeId?: string;
  advertisementName: string;
  mediaUrl: string;
  mediaType: 'image' | 'video';
  durationSeconds: number;
  priority: string;
  isActive: boolean;
  isDefaultForAll: boolean;
}

export interface LoungeAdSlotSummary {
  scheduleWindowSeconds: number;
  adWindowSeconds: number;
  bookedSeconds: number;
  remainingSeconds: number;
  timeSlots?: LoungeAdTimeSlot[];
  availableSlots?: LoungeAdTimeSlot[];
  bookedByScheduleType?: Record<string, number>;
  effectiveLoungeGroups?: string[];
}

export interface LoungeAdTimeSlot {
  type: 'schedule' | 'company' | 'available' | string;
  label: string;
  startSecond: number;
  endSecond: number;
  durationSeconds: number;
  interactive: boolean;
}

export const SUPPORTED_LANGUAGES = [
  { code: 'en', name: 'English', flag: '🇬🇧' },
  { code: 'es', name: 'Español', flag: '🇪🇸' },
  { code: 'fr', name: 'Français', flag: '🇫🇷' },
  { code: 'de', name: 'Deutsch', flag: '🇩🇪' },
  { code: 'zh', name: '中文', flag: '🇨🇳' },
  { code: 'ja', name: '日本語', flag: '🇯🇵' },
  { code: 'hi', name: 'हिन्दी', flag: '🇮🇳' },
  { code: 'ar', name: 'العربية', flag: '🇸🇦' },
  { code: 'pt', name: 'Português', flag: '🇵🇹' },
  { code: 'ru', name: 'Русский', flag: '🇷🇺' },
];
