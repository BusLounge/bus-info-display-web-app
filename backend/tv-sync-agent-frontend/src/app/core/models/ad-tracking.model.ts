export interface AdPlayback {
  id: string;
  deviceId: string;
  adId: string;
  adName: string;
  adDuration: number;
  playbackStartTime: Date;
  playbackEndTime?: Date;
  status: PlaybackStatus;
  impressions: number;
  completionRate: number;
  billingAmount?: number;
}

export enum PlaybackStatus {
  PLAYING = 'playing',
  COMPLETED = 'completed',
  SKIPPED = 'skipped',
  ERROR = 'error'
}

export interface AdBillingReport {
  deviceId: string;
  deviceName: string;
  totalImpressions: number;
  totalPlaybackTime: number;
  totalBillingAmount: number;
  reportPeriod: {
    startDate: Date;
    endDate: Date;
  };
  ads: AdPlaybackSummary[];
}

export interface AdPlaybackSummary {
  adId: string;
  adName: string;
  totalImpressions: number;
  totalPlaybackTime: number;
  averageCompletionRate: number;
  billingAmount: number;
}
