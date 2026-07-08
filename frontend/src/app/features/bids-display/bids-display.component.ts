import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute } from '@angular/router';
import { Subscription, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { environment } from '../../../environments/environment';
import { LocalBridgeService } from '../../core/services/local-bridge.service';
import { ScheduleService } from '../../core/services/schedule.service';
import { TrafficLevelService } from '../../core/services/traffic-level.service';
import { AdPlaybackLoggerService } from '../../core/services/ad-playback-logger.service';
import { AdvertisementService } from '../../core/services/advertisement.service';
import type { TVAdManifestItem } from '../../core/services/advertisement.service';

interface KioskAdvertisement {
  id: string;
  name: string;
  mediaUrl: string;
  mediaType: 'image' | 'video';
  durationMs: number;
  nextPlayAt?: string;
  playTimeSlots?: string[];
  playTimeSlot?: string;
}

interface KioskLoungeAd {
  id: string;
  name: string;
  mediaUrl: string;
  mediaType: 'image' | 'video';
  durationMs: number;
  isLocalCreated: boolean;
  isDefaultForAll?: boolean;
}

type DisplayLanguage = 'en' | 'si' | 'ta';
type TrafficLevel = 'Peak' | 'Moderate' | 'Off-Peak';

type TrafficCycleConfig = {
  trafficLevel: TrafficLevel;
  schedulePhaseMs: number;
  adPhaseMs: number;
};
type TranslationKey =
  | 'loadingLiveSchedule'
  | 'departures'
  | 'arrivals'
  | 'time'
  | 'busNo'
  | 'routeNumber'
  | 'destination'
  | 'origin'
  | 'remarks'
  | 'noDepartures'
  | 'noArrivals'
  | 'footerText';

interface LocalScheduleSnapshot {
  loungeId: string;
  departuresRaw?: {
    loungeName?: string;
    departures?: Array<{
      time?: string;
      busNo?: string;
      routeNumber?: string;
      destination?: string;
      remarks?: string;
      status?: string;
    }>;
  };
  arrivalsRaw?: {
    loungeName?: string;
    arrivals?: Array<{
      time?: string;
      busNo?: string;
      routeNumber?: string;
      origin?: string;
      remarks?: string;
      status?: string;
    }>;
  };
}

interface LocalAdsSnapshot {
  loungeGroup?: string;
  items?: Array<{
    id?: string;
    name?: string;
    mediaUrl?: string;
    mediaType?: string;
    localFile?: string;
    nextPlayAt?: string;
    playTimeSlots?: string[];
    playTimeSlot?: string;
  }>;
}

interface LocalBroadcastSnapshot {
  items?: Array<{
    id?: string;
    message?: string;
    priority?: string;
    displayDurationSeconds?: number;
    frequencySeconds?: number;
    startAt?: string;
    endAt?: string;
    isActive?: boolean;
    showOnLoungeTV?: boolean;
  }>;
}

interface LocalLoungeAdsSnapshot {
  items?: Array<{
    id?: string;
    advertisementName?: string;
    mediaUrl?: string;
    mediaType?: string;
    durationSeconds?: number;
    isActive?: boolean;
    isDefaultForAll?: boolean;
  }>;
}

interface KioskBroadcastMessage {
  id: string;
  message: string;
  priority: string;
  displayDurationSeconds: number;
  frequencySeconds: number;
  startAt: string;
  endAt?: string;
  isActive: boolean;
  showOnLoungeTV: boolean;
}

interface LocalAgentStatus {
  language?: string;
  displayMode?: string;
  layoutMode?: string;
  syncFrequencySeconds?: number;
  displayResolution?: string;
  broadcastsEnabled?: boolean;
  lastBroadcastSync?: string;
}

@Component({
  selector: 'app-bids-display',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './bids-display.component.html',
  styleUrl: './bids-display.component.scss'
})
export class BidsDisplayComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private bridgeService = inject(LocalBridgeService);
  private scheduleService = inject(ScheduleService);
  private trafficLevelService = inject(TrafficLevelService);
  private adLogger = inject(AdPlaybackLoggerService);
  private advertisementService = inject(AdvertisementService);
  private readonly localBridgeBaseUrl = 'http://localhost:3001/local';
  private networkStatusCheckInterval: any;
  private scheduleRefreshMs = 30000;
  private adsRefreshMs = 60000;
  private adCheckIntervalMs = 1000;
  private languageCheckIntervalMs = 3000; // Check language every 3 seconds
  private broadcastRefreshMs = 15000;
  private loungeAdsRefreshMs = 15000;
  private readonly alternateCycleMs = 30000;
  private readonly minimumAdDurationMs = 5000;
  private readonly loopAdDurationMs = 10000; // Duration for each ad in loop mode (10 seconds)
  
  currentTime: string = '';
  currentDate: string = '';
  loungeId: string = '';
  loungeName: string = '';
  departures: any[] = [];
  arrivals: any[] = [];
  advertisements: KioskAdvertisement[] = [];
  currentlyPlayingAd: KioskAdvertisement | KioskLoungeAd | null = null;
  currentSplitScreenAd: KioskAdvertisement | KioskLoungeAd | null = null; // For split screen ad display
  currentLoopAdIndex: number = 0; // Track which ad to show in loop mode
  kioskMode: boolean = false;
  loading: boolean = false;
  error: string = '';
  adError: string = '';
  isAdPlaying: boolean = false;
  debugNowUtc: string = '';
  debugNextDueAdId: string = 'N/A';
  debugNextDueNextPlayAt: string = 'N/A';
  debugMatchedSlotRange: string = 'N/A';
  debugNextSlotAt: string = 'N/A';
  debugNextSlotCountdown: string = 'N/A';
  private timeInterval: any;
  private scheduleInterval: any;
  private adInterval: any;
  private adCheckInterval: any;
  private languageInterval: any;
  private broadcastInterval: any;
  private loungeAdsInterval: any;
  private alternateCycleInterval: any;
  private loopAdInterval: any; // For looping ads in split screen or ads-only mode
  private adPlaybackTimeout: any;
  private paramsSub?: Subscription;
  private playedScheduleKeys = new Set<string>();
  private initialAdsLoaded: boolean = false;
  private lastKnownBroadcastSync: string = '';
  private hasLanguageOverride: boolean = false;
  isInternetConnected: boolean = true;
  displayScale: number = 1;
  currentLanguage: DisplayLanguage = 'en';
  currentDisplayMode: 'both' | 'schedules' | 'ads' = 'both';
  currentLayoutMode: 'split' | 'alternate' = 'split';
  currentDisplayResolution: string = '1920x1080';
  broadcastsEnabled: boolean = true;
  currentDisplayOrientation: 'landscape' | 'portrait' = 'landscape';
  broadcastMessages: KioskBroadcastMessage[] = [];
  activeBroadcastMessage: KioskBroadcastMessage | null = null;
  loungeSpecificAds: KioskLoungeAd[] = [];
  isAlternateAdPhase: boolean = false;
  alternateAdQueue: Array<{ ad: KioskAdvertisement | KioskLoungeAd; playMs: number }> = [];
  currentAlternateAd: KioskAdvertisement | KioskLoungeAd | null = null;
  private lastKnownAd: KioskAdvertisement | KioskLoungeAd | null = null;
  private loungeFallbackIndex: number = 0;
  private alternatePrimaryAdIndex: number = 0;
  
  private readonly FALLBACK_PLACEHOLDER_AD: KioskLoungeAd = {
    id: '__fallback_placeholder__',
    name: 'Lounge Fallback Placeholder',
    mediaUrl: 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTkyMCIgaGVpZ2h0PSIxMDgwIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxyZWN0IHdpZHRoPSIxOTIwIiBoZWlnaHQ9IjEwODAiIGZpbGw9IiMzMzMiLz48dGV4dCB4PSI5NjAiIHk9IjU0MCIgZm9udC1zaXplPSI0OCIgZmlsbD0iI2ZmZiIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZHk9Ii4zZW0iPkxvdW5nZSBBZCBTY3JlZW48L3RleHQ+PC9zdmc+',
    mediaType: 'image',
    durationMs: 10000,
    isLocalCreated: false,
    isDefaultForAll: false
  };

  private readonly translations: Record<DisplayLanguage, Record<TranslationKey, string>> = {
    en: {
      loadingLiveSchedule: 'LOADING LIVE SCHEDULE...',
      departures: 'DEPARTURES',
      arrivals: 'ARRIVALS',
      time: 'TIME',
      busNo: 'BUS NO',
      routeNumber: 'ROUTE NUMBER',
      destination: 'DESTINATION',
      origin: 'ORIGIN',
      remarks: 'REMARKS',
      noDepartures: 'No departures available.',
      noArrivals: 'No arrivals available.',
      footerText: 'MAKE HAPPY JOURNEYS'
    },
    si: {
      loadingLiveSchedule: 'සජීවී කාලසටහන පූරණය වෙමින්...',
      departures: 'පිටත්වීම්',
      arrivals: 'පැමිණීම්',
      time: 'වේලාව',
      busNo: 'බස් අංකය',
      routeNumber: 'මාර්ග අංකය',
      destination: 'ගමනාන්තය',
      origin: 'ආරම්භය',
      remarks: 'සටහන්',
      noDepartures: 'පිටත්වීම් නොමැත.',
      noArrivals: 'පැමිණීම් නොමැත.',
      footerText: 'සුභ ගමන්!'
    },
    ta: {
      loadingLiveSchedule: 'நேரடி அட்டவணை ஏற்றப்படுகிறது...',
      departures: 'புறப்பாடுகள்',
      arrivals: 'வருகைகள்',
      time: 'நேரம்',
      busNo: 'பஸ் எண்',
      routeNumber: 'வழித்தட எண்',
      destination: 'இலக்கு',
      origin: 'தொடக்கம்',
      remarks: 'குறிப்புகள்',
      noDepartures: 'புறப்பாடுகள் இல்லை.',
      noArrivals: 'வருகைகள் இல்லை.',
      footerText: 'மகிழ்ச்சியான பயணம்!'
    }
  };

  get currentAdvertisement(): KioskAdvertisement | KioskLoungeAd | null {
    return this.currentlyPlayingAd;
  }

  // Determine if we should use looping ads (no scheduled time)
  get shouldLoopAds(): boolean {
    return this.currentDisplayMode === 'ads' ||
           (this.currentDisplayMode === 'both' && this.currentLayoutMode === 'split');
  }

  // Determine if we should use scheduled ads (with scheduled time)
  get shouldUseScheduledAds(): boolean {
    return this.currentDisplayMode === 'both' && this.currentLayoutMode === 'alternate';
  }

  // Get the current ad for split screen display
  get currentSplitAd(): KioskAdvertisement | KioskLoungeAd | null {
    return this.currentSplitScreenAd;
  }

  // In hybrid alternate layout, ad phases should be pure media without chrome.
  get isHybridAlternateAdDisplay(): boolean {
    return this.currentDisplayMode === 'both' &&
      this.currentLayoutMode === 'alternate' &&
      (this.isAlternateAdPhase || this.isAdPlaying);
  }

  ngOnInit() {
    this.updateInternetConnectionStatus();
    if (typeof window !== 'undefined') {
      window.addEventListener('online', this.handleOnlineStatusChange);
      window.addEventListener('offline', this.handleOnlineStatusChange);
      this.networkStatusCheckInterval = window.setInterval(() => this.updateInternetConnectionStatus(), 10000);
    }

    this.paramsSub = this.route.queryParams.subscribe(params => {
      this.loungeId = params['loungeId'] || '';
      this.loungeName = params['loungeName'] || '';
      
      // Apply settings from URL parameters
      this.applyDisplayMode(params['displayMode']);
      this.applyLayoutMode(params['layoutMode']);
      this.applyDisplayOrientation(params['displayOrientation']);
      this.applyLanguage(params['language'] || params['lang'], true);
      this.applySyncFrequency(params['syncFrequencySeconds']);
      this.applyDisplayResolution(params['displayResolution']);

      this.loadLanguageFromLocalStatus();
      this.kioskMode = (params['kiosk'] || '').toLowerCase() === 'true';

      if (this.kioskMode) {
        this.tryEnterFullscreen();
      }

      this.initialAdsLoaded = false;
      this.fetchSchedule();
      this.startScheduleRefresh();
      this.startAdvertisementRefresh();
      this.startAdPlaybackWatcher();
      this.startLanguagePolling();
      this.startBroadcastRefresh();
      this.fetchBroadcastMessages();
      this.startLoungeAdsRefresh();
      this.fetchLoungeSpecificAds();
      this.startAlternateModeCycle();
      this.startLoopAdRotation();
    });

    this.updateTime();
    this.timeInterval = setInterval(() => this.updateTime(), 1000);
  }

  t(key: TranslationKey): string {
    return this.translations[this.currentLanguage][key] || this.translations.en[key];
  }

  ngOnDestroy() {
    if (typeof window !== 'undefined') {
      window.removeEventListener('online', this.handleOnlineStatusChange);
      window.removeEventListener('offline', this.handleOnlineStatusChange);
    }
    if (this.networkStatusCheckInterval) {
      clearInterval(this.networkStatusCheckInterval);
    }
    if (this.timeInterval) {
      clearInterval(this.timeInterval);
    }
    if (this.scheduleInterval) {
      clearInterval(this.scheduleInterval);
    }
    if (this.adInterval) {
      clearInterval(this.adInterval);
    }
    if (this.adCheckInterval) {
      clearInterval(this.adCheckInterval);
    }
    if (this.languageInterval) {
      clearInterval(this.languageInterval);
    }
    if (this.broadcastInterval) {
      clearInterval(this.broadcastInterval);
    }
    if (this.loungeAdsInterval) {
      clearInterval(this.loungeAdsInterval);
    }
    if (this.alternateCycleInterval) {
      clearInterval(this.alternateCycleInterval);
    }
    if (this.loopAdInterval) {
      clearInterval(this.loopAdInterval);
    }
    if (this.adPlaybackTimeout) {
      clearTimeout(this.adPlaybackTimeout);
    }
    this.paramsSub?.unsubscribe();
  }

  private handleOnlineStatusChange = () => {
    this.updateInternetConnectionStatus();
  };

  private updateInternetConnectionStatus(): void {
    if (typeof navigator === 'undefined') {
      this.isInternetConnected = true;
      return;
    }

    this.isInternetConnected = navigator.onLine;
  }

  private startScheduleRefresh() {
    if (this.scheduleInterval) {
      clearInterval(this.scheduleInterval);
    }
    this.scheduleInterval = setInterval(() => this.fetchSchedule(true), this.scheduleRefreshMs);
  }

  private startAdvertisementRefresh() {
    if (this.adInterval) {
      clearInterval(this.adInterval);
    }
    this.adInterval = setInterval(() => this.fetchAdvertisementsForLounge(), this.adsRefreshMs);
  }

  private startAdPlaybackWatcher() {
    if (this.adCheckInterval) {
      clearInterval(this.adCheckInterval);
    }

    // Only watch for scheduled ads in hybrid + alternate mode
    this.adCheckInterval = setInterval(() => {
      if (this.shouldUseScheduledAds) {
        this.tryStartScheduledAdvertisement();
      }
    }, this.adCheckIntervalMs);
  }

  private startLoopAdRotation() {
    if (this.loopAdInterval) {
      clearInterval(this.loopAdInterval);
    }

    // Start immediately to show the first ad
    this.updateLoopingAd();

    // Rotate ads every loopAdDurationMs
    this.loopAdInterval = setInterval(() => {
      if (this.shouldLoopAds) {
        this.updateLoopingAd();
      }
    }, this.loopAdDurationMs);
  }

  private updateLoopingAd(): void {
    const pool = this.getLoopingAdsPool();
    if (pool.length === 0) {
      this.currentSplitScreenAd = null;
      this.currentlyPlayingAd = null;
      this.isAdPlaying = false;
      return;
    }

    // Get the next ad in the loop
    const ad = pool[this.currentLoopAdIndex % pool.length];

    // Log looping ad playback in split screen mode
    if (this.currentDisplayMode === 'both' && this.currentLayoutMode === 'split') {
      const currentTraffic = this.trafficLevelService.getCurrentTrafficLevel();
      this.adLogger.logAdPlayback(ad, this.loopAdDurationMs, currentTraffic);
    }

    // Update based on mode
    if (this.currentDisplayMode === 'both' && this.currentLayoutMode === 'split') {
      // Split screen mode: show ad in split screen area
      this.currentSplitScreenAd = ad;
      this.currentlyPlayingAd = null;
      this.isAdPlaying = false;
      this.lastKnownAd = ad;
      console.log('[LOOPING_AD] Split screen:', ad.name);
    } else if (this.currentDisplayMode === 'ads') {
      // Ads only mode: show ad in full screen ads-only container
      this.currentSplitScreenAd = null;
      this.currentlyPlayingAd = ad;
      this.isAdPlaying = false;
      this.lastKnownAd = ad;
      
      // Log ad playback
      const currentTraffic = this.trafficLevelService.getCurrentTrafficLevel();
      this.adLogger.logAdPlayback(ad, this.loopAdDurationMs, currentTraffic);
      console.log('[LOOPING_AD] Full screen:', ad.name);
    }

    // Move to next ad for next rotation
    this.currentLoopAdIndex = (this.currentLoopAdIndex + 1) % pool.length;
  }

  private getLoopingAdsPool(): Array<KioskAdvertisement | KioskLoungeAd> {
    // 1. Prefer lounge-specific ads synced by tv-sync-agent
    if (this.loungeSpecificAds.length > 0) {
      console.log('[AD_POOL] Using lounge-specific ads:', this.loungeSpecificAds.length);
      return this.loungeSpecificAds;
    }

    // 2. Then try unscheduled company ads (ads without nextPlayAt)
    const unscheduledCompanyAds = this.advertisements.filter(ad => !ad.nextPlayAt);
    if (unscheduledCompanyAds.length > 0) {
      console.log('[AD_POOL] Using unscheduled company ads:', unscheduledCompanyAds.length);
      return unscheduledCompanyAds;
    }

    // 3. Use all company ads if no unscheduled ones
    if (this.advertisements.length > 0) {
      console.log('[AD_POOL] Using all company ads (including scheduled):', this.advertisements.length);
      return this.advertisements;
    }

    // 4. Use last known ad if available
    if (this.lastKnownAd) {
      console.log('[AD_POOL] Using last known ad as fallback:', this.lastKnownAd.name);
      return [this.lastKnownAd];
    }

    // 5. Show placeholder ad indicating lounge fallback is unavailable (prevents blank screen)
    console.warn('[AD_POOL] No ads available - using fallback placeholder');
    return [this.FALLBACK_PLACEHOLDER_AD];
  }

  private startLanguagePolling() {
    if (this.languageInterval) {
      clearInterval(this.languageInterval);
    }

    // Poll for language changes every 3 seconds
    this.languageInterval = setInterval(() => {
      this.loadLanguageFromLocalStatus();
    }, this.languageCheckIntervalMs);
  }

  private startBroadcastRefresh() {
    if (this.broadcastInterval) {
      clearInterval(this.broadcastInterval);
    }

    this.broadcastInterval = setInterval(() => {
      this.fetchBroadcastMessages();
    }, this.broadcastRefreshMs);
  }

  private startLoungeAdsRefresh() {
    if (this.loungeAdsInterval) {
      clearInterval(this.loungeAdsInterval);
    }

    this.loungeAdsInterval = setInterval(() => {
      this.fetchLoungeSpecificAds();
    }, this.loungeAdsRefreshMs);
  }

  private startAlternateModeCycle() {
    if (this.alternateCycleInterval) {
      clearInterval(this.alternateCycleInterval);
    }

    this.alternateCycleInterval = setInterval(() => {
      this.evaluateAlternateModePhase();
    }, 1000);

    this.evaluateAlternateModePhase();
  }

  private evaluateAlternateModePhase(): void {
    const isAlternate = this.currentDisplayMode === 'both' && this.currentLayoutMode === 'alternate';
    if (!isAlternate) {
      this.isAlternateAdPhase = false;
      this.currentAlternateAd = null;
      return;
    }

    const nowMs = Date.now();
    const cycleConfig = this.getCurrentTrafficCycleConfig();
    const phaseMs = nowMs % this.alternateCycleMs;
    const inAdPhase = phaseMs >= cycleConfig.schedulePhaseMs;

    if (inAdPhase && !this.isAlternateAdPhase) {
      this.isAlternateAdPhase = true;
      this.startAlternateAdPhaseQueue();
    } else if (!inAdPhase && this.isAlternateAdPhase) {
      if (this.shouldKeepScheduledAdInSchedulePhase(nowMs)) {
        return;
      }

      this.isAlternateAdPhase = false;
      this.currentAlternateAd = null;
      this.stopAdvertisementPlayback();
    }
  }

  private shouldKeepScheduledAdInSchedulePhase(nowMs: number): boolean {
    if (!this.currentAlternateAd) {
      return false;
    }

    const currentAd = this.advertisements.find((item) => item.id === this.currentAlternateAd?.id);
    if (!currentAd) {
      return false;
    }

    const slotStart = this.getCurrentMatchingSlotStartIso(currentAd, nowMs);
    if (slotStart) {
      return true;
    }

    return false;
  }

  private startAlternateAdPhaseQueue(): void {
    const queue = this.buildAlternateAdQueue();
    if (!queue.length) {
      // No ad available right now; keep schedules visible and avoid blank ad phase.
      this.isAlternateAdPhase = false;
      this.currentAlternateAd = null;
      return;
    }
    this.alternateAdQueue = queue;
    this.playNextAlternateAd();
  }

  private buildAlternateAdQueue(): Array<{ ad: KioskAdvertisement | KioskLoungeAd; playMs: number }> {
    const cycleConfig = this.getCurrentTrafficCycleConfig();
    let remainingMs = cycleConfig.adPhaseMs;
    const queue: Array<{ ad: KioskAdvertisement | KioskLoungeAd; playMs: number }> = [];
    let loopIterations = 0;
    const maxIterations = 100; // Prevent infinite loops

    console.log('[ALTERNATE_QUEUE] Building ad queue for', Math.round(remainingMs / 1000), 'seconds');
    console.log('[ALTERNATE_QUEUE] Traffic split:', cycleConfig.trafficLevel, 'schedule', Math.round(cycleConfig.schedulePhaseMs / 1000), 's, ad', Math.round(cycleConfig.adPhaseMs / 1000), 's');
    console.log('[ALTERNATE_QUEUE] Current state - loungeSpecificAds:', this.loungeSpecificAds.length, 
      'advertisements:', this.advertisements.length, 'lastKnownAd:', this.lastKnownAd?.name);

    while (remainingMs > 0 && loopIterations < maxIterations) {
      loopIterations++;
      const defaultAd = this.getNextDefaultAd();
      if (!defaultAd) {
        console.warn('[ALTERNATE_QUEUE] No more ads available after', queue.length, 'items');
        break;
      }
      
      const rawDurationMs = Number(defaultAd.durationMs);
      const safeDurationMs = Number.isFinite(rawDurationMs) && rawDurationMs > 0
        ? rawDurationMs
        : this.loopAdDurationMs;
      
      // If ad is longer than remaining time, truncate it to fit
      const playMs = Math.min(Math.max(safeDurationMs, 1000), remainingMs);
      queue.push({ ad: defaultAd, playMs });
      
      console.log('[ALTERNATE_QUEUE] Added ad:', defaultAd.name, '- Duration:', Math.round(playMs / 1000) + 's');
      remainingMs -= playMs;
    }

    // If the queue cannot fill the full 24-second ad phase, fill the remainder using default lounge ads.
    const defaultLoungeAds = this.getDefaultLoungeAds();
    let defaultIndex = 0;
    while (remainingMs > 0 && defaultLoungeAds.length > 0 && defaultIndex < maxIterations) {
      defaultIndex++;
      const fallbackAd = defaultLoungeAds[(this.loungeFallbackIndex + defaultIndex) % defaultLoungeAds.length];
      const fallbackRawMs = Number(fallbackAd.durationMs);
      const fallbackSafeMs = Number.isFinite(fallbackRawMs) && fallbackRawMs > 0
        ? fallbackRawMs
        : this.loopAdDurationMs;
      const fallbackPlayMs = Math.min(Math.max(fallbackSafeMs, 1000), remainingMs);
      queue.push({ ad: fallbackAd, playMs: fallbackPlayMs });
      remainingMs -= fallbackPlayMs;
    }

    if (remainingMs > 0) {
      console.log('[ALTERNATE_QUEUE] Remaining time not filled:', Math.round(remainingMs / 1000), 'seconds');
    }

    console.log('[ALTERNATE_QUEUE] Queue built with', queue.length, 'items, total duration:', 
      Math.round((cycleConfig.adPhaseMs - remainingMs) / 1000), 'seconds');

    return queue;
  }

  private getDefaultLoungeAds(): Array<KioskAdvertisement | KioskLoungeAd> {
    // First preference: explicit default lounge ads
    const explicitDefaults = this.loungeSpecificAds.filter((item) => item.isDefaultForAll === true);
    if (explicitDefaults.length > 0) {
      console.log('[DEFAULT_ADS] Using explicit default lounge ads:', explicitDefaults.length);
      return explicitDefaults;
    }

    // Fallback: any lounge-specific ads if default flag is not set on any item
    if (this.loungeSpecificAds.length > 0) {
      console.log('[DEFAULT_ADS] Using all lounge-specific ads:', this.loungeSpecificAds.length);
      return this.loungeSpecificAds;
    }

    // Final fallback: any non-empty looping pool (company ads / lastKnownAd)
    const pool = this.getLoopingAdsPool();
    console.log('[DEFAULT_ADS] Using looping ads pool:', pool.length, 'items');
    return pool;
  }

  private getNextDefaultAd(): KioskAdvertisement | KioskLoungeAd | null {
    const nowMs = Date.now();
    const dueScheduled = this.getDueScheduledAdvertisement(nowMs);
    if (dueScheduled) {
      // Mark as played for this specific schedule key (slot or nextPlayAt)
      this.playedScheduleKeys.add(this.buildScheduleKey(dueScheduled, nowMs));
      return dueScheduled;
    }

    const pool = this.getLoopingAdsPool();
    if (!pool.length) {
      return null;
    }

    const ad = pool[this.loungeFallbackIndex % pool.length];
    this.loungeFallbackIndex = (this.loungeFallbackIndex + 1) % pool.length;
    return ad;
  }

  private getDueScheduledAdvertisement(nowMs: number): KioskAdvertisement | null {
    const dueAds = this.advertisements
      .filter(ad => {
        const slotStart = this.getCurrentMatchingSlotStartIso(ad, nowMs);
        if (slotStart) {
          return true;
        }

        const nextPlay = Date.parse(ad.nextPlayAt || '');
        return !Number.isNaN(nextPlay) && nextPlay <= nowMs;
      })
      .sort((a, b) => this.getDueSortTime(a, nowMs) - this.getDueSortTime(b, nowMs));

    return dueAds.find(ad => !this.playedScheduleKeys.has(this.buildScheduleKey(ad, nowMs))) || null;
  }

  private playNextAlternateAd(): void {
    if (!this.isAlternateAdPhase) {
      this.currentAlternateAd = null;
      return;
    }

    const nowMs = Date.now();
    const remainingPhaseMs = this.getRemainingCurrentAdPhaseMs(nowMs);
    if (remainingPhaseMs <= 0) {
      this.isAlternateAdPhase = false;
      this.currentAlternateAd = null;
      this.isAdPlaying = false;
      return;
    }

    const dueScheduled = this.getDueScheduledAdvertisement(nowMs);
    if (dueScheduled) {
      const dueRawDurationMs = Number(dueScheduled.durationMs);
      const dueSafeDurationMs = Number.isFinite(dueRawDurationMs) && dueRawDurationMs > 0
        ? dueRawDurationMs
        : this.loopAdDurationMs;
      const duePlayMs = Math.min(Math.max(dueSafeDurationMs, 1000), remainingPhaseMs);

      this.playedScheduleKeys.add(this.buildScheduleKey(dueScheduled, nowMs));
      this.currentAlternateAd = dueScheduled;
      this.isAdPlaying = true;
      this.lastKnownAd = dueScheduled;

      const currentTraffic = this.trafficLevelService.getCurrentTrafficLevel();
      this.adLogger.logAdPlayback(dueScheduled, duePlayMs, currentTraffic);

      if (this.adPlaybackTimeout) {
        clearTimeout(this.adPlaybackTimeout);
      }

      this.adPlaybackTimeout = setTimeout(() => {
        this.playNextAlternateAd();
      }, duePlayMs);

      return;
    }

    if (!this.alternateAdQueue.length) {
      // Refill queue so ad phase doesn't go blank between clips.
      this.alternateAdQueue = this.buildAlternateAdQueue();
      if (!this.alternateAdQueue.length) {
        // Nothing playable: fall back to schedules instead of showing empty ad slot.
        this.isAlternateAdPhase = false;
        this.currentAlternateAd = null;
        this.isAdPlaying = false;
        return;
      }
    }

    const nextItem = this.alternateAdQueue.shift();
    if (!nextItem) {
      this.currentAlternateAd = null;
      return;
    }

    this.currentAlternateAd = nextItem.ad;
    this.isAdPlaying = true;
    this.lastKnownAd = nextItem.ad;

    // Log ad playback for cost tracking
    const currentTraffic = this.trafficLevelService.getCurrentTrafficLevel();
    this.adLogger.logAdPlayback(nextItem.ad, nextItem.playMs, currentTraffic);

    if (this.adPlaybackTimeout) {
      clearTimeout(this.adPlaybackTimeout);
    }

    this.adPlaybackTimeout = setTimeout(() => {
      this.playNextAlternateAd();
    }, nextItem.playMs);
  }

  private getRemainingCurrentAdPhaseMs(nowMs: number): number {
    const cycleConfig = this.getCurrentTrafficCycleConfig();
    const phaseMs = nowMs % this.alternateCycleMs;
    if (phaseMs < cycleConfig.schedulePhaseMs) {
      return 0;
    }

    const elapsedAdMs = phaseMs - cycleConfig.schedulePhaseMs;
    return Math.max(0, cycleConfig.adPhaseMs - elapsedAdMs);
  }

  private getNextCompanyAd(): KioskAdvertisement | null {
    if (!this.advertisements.length) {
      return null;
    }
    const ad = this.advertisements[this.alternatePrimaryAdIndex % this.advertisements.length];
    this.alternatePrimaryAdIndex = (this.alternatePrimaryAdIndex + 1) % this.advertisements.length;
    return ad;
  }

  private getNextLoungeAd(): KioskLoungeAd | null {
    if (!this.loungeSpecificAds.length) {
      return null;
    }
    const ad = this.loungeSpecificAds[this.loungeFallbackIndex % this.loungeSpecificAds.length];
    this.loungeFallbackIndex = (this.loungeFallbackIndex + 1) % this.loungeSpecificAds.length;
    return ad;
  }

  private fetchLoungeSpecificAds(): void {
    this.bridgeService.get<LocalLoungeAdsSnapshot>('lounge-ads').subscribe({
      next: (snapshot) => {
        console.log('[LOUNGE_ADS] Fetched lounge ads snapshot:', snapshot.items?.length || 0, 'items');
        
        const mappedAds = (snapshot.items || [])
          .filter(item => item.isActive !== false && !!(item.mediaUrl || '').trim())
          .map((item, index) => ({
            id: item.id || `lounge-ad-${index}`,
            name: (item.advertisementName || 'Lounge Ad').trim(),
            mediaUrl: this.resolveLoungeAdMediaUrl(item.mediaUrl || ''),
            mediaType: this.inferMediaType(item.mediaType, item.mediaUrl || ''),
            durationMs: Math.max(1000, (Number(item.durationSeconds) || 4) * 1000),
            isLocalCreated: (item.mediaUrl || '').trim().startsWith('/local/media/'),
            isDefaultForAll: item.isDefaultForAll === true,
          }));

        // When local ads exist, use only those and ignore remote/default lounge ads.
        const localCreatedAds = mappedAds.filter(ad => ad.isLocalCreated);
        const nextLoungeAds = localCreatedAds.length > 0 ? localCreatedAds : mappedAds;
        
        if (nextLoungeAds.length > 0) {
          this.loungeSpecificAds = nextLoungeAds;
          console.log('[LOUNGE_ADS] Updated loungeSpecificAds with', nextLoungeAds.length, 'active ads');
          const defaultCount = nextLoungeAds.filter(ad => ad.isDefaultForAll).length;
          if (defaultCount > 0) {
            console.log('[LOUNGE_ADS] Contains', defaultCount, 'default lounge ads');
          }
        } else {
          console.warn('[LOUNGE_ADS] No active lounge ads found after filtering');
        }

        if (this.shouldLoopAds) {
          this.updateLoopingAd();
        }
      },
      error: (err) => {
        // Keep last known lounge ads to avoid display gaps on transient local bridge errors.
        console.error('[LOUNGE_ADS] Failed to fetch lounge ads:', err.message || err);
      }
    });
  }

  private resolveLoungeAdMediaUrl(mediaUrl: string): string {
    const trimmed = (mediaUrl || '').trim();
    if (!trimmed) {
      return '';
    }

    if (trimmed.startsWith('/local/media/')) {
      const fileName = trimmed.slice('/local/media/'.length);
      return `${this.localBridgeBaseUrl}/media/${encodeURIComponent(fileName)}`;
    }

    return this.toAbsoluteMediaUrl(trimmed);
  }

  private fetchSchedule(silentRefresh: boolean = false) {
    if (!silentRefresh) {
      this.loading = true;
    }
    this.error = '';
    this.loadLanguageFromLocalStatus();

    this.bridgeService.get<LocalScheduleSnapshot>('schedule').subscribe({
      next: (schedule) => {
        const snapshotLoungeId = (schedule.loungeId || '').trim();
        if (!this.loungeId && snapshotLoungeId) {
          this.loungeId = snapshotLoungeId;
        }

        if (this.hasUsableLocalSchedule(schedule)) {
          this.applyLocalScheduleSnapshot(schedule);
          this.loading = false;
          return;
        }

        this.fetchLiveScheduleFallback();
      },
      error: () => {
        this.fetchLiveScheduleFallback();
      }
    });
  }

  private fetchLiveScheduleFallback() {
    if (!this.loungeId) {
      this.loadScheduleFromLocalStore();
      return;
    }

    this.scheduleService.getScheduleByLounge(this.loungeId).subscribe({
      next: (schedule) => {
        this.departures = schedule.departures;
        this.arrivals = schedule.arrivals;
        if (schedule.loungeName) {
          this.loungeName = schedule.loungeName;
        }
        if (!this.initialAdsLoaded && this.loungeName.trim()) {
          this.initialAdsLoaded = true;
          this.fetchAdvertisementsForLounge();
        }
        this.loading = false;
      },
      error: () => {
        this.loadScheduleFromLocalStore();
      }
    });
  }

  private hasUsableLocalSchedule(schedule: LocalScheduleSnapshot): boolean {
    const snapshotLoungeId = (schedule.loungeId || '').trim();
    const requestedLoungeId = (this.loungeId || '').trim();
    const hasRows = !!schedule.departuresRaw?.departures?.length || !!schedule.arrivalsRaw?.arrivals?.length;

    if (!hasRows) {
      return false;
    }

    return !requestedLoungeId || !snapshotLoungeId || snapshotLoungeId === requestedLoungeId;
  }

  private applyLocalScheduleSnapshot(schedule: LocalScheduleSnapshot) {
    const departuresRaw = schedule.departuresRaw?.departures || [];
    const arrivalsRaw = schedule.arrivalsRaw?.arrivals || [];

    this.departures = departuresRaw.map((dep) => ({
      time: this.toDisplayTime(dep.time),
      busNo: dep.busNo || '-',
      routeNo: dep.routeNumber || '-',
      destination: dep.destination || '-',
      remarks: dep.remarks || dep.status || '',
      status: this.getStatusClass(dep.remarks || dep.status || ''),
      indicator: this.hasIndicator(dep.remarks || dep.status || '')
    }));

    this.arrivals = arrivalsRaw.map((arr) => ({
      time: this.toDisplayTime(arr.time),
      busNo: arr.busNo || '-',
      routeNo: arr.routeNumber || '-',
      origin: arr.origin || '-',
      remarks: arr.remarks || arr.status || '',
      status: this.getStatusClass(arr.remarks || arr.status || ''),
      indicator: this.hasIndicator(arr.remarks || arr.status || '')
    }));

    const loungeNameFromSchedule = schedule.departuresRaw?.loungeName || schedule.arrivalsRaw?.loungeName || '';
    if (loungeNameFromSchedule) {
      this.loungeName = loungeNameFromSchedule;
    }

    if (!this.initialAdsLoaded && this.loungeName.trim()) {
      this.initialAdsLoaded = true;
      this.fetchAdvertisementsForLounge();
    }
  }

  private fetchAdvertisementsForLounge() {
    this.adError = '';

    this.bridgeService.get<LocalAdsSnapshot>('ads').subscribe({
      next: (remoteItems) => {
        let skippedAvi = 0;
        const filtered = (remoteItems.items || [])
          .filter(item => {
            const mediaUrl = (item.localFile || item.mediaUrl || '').trim();
            if (!mediaUrl) return false;

            return true;
          })
          .map((item, index) => {
            const mediaType = this.inferMediaType(item.mediaType, item.mediaUrl || '');
            const absoluteMediaUrl = item.localFile
              ? `${this.localBridgeBaseUrl}/media/${encodeURIComponent(item.localFile)}`
              : this.toAbsoluteMediaUrl(item.mediaUrl || '');

            if (mediaType === 'video' && absoluteMediaUrl.toLowerCase().endsWith('.avi')) {
              skippedAvi += 1;
              return null;
            }

            return {
              id: item.id || `ad-${index}`,
              name: item.name || 'Advertisement',
              mediaUrl: absoluteMediaUrl,
              mediaType,
              durationMs: this.resolveAdDurationMs(undefined),
              nextPlayAt: (item.nextPlayAt || '').trim() || undefined,
              playTimeSlots: Array.isArray(item.playTimeSlots) ? item.playTimeSlots : undefined,
              playTimeSlot: (item.playTimeSlot || '').trim() || undefined,
            } as KioskAdvertisement;
          })
          .filter((ad): ad is KioskAdvertisement => ad !== null);

        if (filtered.length > 0) {
          this.advertisements = filtered;
        }
        if (skippedAvi > 0) {
          this.adError = `${skippedAvi} AVI ad(s) were skipped because AVI browser playback is unreliable. Use MP4 (H.264/AAC).`;
        }
        this.prunePlayedScheduleKeys();
        this.updateDebugState();
        this.tryStartScheduledAdvertisement();
        this.refreshAdsScheduleMetadataFromRemote();
        // Update looping ad display when ads are loaded
        if (this.shouldLoopAds) {
          this.updateLoopingAd();
        }
      },
      error: () => {
        this.advertisementService.getTVAdsManifest(this.loungeName).pipe(
          catchError(() => of([] as TVAdManifestItem[]))
        ).subscribe({
          next: (remoteItems) => {
            let skippedAvi = 0;
            const filtered = remoteItems
              .filter(item => {
                const mediaUrl = (item.mediaUrl || '').trim();
                if (!mediaUrl) return false;

                return true;
              })
              .map((item, index) => {
                const mediaType = this.inferMediaType(item.mediaType, item.mediaUrl || '');
                const absoluteMediaUrl = this.toAbsoluteMediaUrl(item.mediaUrl || '');

                if (mediaType === 'video' && absoluteMediaUrl.toLowerCase().endsWith('.avi')) {
                  skippedAvi += 1;
                  return null;
                }

                return {
                  id: item.id || `ad-${index}`,
                  name: item.advertisementName || 'Advertisement',
                  mediaUrl: absoluteMediaUrl,
                  mediaType,
                  durationMs: this.resolveAdDurationMs(item.mediaDuration),
                  nextPlayAt: (item.nextPlayAt || '').trim() || undefined,
                  playTimeSlots: Array.isArray(item.playTimeSlots) ? item.playTimeSlots : undefined,
                  playTimeSlot: (item.playTimeSlot || '').trim() || undefined,
                } as KioskAdvertisement;
              })
              .filter((ad): ad is KioskAdvertisement => ad !== null);

            if (filtered.length > 0) {
              this.advertisements = filtered;
            }
            if (skippedAvi > 0) {
              this.adError = `${skippedAvi} AVI ad(s) were skipped because AVI browser playback is unreliable. Use MP4 (H.264/AAC).`;
            }
            this.prunePlayedScheduleKeys();
            this.updateDebugState();
            this.tryStartScheduledAdvertisement();
            // Update looping ad display when ads are loaded
            if (this.shouldLoopAds) {
              this.updateLoopingAd();
            }
          },
          error: () => {
            // Fallback: Try loading from local store
            this.loadAdsFromLocalStore();
          }
        });
      }
    });
  }

  private refreshAdsScheduleMetadataFromRemote() {
    this.advertisementService.getTVAdsManifest(this.loungeName).pipe(
      catchError(() => of([] as TVAdManifestItem[]))
    ).subscribe({
      next: (remoteItems) => {
        if (!Array.isArray(remoteItems) || remoteItems.length === 0) {
          return;
        }

        let skippedAvi = 0;
        const remoteFiltered = remoteItems
          .filter(item => {
            const mediaUrl = (item.mediaUrl || '').trim();
            if (!mediaUrl) return false;
            return true;
          })
          .map((item, index) => {
            const mediaType = this.inferMediaType(item.mediaType, item.mediaUrl || '');
            const absoluteMediaUrl = this.toAbsoluteMediaUrl(item.mediaUrl || '');

            if (mediaType === 'video' && absoluteMediaUrl.toLowerCase().endsWith('.avi')) {
              skippedAvi += 1;
              return null;
            }

            return {
              id: item.id || `ad-${index}`,
              name: item.advertisementName || 'Advertisement',
              mediaUrl: absoluteMediaUrl,
              mediaType,
              durationMs: this.resolveAdDurationMs(item.mediaDuration),
              nextPlayAt: (item.nextPlayAt || '').trim() || undefined,
              playTimeSlots: Array.isArray(item.playTimeSlots) ? item.playTimeSlots : undefined,
              playTimeSlot: (item.playTimeSlot || '').trim() || undefined,
            } as KioskAdvertisement;
          })
          .filter((ad): ad is KioskAdvertisement => ad !== null);

        if (remoteFiltered.length === 0) {
          return;
        }

        const currentIds = new Set(this.advertisements.map((item) => item.id));
        const remoteIds = new Set(remoteFiltered.map((item) => item.id));
        const hasNewRemoteAd = Array.from(remoteIds).some((id) => !currentIds.has(id));

        if (hasNewRemoteAd || remoteFiltered.length !== this.advertisements.length) {
          this.advertisements = remoteFiltered;
          if (skippedAvi > 0) {
            this.adError = `${skippedAvi} AVI ad(s) were skipped because AVI browser playback is unreliable. Use MP4 (H.264/AAC).`;
          }
          this.prunePlayedScheduleKeys();
          this.updateDebugState();
          this.tryStartScheduledAdvertisement();
          if (this.shouldLoopAds) {
            this.updateLoopingAd();
          }
        }
      }
    });
  }

  private loadScheduleFromLocalStore() {
    this.bridgeService.get<LocalScheduleSnapshot>('schedule').subscribe({
      next: (schedule) => {
        const snapshotLoungeId = (schedule.loungeId || '').trim();
        if (!this.loungeId && snapshotLoungeId) {
          this.loungeId = snapshotLoungeId;
        }

        if (!this.hasUsableLocalSchedule(schedule)) {
          this.error = 'Failed to load schedule data. Offline storage unavailable.';
          this.loading = false;
          return;
        }

        this.applyLocalScheduleSnapshot(schedule);
        this.error = 'Schedule loaded from offline storage.';
        this.loading = false;
      },
      error: () => {
        this.error = 'Failed to load schedule data. Offline storage unavailable.';
        if (!this.initialAdsLoaded && this.loungeName.trim()) {
          this.initialAdsLoaded = true;
          this.fetchAdvertisementsForLounge();
        }
        this.loading = false;
      }
    });
  }

  private loadAdsFromLocalStore() {
    this.bridgeService.get<LocalAdsSnapshot>('ads').subscribe({
      next: (remoteItems) => {
        let skippedAvi = 0;
        const filtered = (remoteItems.items || [])
          .filter(item => {
            const mediaUrl = (item.localFile || item.mediaUrl || '').trim();
            if (!mediaUrl) return false;
            return true;
          })
          .map((item, index) => {
            const mediaType = this.inferMediaType(item.mediaType, item.mediaUrl || '');
            const absoluteMediaUrl = item.localFile
              ? `${this.localBridgeBaseUrl}/media/${encodeURIComponent(item.localFile)}`
              : this.toAbsoluteMediaUrl(item.mediaUrl || '');

            if (mediaType === 'video' && absoluteMediaUrl.toLowerCase().endsWith('.avi')) {
              skippedAvi += 1;
              return null;
            }

            return {
              id: item.id || `ad-${index}`,
              name: item.name || 'Advertisement',
              mediaUrl: absoluteMediaUrl,
              mediaType,
              durationMs: this.resolveAdDurationMs(undefined),
              nextPlayAt: (item.nextPlayAt || '').trim() || undefined,
              playTimeSlots: Array.isArray(item.playTimeSlots) ? item.playTimeSlots : undefined,
              playTimeSlot: (item.playTimeSlot || '').trim() || undefined,
            } as KioskAdvertisement;
          })
          .filter((ad): ad is KioskAdvertisement => ad !== null);

        if (filtered.length > 0) {
          this.advertisements = filtered;
        }
        if (skippedAvi > 0) {
          this.adError = `${skippedAvi} AVI ad(s) were skipped. Using offline ads storage.`;
        } else {
          this.adError = 'Ads loaded from offline storage.';
        }
        this.prunePlayedScheduleKeys();
        this.updateDebugState();
        this.tryStartScheduledAdvertisement();
        if (this.shouldLoopAds) {
          this.updateLoopingAd();
        }
      },
      error: () => {
        this.adError = 'Failed to load advertisements. Offline storage unavailable.';
        this.updateDebugState();
      }
    });
  }

  private loadLanguageFromLocalStatus() {
    this.bridgeService.get<LocalAgentStatus>('status').subscribe({
      next: (status) => {
        console.log('[DEBUG] Status loaded from local bridge:', status);
        
        // Only apply language from status if it wasn't overridden by URL
        if (!this.hasLanguageOverride) {
          this.applyLanguage(status.language, false);
        }

        // Sync display mode and layout mode
        if (status.displayMode) {
          this.applyDisplayMode(status.displayMode);
        }
        if (status.layoutMode) {
          this.applyLayoutMode(status.layoutMode);
        }
        if (status.syncFrequencySeconds) {
          this.applySyncFrequency(status.syncFrequencySeconds);
        }
        if (status.displayResolution) {
          this.applyDisplayResolution(status.displayResolution);
        }

        if (typeof status.broadcastsEnabled === 'boolean') {
          console.log('[DEBUG] Setting broadcastsEnabled from status:', status.broadcastsEnabled);
          this.broadcastsEnabled = status.broadcastsEnabled;
          this.evaluateBroadcastDisplay();
        } else {
          console.log('[DEBUG] broadcastsEnabled not in status, keeping default:', this.broadcastsEnabled);
        }

        if (status.lastBroadcastSync && status.lastBroadcastSync !== this.lastKnownBroadcastSync) {
          console.log('[DEBUG] lastBroadcastSync changed, refreshing broadcasts immediately:', status.lastBroadcastSync);
          this.lastKnownBroadcastSync = status.lastBroadcastSync;
          this.fetchBroadcastMessages();
        }
      },
      error: (err) => {
        console.log('[WARNING] Failed to load status from local bridge:', err);
        // Keep current language when local status is unavailable.
      }
    });
  }

  private applyDisplayMode(mode: string | undefined): void {
    const newMode = (mode || 'both').toLowerCase().replace(/[-\s]/g, '_');
    if (newMode === 'schedules_only' || newMode === 'schedule_only' || newMode === 'schedules') {
      this.currentDisplayMode = 'schedules';
      this.currentLayoutMode = 'alternate'; // Force full screen
    } else if (newMode === 'ads_only' || newMode === 'ad_only' || newMode === 'ads') {
      this.currentDisplayMode = 'ads';
      this.currentLayoutMode = 'alternate'; // Force full screen
    } else {
      this.currentDisplayMode = 'both';
    }
  }

  private applyLayoutMode(mode: string | undefined): void {
    if (this.currentDisplayMode === 'both') {
      const normalizedMode = (mode || '').toLowerCase().replace(/[-\s]/g, '_');
      if (normalizedMode === 'split_screen' || normalizedMode === 'split') {
        this.currentLayoutMode = 'split';
      } else if (normalizedMode === 'full_screen_alternate' || normalizedMode === 'alternate') {
        this.currentLayoutMode = 'alternate';
      } else {
        this.currentLayoutMode = 'split'; // Default for 'both'
      }
    } else {
      this.currentLayoutMode = 'split'; // Default for other modes
    }
  }

  private applyDisplayOrientation(value: string | undefined): void {
    const v = String(value || '').toLowerCase().trim();
    if (v === 'portrait' || v === 'portrait-primary') {
      this.currentDisplayOrientation = 'portrait';
    } else {
      this.currentDisplayOrientation = 'landscape';
    }
  }

  private applySyncFrequency(raw: string | number | undefined): void {
    const seconds = this.normalizeSyncFrequency(raw);
    if (!seconds) {
      return;
    }

    if (seconds === Math.round(this.scheduleRefreshMs / 1000)) {
      return;
    }

    this.scheduleRefreshMs = seconds * 1000;
    this.adsRefreshMs = seconds * 1000;
    this.broadcastRefreshMs = seconds * 1000;
    this.loungeAdsRefreshMs = seconds * 1000;
    this.languageCheckIntervalMs = seconds * 1000;

    if (this.scheduleInterval || this.adInterval || this.languageInterval || this.broadcastInterval || this.loungeAdsInterval) {
      this.startScheduleRefresh();
      this.startAdvertisementRefresh();
      this.startLanguagePolling();
      this.startBroadcastRefresh();
      this.startLoungeAdsRefresh();
    }
  }

  private normalizeSyncFrequency(raw: string | number | undefined): number | null {
    const parsed = Number(raw);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return null;
    }
    return Math.max(5, Math.floor(parsed));
  }

  private applyDisplayResolution(raw: string | undefined): void {
    const normalized = this.normalizeDisplayResolution(raw);
    if (!normalized) {
      return;
    }

    this.currentDisplayResolution = normalized;
    this.displayScale = this.getDisplayScale(normalized);
  }

  private normalizeDisplayResolution(raw: string | undefined): string | null {
    const value = (raw || '').trim().toLowerCase().replace(/\s+/g, '');
    switch (value) {
      case '1280x720':
      case '720p':
      case 'hd':
        return '1280x720';
      case '1920x1080':
      case '1080p':
      case 'fullhd':
        return '1920x1080';
      case '3840x2160':
      case '4k':
      case '4kultrahd':
        return '3840x2160';
      case '7680x4320':
      case '8k':
      case '8kultrahd':
        return '7680x4320';
      default:
        return null;
    }
  }

  private getDisplayScale(resolution: string): number {
    switch (resolution) {
      case '1280x720':
        return 0.667;
      case '3840x2160':
        return 2;
      case '7680x4320':
        return 4;
      case '1920x1080':
      default:
        return 1;
    }
  }

  private resetAdPlayback(): void {
    // Stop current ad playback
    this.stopAdvertisementPlayback();
    this.currentSplitScreenAd = null;
    this.currentAlternateAd = null;
    this.currentLoopAdIndex = 0;

    // Restart appropriate ad playback
    this.startLoopAdRotation();
  }

  private normalizeDisplayMode(raw: string | undefined): 'both' | 'schedules' | 'ads' | null {
    const value = (raw || '').trim().toLowerCase().replace(/[-\s]/g, '_');
    if (value === 'both') {
      return 'both';
    }
    if (value === 'schedules' || value === 'schedules_only' || value === 'schedule_only') {
      return 'schedules';
    }
    if (value === 'ads' || value === 'ads_only' || value === 'ad_only') {
      return 'ads';
    }
    return null;
  }

  private normalizeLayoutMode(raw: string | undefined): 'split' | 'alternate' | null {
    const value = (raw || '').trim().toLowerCase().replace(/[-\s]/g, '_');
    if (value === 'split' || value === 'split_screen') {
      return 'split';
    }
    if (value === 'alternate' || value === 'full_screen_alternate') {
      return 'alternate';
    }
    return null;
  }

  private resolveBridgeBaseUrl(value: string | undefined): string {
    const raw = (value || '').trim();
    if (!raw) {
      return '/local';
    }
    return raw.endsWith('/') ? raw.slice(0, -1) : raw;
  }

  private fetchBroadcastMessages() {
    this.bridgeService.get<LocalBroadcastSnapshot>('broadcasts').subscribe({
      next: (snapshot) => {
        console.log('[DEBUG] Broadcast snapshot received:', snapshot);
        this.broadcastMessages = (snapshot.items || [])
          .filter((item) => !!item.message)
          .map((item, index) => ({
            id: item.id || `broadcast-${index}`,
            message: (item.message || '').trim(),
            priority: (item.priority || 'normal').toLowerCase(),
            displayDurationSeconds: Math.max(1, Number(item.displayDurationSeconds) || 10),
            frequencySeconds: Math.max(1, Number(item.frequencySeconds) || 60),
            startAt: item.startAt || new Date().toISOString(),
            endAt: item.endAt,
            isActive: item.isActive !== false,
            showOnLoungeTV: item.showOnLoungeTV !== false,
          }));

        console.log('[DEBUG] Parsed broadcast messages:', this.broadcastMessages);
        console.log('[DEBUG] Broadcasts enabled:', this.broadcastsEnabled);
        this.evaluateBroadcastDisplay();
      },
      error: (err) => {
        console.error('[ERROR] Failed to fetch broadcast messages:', err);
        this.broadcastMessages = [];
        this.activeBroadcastMessage = null;
      },
    });
  }

  private evaluateBroadcastDisplay() {
    if (!this.broadcastsEnabled || !this.broadcastMessages.length) {
      console.log('[DEBUG] Broadcast display disabled:', {
        broadcastsEnabled: this.broadcastsEnabled,
        hasMessages: this.broadcastMessages.length > 0,
        messageCount: this.broadcastMessages.length,
      });
      this.activeBroadcastMessage = null;
      return;
    }

    const now = Date.now();
    const priorityRank: Record<string, number> = { critical: 3, high: 2, normal: 1, low: 0 };

    const activeMessages = this.broadcastMessages.filter((item) => this.isBroadcastActiveNow(item, now));
    activeMessages.sort((a, b) => {
      const rankA = priorityRank[a.priority] ?? 0;
      const rankB = priorityRank[b.priority] ?? 0;
      if (rankA !== rankB) {
        return rankB - rankA;
      }
      return a.startAt.localeCompare(b.startAt);
    });

    this.activeBroadcastMessage = activeMessages[0] || null;
    console.log('[DEBUG] Broadcast display evaluated:', {
      totalMessages: this.broadcastMessages.length,
      activeMessages: activeMessages.length,
      selected: this.activeBroadcastMessage?.message || 'none',
    });
  }

  private isBroadcastActiveNow(item: KioskBroadcastMessage, nowMs: number): boolean {
    if (!item.isActive || !item.showOnLoungeTV || !item.message.trim()) {
      console.log('[DEBUG] Broadcast failed flag check:', {
        message: item.message,
        isActive: item.isActive,
        showOnLoungeTV: item.showOnLoungeTV,
        hasMessage: !!item.message.trim(),
      });
      return false;
    }

    const startMs = Date.parse(item.startAt);
    if (Number.isNaN(startMs) || nowMs < startMs) {
      console.log('[DEBUG] Broadcast failed start time check:', {
        message: item.message,
        startAt: item.startAt,
        startMs,
        nowMs,
        passed: !Number.isNaN(startMs) && nowMs >= startMs,
      });
      return false;
    }

    if (item.endAt) {
      const endMs = Date.parse(item.endAt);
      if (!Number.isNaN(endMs) && nowMs > endMs) {
        console.log('[DEBUG] Broadcast failed end time check:', {
          message: item.message,
          endAt: item.endAt,
          endMs,
          nowMs,
        });
        return false;
      }
    }

    const frequencyMs = Math.max(1000, item.frequencySeconds * 1000);
    const durationMs = Math.max(1000, Math.min(item.displayDurationSeconds * 1000, frequencyMs));
    const elapsed = nowMs - startMs;
    const phase = elapsed % frequencyMs;
    const isActive = phase < durationMs;
    console.log('[DEBUG] Broadcast frequency check:', {
      message: item.message,
      frequency: item.frequencySeconds,
      duration: item.displayDurationSeconds,
      elapsed: Math.round(elapsed / 1000),
      phase: Math.round(phase / 1000),
      isActive,
    });
    return isActive;
  }

  private getCurrentTrafficCycleConfig(): TrafficCycleConfig {
    const currentLevel = this.trafficLevelService.getCurrentTrafficLevel();

    switch (currentLevel) {
      case 'Peak':
        return {
          trafficLevel: 'Peak',
          schedulePhaseMs: 20000,
          adPhaseMs: 10000,
        };
      case 'Moderate':
        return {
          trafficLevel: 'Moderate',
          schedulePhaseMs: 18000,
          adPhaseMs: 12000,
        };
      case 'Off-Peak':
      default:
        return {
          trafficLevel: 'Off-Peak',
          schedulePhaseMs: 15000,
          adPhaseMs: 15000,
        };
    }
  }

  private applyLanguage(raw: string | undefined, isOverride: boolean): void {
    const normalized = this.normalizeLanguage(raw);
    if (!normalized) {
      return;
    }
    this.currentLanguage = normalized;
    if (isOverride) {
      this.hasLanguageOverride = true;
    }
    this.updateTime();
  }

  private normalizeLanguage(raw: string | undefined): DisplayLanguage | null {
    const value = (raw || '').trim().toLowerCase();
    if (value === 'en' || value === 'si' || value === 'ta') {
      return value;
    }
    return null;
  }

  private getLocaleForLanguage(): string {
    switch (this.currentLanguage) {
      case 'si':
        return 'si-LK';
      case 'ta':
        return 'ta-LK';
      default:
        return 'en-GB';
    }
  }

  private toDisplayTime(value: string | null | undefined): string {
    if (!value) {
      return '-';
    }
    if (/^\d{2}:\d{2}$/.test(value)) {
      return value;
    }
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
      return value;
    }
    return parsed.toLocaleTimeString('en-GB', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    });
  }

  private getStatusClass(remarks: string): string {
    const lowerRemarks = (remarks || '').toLowerCase();
    if (lowerRemarks.includes('departed') || lowerRemarks.includes('arrived')) {
      return lowerRemarks.includes('departed') ? 'departed' : 'arrived';
    }
    if (lowerRemarks.includes('check in') || lowerRemarks.includes('boarding')) {
      return 'checkin';
    }
    if (lowerRemarks.includes('delay') || lowerRemarks.includes('delayed')) {
      return lowerRemarks.includes('delayed') ? 'delayed' : 'delay';
    }
    if (lowerRemarks.includes('expected') || lowerRemarks.includes('arriving')) {
      return 'expected';
    }
    if (lowerRemarks.includes('on time')) {
      return 'ontime';
    }
    return '';
  }

  private hasIndicator(remarks: string): boolean {
    const lowerRemarks = (remarks || '').toLowerCase();
    return (
      lowerRemarks.includes('check in') ||
      lowerRemarks.includes('boarding') ||
      lowerRemarks.includes('delay') ||
      lowerRemarks.includes('expected') ||
      lowerRemarks.includes('on time') ||
      lowerRemarks.includes('arriving')
    );
  }

  private tryStartScheduledAdvertisement() {
    if (!this.shouldUseScheduledAds || !this.advertisements.length) {
      return;
    }

    this.prunePlayedScheduleKeys();

    const nowMs = Date.now();
    const dueAds = this.advertisements
      .filter(ad => {
        const slotStart = this.getCurrentMatchingSlotStartIso(ad, nowMs);
        if (slotStart) {
          return true;
        }

        const nextPlay = Date.parse(ad.nextPlayAt || '');
        if (!Number.isNaN(nextPlay) && nextPlay <= nowMs) {
          return true;
        }

        return false;
      })
      .sort((a, b) => this.getDueSortTime(a, nowMs) - this.getDueSortTime(b, nowMs));

    const nextDue = dueAds.find(ad => !this.playedScheduleKeys.has(this.buildScheduleKey(ad, nowMs)));
    if (!nextDue) {
      return;
    }

    // In alternate mode, preempt fallback clips when a due scheduled slot arrives.
    if (this.isAdPlaying && this.currentAlternateAd?.id === nextDue.id) {
      return;
    }

    if (this.isAdPlaying && this.adPlaybackTimeout) {
      clearTimeout(this.adPlaybackTimeout);
    }

    this.startAdvertisementPlayback(nextDue);
  }
private startAdvertisementPlayback(ad: KioskAdvertisement) {
    if (this.shouldUseScheduledAds) {
      this.startAlternateScheduledPlayback(ad);
      return;
    }

    this.isAdPlaying = true;
    this.currentlyPlayingAd = ad;
    this.playedScheduleKeys.add(this.buildScheduleKey(ad, Date.now()));
    this.updateDebugState();

    // Log scheduled ad playback for cost tracking
    const currentTraffic = this.trafficLevelService.getCurrentTrafficLevel();
    this.adLogger.logAdPlayback(ad, ad.durationMs, currentTraffic);

    if (this.adPlaybackTimeout) {
      clearTimeout(this.adPlaybackTimeout);
    }

    this.adPlaybackTimeout = setTimeout(() => {
      this.stopAdvertisementPlayback();
    }, ad.durationMs);
  }

  private startAlternateScheduledPlayback(ad: KioskAdvertisement): void {
    const nowMs = Date.now();
    const remainingPhaseMs = this.getRemainingCurrentAdPhaseMs(nowMs);
    const defaultDurationMs = Number.isFinite(Number(ad.durationMs)) && Number(ad.durationMs) > 0
      ? Number(ad.durationMs)
      : this.loopAdDurationMs;

    // If currently in schedule phase, preempt into ad phase to avoid missing short slot windows.
    const playMs = remainingPhaseMs > 0
      ? Math.min(Math.max(defaultDurationMs, 1000), remainingPhaseMs)
      : Math.max(defaultDurationMs, 1000);

    this.isAlternateAdPhase = true;
    this.currentAlternateAd = ad;
    this.currentlyPlayingAd = null;
    this.isAdPlaying = true;
    this.lastKnownAd = ad;
    this.playedScheduleKeys.add(this.buildScheduleKey(ad, nowMs));
    this.updateDebugState();

    const currentTraffic = this.trafficLevelService.getCurrentTrafficLevel();
    this.adLogger.logAdPlayback(ad, playMs, currentTraffic);

    if (this.adPlaybackTimeout) {
      clearTimeout(this.adPlaybackTimeout);
    }

    this.adPlaybackTimeout = setTimeout(() => {
      this.playNextAlternateAd();
    }, playMs);
  }

  private stopAdvertisementPlayback() {
    this.isAdPlaying = false;
    this.currentlyPlayingAd = null;
    if (!this.isAlternateAdPhase) {
      this.currentAlternateAd = null;
    }
    this.updateDebugState();
  }

  private buildScheduleKey(ad: KioskAdvertisement, nowMs: number = Date.now()): string {
    const slotStart = this.getCurrentMatchingSlotStartIso(ad, nowMs);
    if (slotStart) {
      return `${ad.id}|slot|${slotStart}`;
    }
    return `${ad.id}|next|${ad.nextPlayAt || ''}`;
  }

  private prunePlayedScheduleKeys() {
    const validIds = new Set(this.advertisements.map(ad => ad.id));
    const nowMs = Date.now();
    for (const key of Array.from(this.playedScheduleKeys)) {
      const adId = key.split('|')[0] || '';
      if (!validIds.has(adId)) {
        this.playedScheduleKeys.delete(key);
        continue;
      }

      if (this.isExpiredScheduleKey(key, nowMs)) {
        this.playedScheduleKeys.delete(key);
      }
    }
  }

  private isExpiredScheduleKey(key: string, nowMs: number): boolean {
    const parts = key.split('|');
    const mode = parts[1] || '';
    const value = parts.slice(2).join('|').trim();

    if (!value) {
      return true;
    }

    const parsed = Date.parse(value);
    if (Number.isNaN(parsed)) {
      return false;
    }

    // Keep the key only while the slot is still current, plus a short grace
    // period so we do not replay the same slot immediately after it ends.
    const graceMs = mode === 'slot' ? 60_000 : 5 * 60_000;
    return nowMs - parsed > graceMs;
  }

  private getDueSortTime(ad: KioskAdvertisement, nowMs: number): number {
    const slotStart = this.getCurrentMatchingSlotStartIso(ad, nowMs);
    if (slotStart) {
      return Date.parse(slotStart) || nowMs;
    }

    const nextPlay = Date.parse(ad.nextPlayAt || '');
    if (!Number.isNaN(nextPlay)) {
      return nextPlay;
    }

    return Number.MAX_SAFE_INTEGER;
  }

  private getCurrentMatchingSlotStartIso(ad: KioskAdvertisement, nowMs: number): string | undefined {
    const slots: string[] = [];

    if (Array.isArray(ad.playTimeSlots)) {
      for (const item of ad.playTimeSlots) {
        if (typeof item === 'string' && item.trim()) {
          slots.push(item.trim());
        }
      }
    }

    if (ad.playTimeSlot && ad.playTimeSlot.trim()) {
      slots.push(...ad.playTimeSlot.split(',').map((item) => item.trim()).filter((item) => item.length > 0));
    }

    for (const slot of slots) {
      const range = this.parseSlotRange(slot);
      if (!range) {
        continue;
      }

      if (range.startMs <= nowMs && nowMs < range.endMs) {
        return range.startIso;
      }
    }

    return undefined;
  }

  private parseSlotRange(slot: string): { startMs: number; endMs: number; startIso: string } | null {
    const text = (slot || '').trim();
    const match = text.match(/^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\s*-\s*(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})$/);
    if (!match) {
      return null;
    }

    // Stored slot timestamps are local date-times (no timezone suffix), so parse as local.
    const startLocal = new Date(match[1].replace(' ', 'T'));
    const endLocal = new Date(match[2].replace(' ', 'T'));
    const startMs = startLocal.getTime();
    const endMs = endLocal.getTime();
    if (Number.isNaN(startMs) || Number.isNaN(endMs) || endMs <= startMs) {
      return null;
    }

    return { startMs, endMs, startIso: startLocal.toISOString() };
  }

  private inferMediaType(inputType: string | undefined, mediaUrl: string): 'image' | 'video' {
    const type = (inputType || '').toLowerCase();
    if (type.includes('video')) return 'video';
    if (type.includes('image')) return 'image';

    const url = mediaUrl.toLowerCase();
    if (url.endsWith('.mp4') || url.endsWith('.webm') || url.endsWith('.mov') || url.endsWith('.m4v') || url.endsWith('.avi')) {
      return 'video';
    }
    return 'image';
  }

  private resolveAdDurationMs(mediaDurationSeconds: number | undefined): number {
    const parsed = Number(mediaDurationSeconds);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return Math.max(this.minimumAdDurationMs, 10000);
    }

    return Math.max(this.minimumAdDurationMs, Math.round(parsed * 1000));
  }

  onAdVideoError(ad: KioskAdvertisement | KioskLoungeAd) {
    this.adError = `Video playback failed for ${ad.name}. Prefer MP4 (H.264/AAC).`;
    this.advertisements = this.advertisements.filter(item => item.id !== ad.id);
    this.loungeSpecificAds = this.loungeSpecificAds.filter(item => item.id !== ad.id);
    if (this.currentlyPlayingAd?.id === ad.id) {
      this.stopAdvertisementPlayback();
    }
    if (this.currentAlternateAd?.id === ad.id) {
      this.currentAlternateAd = null;
    }
    this.updateDebugState();
  }

  private toAbsoluteMediaUrl(mediaUrl: string): string {
    const trimmed = (mediaUrl || '').trim();
    if (!trimmed) {
      return '';
    }

    if (/^https?:\/\//i.test(trimmed)) {
      return trimmed;
    }

    const apiBase = environment.apiUrl.replace(/\/api\/?$/, '');
    if (trimmed.startsWith('/')) {
      return `${apiBase}${trimmed}`;
    }

    return `${apiBase}/${trimmed}`;
  }

  private tryEnterFullscreen() {
    if (typeof document === 'undefined') {
      return;
    }

    setTimeout(() => {
      const root = document.documentElement as any;
      if (!document.fullscreenElement && root?.requestFullscreen) {
        root.requestFullscreen().catch(() => {
          // Some browsers require explicit user gesture; ignore failure.
        });
      }
    }, 100);
  }

  updateTime() {
    const now = new Date();
    const locale = this.getLocaleForLanguage();
    this.debugNowUtc = now.toISOString();
    this.currentTime = now.toLocaleTimeString(locale, {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });
    this.currentDate = now.toLocaleDateString(locale, {
      year: 'numeric',
      month: 'long',
      day: '2-digit'
    });
    if (this.currentLanguage === 'en') {
      this.currentDate = this.currentDate.toUpperCase();
    }

    this.updateDebugState();
    this.evaluateBroadcastDisplay();
  }

  private updateDebugState() {
    const nowMs = Date.now();
    const candidate = this.getNextDueCandidate();
    this.debugNextDueAdId = candidate?.id || 'N/A';
    this.debugNextDueNextPlayAt = candidate?.nextPlayAt || 'N/A';
    this.debugMatchedSlotRange = this.getLiveMatchedSlotDebug(nowMs);

    const nextSlot = this.getNextUpcomingSlotInfo(nowMs);
    this.debugNextSlotAt = nextSlot?.startLabel || 'N/A';
    this.debugNextSlotCountdown = nextSlot
      ? this.formatCountdown(Math.max(0, nextSlot.startMs - nowMs))
      : 'N/A';
  }

  private getNextUpcomingSlotInfo(nowMs: number): { startMs: number; startLabel: string; adId: string } | null {
    let next: { startMs: number; startLabel: string; adId: string } | null = null;

    for (const ad of this.advertisements) {
      const slotTexts = this.getAdSlotTexts(ad);

      for (const slot of slotTexts) {
        const range = this.parseSlotRange(slot);
        if (!range) {
          continue;
        }

        if (range.startMs <= nowMs) {
          continue;
        }

        if (!next || range.startMs < next.startMs) {
          next = {
            startMs: range.startMs,
            startLabel: slot,
            adId: ad.id,
          };
        }
      }
    }

    return next;
  }

  private getAdSlotTexts(ad: KioskAdvertisement): string[] {
    const slots: string[] = [];

    if (Array.isArray(ad.playTimeSlots)) {
      for (const item of ad.playTimeSlots) {
        if (typeof item === 'string' && item.trim()) {
          slots.push(item.trim());
        }
      }
    }

    if (ad.playTimeSlot && ad.playTimeSlot.trim()) {
      slots.push(...ad.playTimeSlot.split(',').map((item) => item.trim()).filter((item) => item.length > 0));
    }

    return slots;
  }

  private formatCountdown(remainingMs: number): string {
    const totalSeconds = Math.floor(remainingMs / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    if (hours > 0) {
      return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }

    return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  }

  private getLiveMatchedSlotDebug(nowMs: number): string {
    for (const ad of this.advertisements) {
      const match = this.getCurrentMatchingSlotRangeLabel(ad, nowMs);
      if (match) {
        return `${ad.id}: ${match}`;
      }
    }

    return 'N/A';
  }

  private getCurrentMatchingSlotRangeLabel(ad: KioskAdvertisement, nowMs: number): string | undefined {
    const slots = this.getAdSlotTexts(ad);

    for (const slot of slots) {
      const range = this.parseSlotRange(slot);
      if (!range) {
        continue;
      }

      if (range.startMs <= nowMs && nowMs < range.endMs) {
        return slot;
      }
    }

    return undefined;
  }

  private getNextDueCandidate(): KioskAdvertisement | null {
    const nowMs = Date.now();
    const candidates = this.advertisements
      .filter(ad => {
        const slotStart = this.getCurrentMatchingSlotStartIso(ad, nowMs);
        if (slotStart) {
          return true;
        }

        const nextPlay = Date.parse(ad.nextPlayAt || '');
        return !Number.isNaN(nextPlay);
      })
      .filter(ad => !this.playedScheduleKeys.has(this.buildScheduleKey(ad, nowMs)))
      .sort((a, b) => this.getDueSortTime(a, nowMs) - this.getDueSortTime(b, nowMs));

    return candidates[0] || null;
  }
}
