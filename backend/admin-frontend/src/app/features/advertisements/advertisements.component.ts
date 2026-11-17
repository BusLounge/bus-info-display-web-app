import { Component, OnInit, OnDestroy, PLATFORM_ID, inject } from '@angular/core';
import { CommonModule, isPlatformBrowser } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { forkJoin, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { HeaderComponent } from '../../shared/components/header/header.component';
import {
  AdvertisementService,
  AdvertisementGroup,
  LoungeService,
  Lounge,
  UploadMediaResponse,
  LoungeAdSlotSummary,
} from '../../core/services';

type LoungeSlotEntry = {
  loungeId: string;
  loungeName: string;
  summary: LoungeAdSlotSummary;
};

type LoungeRef = {
  loungeId: string;
  loungeName: string;
};

type TrafficLevel = 'Peak' | 'Moderate' | 'Off-Peak';

type TrafficTimePeriod = {
  name: string;
  startTime: string;
  endTime: string;
  trafficLevel: TrafficLevel;
  wrapsToNextDay?: boolean;
};

type ExactPlayableSlot = {
  startIso: string;
  endIso: string;
  startLabel: string;
  endLabel: string;
};

@Component({
  selector: 'app-advertisements',
  standalone: true,
  imports: [HeaderComponent, CommonModule, FormsModule],
  templateUrl: './advertisements.component.html',
  styleUrl: './advertisements.component.scss'
})
export class AdvertisementsComponent implements OnInit, OnDestroy {
  private platformId = inject(PLATFORM_ID);
  private loungeService = inject(LoungeService);
  activeTab: string = 'advertisements'; // Track active tab
  showModal = false;
  showGroupModal = false;
  showLoungeDropdown = false;
  showGroupDropdown = false;
  showConflictModal = false;
  isViewMode = false;
  isEditMode = false;
  isViewGroupMode = false;
  isEditGroupMode = false;
  selectedFile: File | null = null;
  scheduleType: string = 'Recurring';
  occursType: string = 'Daily';
  searchTerm: string = '';
  loungeSearchTerm: string = '';
  groupSearchTerm: string = '';
  isEmergencyCategory: boolean = false;
  selectedConflictOption: string = '';
  isSaving: boolean = false;

  conflictData: any = {
    message: '',
    removeOption: '',
    conflictingLounge: '',
    conflictingGroup: '',
    conflictingGroups: [],
    affectedLounges: [],
    timeSlot: ''
  };

  excludedLoungesByGroup: Record<string, Set<string>> = {};
  currentConflictGroupIndex: number = -1;

  selectedLounges: string[] = [];
  selectedGroups: string[] = [];
  groupAdvertisements: string[] = [];
  availableLounges: string[] = [];

  availableGroups: string[] = [];

  groupData = {
    id: '',
    groupName: '',
    lounges: []
  };
  
  // Form values for description
  occursOnceTime: string = '12:00';
  startingTime: string = '12:00';
  endingTime: string = '11:59';
  startDate: string = '2026-01-16';
  endDate: string = '2027-01-16';
  noEndDate: boolean = false;
  
  // One-time schedule fields
  oneTimeScheduleDate: string = '2026-01-16';
  oneTimeScheduleTime: string = '00:00';
  
  // Weekly schedule fields
  weeklyMonday: boolean = false;
  weeklyTuesday: boolean = false;
  weeklyWednesday: boolean = false;
  weeklyThursday: boolean = false;
  weeklyFriday: boolean = false;
  weeklySaturday: boolean = false;
  weeklySunday: boolean = false;
  
  // Monthly schedule fields
  monthlyDayOfMonth: number = 1;
  monthlyWeek: string = 'First';
  monthlyDay: string = 'Monday';
  monthlyType: string = 'dayOfMonth'; // 'dayOfMonth' or 'weekday'
  
  // Frequency type (for 'Occurs every' option)
  occursEveryInterval: number = 1;
  occursEveryUnit: string = 'Hourly';
  frequencyType: string = 'once'; // 'once' or 'every'
  
  // Recurrence interval (how many days/weeks/months between occurrences)
  recurrenceInterval: number = 1;
  
  // Max idle loop duration
  maxIdleLoopDuration: number = 60;
  
  // Enabled checkbox state
  isEnabled: boolean = true;
  
  advertisements: any[] = [];

  advertisementGroups: AdvertisementGroup[] = [];
  private refreshInterval: any;
  private readonly refreshIntervalMs = 30000;
  private filteredAdvertisementsCache: any[] = [];
  private filteredAdvertisementsKey: string = '';
  private filteredGroupsCache: AdvertisementGroup[] = [];
  private filteredGroupsKey: string = '';
  selectedLoungeName: string = '';
  selectedLoungeId: string = '';
  loungesCatalog: Lounge[] = [];

  slotSummariesByLounge: LoungeSlotEntry[] = [];
  slotHelperLoading: boolean = false;
  slotHelperError: string = '';
  commonAdWindowStartSecond: number = 20;
  commonAvailableSeconds: number = 0;
  interactiveDurationSlots: { durationSeconds: number; startSecond: number; endSecond: number }[] = [];
  selectedExactSlotStartIsos: string[] = [];
  selectedTrafficLevel: TrafficLevel | '' = '';
  selectedTimePeriodName: string = '';

  private readonly cycleDurationSeconds = 30;
  private readonly trafficCycleSeconds: Record<TrafficLevel, { schedule: number; ad: number }> = {
    'Peak': { schedule: 20, ad: 10 },
    'Moderate': { schedule: 18, ad: 12 },
    'Off-Peak': { schedule: 15, ad: 15 },
  };

  readonly trafficTimePeriods: TrafficTimePeriod[] = [
    { name: 'Early Morning', startTime: '04:30', endTime: '06:30', trafficLevel: 'Off-Peak' },
    { name: 'Morning', startTime: '06:30', endTime: '09:30', trafficLevel: 'Peak' },
    { name: 'Late Morning', startTime: '09:30', endTime: '12:00', trafficLevel: 'Moderate' },
    { name: 'Noon', startTime: '12:00', endTime: '13:00', trafficLevel: 'Off-Peak' },
    { name: 'Afternoon', startTime: '13:00', endTime: '16:30', trafficLevel: 'Moderate' },
    { name: 'Evening', startTime: '16:30', endTime: '19:30', trafficLevel: 'Peak' },
    { name: 'Night', startTime: '19:30', endTime: '21:30', trafficLevel: 'Moderate' },
    { name: 'Late Night', startTime: '21:30', endTime: '04:30', trafficLevel: 'Off-Peak', wrapsToNextDay: true },
  ];

  constructor(
    private advertisementService: AdvertisementService,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    // Only load data in browser environment
    if (isPlatformBrowser(this.platformId)) {
      this.route.queryParamMap.subscribe((params) => {
        this.selectedLoungeId = (params.get('loungeId') || '').trim();
        this.selectedLoungeName = (params.get('loungeName') || '').trim();

        if (this.selectedLoungeName) {
          this.activeTab = 'advertisements';
          // Lounge filter depends on groups->lounges mapping.
          this.loadAdvertisementGroups();
        }

        this.invalidateFilterCache();
      });

      // Load current tab first for fast initial paint.
      this.loadAdvertisements();
      this.refreshInterval = setInterval(() => {
        if (this.activeTab === 'advertisements') {
          this.loadAdvertisements();
        } else {
          this.loadAdvertisementGroups();
        }
      }, this.refreshIntervalMs);
    }
  }

  ngOnDestroy() {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }
  }

  loadAdvertisements() {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }
    
    this.advertisementService.getAllAdvertisements().subscribe({
      next: (advertisements) => {
        // Map backend data to template format
        this.advertisements = advertisements.map((ad, index) => ({
          addNo: ad.id || `Add-${String(index + 1).padStart(2, '0')}`,
          name: ad.advertisementName || 'Unnamed',
          category: ad.advertisementCategory || 'N/A',
          duration: ad.mediaDuration ? `${ad.mediaDuration}s` : '0s',
          scheduleType: ad.scheduleType || 'N/A',
          loungeGroups: ad.loungeGroupName || 'N/A',
          priority: this.capitalizeFirst(ad.priority || 'medium'),
          version: ad.version ? `v${ad.version}.0` : 'v1.0',
          status: this.capitalizeFirst(ad.status || 'active'),
          // Store original data for edit/view operations
          originalData: ad
        }));
        this.invalidateFilterCache();
      },
      error: (error) => {
        console.error('Error loading advertisements:', error);
        // Keep empty array if error
        this.advertisements = [];
        this.invalidateFilterCache();
      }
    });
  }

  capitalizeFirst(str: string): string {
    if (!str) return '';
    return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
  }

  loadAdvertisementGroups(suppressLoader: boolean = false) {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }
    
    this.advertisementService.getAllGroups(suppressLoader).subscribe({
      next: (groups) => {
        this.advertisementGroups = groups;
        // Also populate availableGroups with group names for the dropdown
        this.availableGroups = groups.map(group => group.groupName);
        this.invalidateFilterCache();

        if (this.loungesCatalog.length === 0) {
          this.loadLounges(true);
        }
      },
      error: (error) => {
        console.error('Error loading advertisement groups:', error);
        this.availableGroups = [];
      }
    });
  }

  loadLounges(suppressLoader: boolean = false) {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }
    
    this.loungeService.getAllLounges(suppressLoader).subscribe({
      next: (lounges) => {
        this.loungesCatalog = lounges;
        this.availableLounges = lounges.map(lounge => lounge.loungeName);
        if (this.selectedGroups.length > 0) {
          this.refreshAvailableSlotsForSelectedGroups();
        }
      },
      error: (error) => {
        console.error('Error loading lounges:', error);
        this.loungesCatalog = [];
        this.availableLounges = [];
      }
    });
  }

  switchTab(tab: string) {
    this.activeTab = tab;
    this.searchTerm = ''; // Reset search when switching tabs
    this.invalidateFilterCache();

    if (tab === 'groups' && this.advertisementGroups.length === 0) {
      this.loadAdvertisementGroups();
    }
  }
  
  get filteredAdvertisements() {
    const term = this.searchTerm.trim().toLowerCase();
    const key = `${term}|${this.advertisements.length}|${this.advertisementGroups.length}|${this.selectedLoungeName.toLowerCase()}`;
    if (key === this.filteredAdvertisementsKey) {
      return this.filteredAdvertisementsCache;
    }

    this.filteredAdvertisementsKey = key;
    const adsForSelectedLounge = this.filterAdvertisementsBySelectedLounge(this.advertisements);

    if (!term) {
      this.filteredAdvertisementsCache = adsForSelectedLounge;
      return this.filteredAdvertisementsCache;
    }

    this.filteredAdvertisementsCache = adsForSelectedLounge.filter(ad => 
      ad.addNo.toLowerCase().includes(term) ||
      ad.name.toLowerCase().includes(term) ||
      ad.category.toLowerCase().includes(term) ||
      ad.scheduleType.toLowerCase().includes(term) ||
      ad.loungeGroups.toLowerCase().includes(term) ||
      ad.priority.toLowerCase().includes(term) ||
      ad.status.toLowerCase().includes(term)
    );
    return this.filteredAdvertisementsCache;
  }

  clearLoungeFilter() {
    this.selectedLoungeName = '';
    this.selectedLoungeId = '';
    this.invalidateFilterCache();
  }

  private filterAdvertisementsBySelectedLounge(ads: any[]): any[] {
    if (!this.selectedLoungeName) {
      return ads;
    }

    const normalizedLounge = this.selectedLoungeName.toLowerCase();
    const matchingGroupNames = new Set(
      this.advertisementGroups
        .filter(group => this.getLoungesArray(group.lounges)
          .some(lounge => lounge.toLowerCase() === normalizedLounge))
        .map(group => group.groupName.toLowerCase())
    );

    if (matchingGroupNames.size === 0) {
      return [];
    }

    return ads.filter(ad => {
      const adGroups = (ad.loungeGroups || '')
        .split(',')
        .map((group: string) => group.trim().toLowerCase())
        .filter((group: string) => !!group && group !== 'n/a');

      return adGroups.some((group: string) => matchingGroupNames.has(group));
    });
  }

  get filteredAdvertisementGroups() {
    const term = this.searchTerm.trim().toLowerCase();
    const key = `${term}|${this.advertisementGroups.length}`;
    if (key === this.filteredGroupsKey) {
      return this.filteredGroupsCache;
    }

    this.filteredGroupsKey = key;
    if (!term) {
      this.filteredGroupsCache = this.advertisementGroups;
      return this.filteredGroupsCache;
    }

    this.filteredGroupsCache = this.advertisementGroups.filter(group => 
      group.id.toLowerCase().includes(term) ||
      group.groupName.toLowerCase().includes(term) ||
      group.lounges.toLowerCase().includes(term)
    );
    return this.filteredGroupsCache;
  }

  private invalidateFilterCache(): void {
    this.filteredAdvertisementsKey = '';
    this.filteredGroupsKey = '';
  }

  private parseDurationValue(value: string | undefined): number | undefined {
    if (!value) return undefined;
    const digits = parseInt(value.replace(/\D/g, ''), 10);
    return Number.isNaN(digits) ? undefined : digits;
  }

  get hasValidAdName(): boolean {
    return !!this.viewData.name && this.viewData.name.trim().length > 0;
  }

  get hasValidAdDuration(): boolean {
    const duration = this.parseDurationValue(this.viewData.duration);
    return duration !== undefined && duration > 0 && duration <= this.maxAllowedAdDurationSeconds;
  }

  get selectedTrafficScheduleSeconds(): number {
    if (!this.selectedTrafficLevel) {
      return this.trafficCycleSeconds['Peak'].schedule;
    }
    return this.trafficCycleSeconds[this.selectedTrafficLevel].schedule;
  }

  get selectedTrafficAdWindowSeconds(): number {
    if (!this.selectedTrafficLevel) {
      return this.trafficCycleSeconds['Peak'].ad;
    }
    return this.trafficCycleSeconds[this.selectedTrafficLevel].ad;
  }

  get maxAllowedAdDurationSeconds(): number {
    return this.selectedTrafficAdWindowSeconds;
  }

  get hasAdMedia(): boolean {
    if (this.selectedFile) {
      return true;
    }

    // Editing is valid without re-upload when media is already attached.
    if (this.isEditMode) {
      const hasFileName = !!this.viewData.fileName && this.viewData.fileName.trim().length > 0;
      const hasMediaUrl = !!this.viewData.mediaUrl && this.viewData.mediaUrl.trim().length > 0;
      return hasFileName || hasMediaUrl;
    }

    return false;
  }

  get hasSelectedLoungeGroup(): boolean {
    return this.selectedGroups.length > 0;
  }

  get hasSelectedPriority(): boolean {
    return !!this.viewData.priority && this.viewData.priority.trim().length > 0;
  }

  get canSaveAdvertisement(): boolean {
    // Base validation
    const basicValid = !this.isViewMode &&
      !this.isSaving &&
      this.hasValidAdName &&
      this.hasValidAdDuration &&
      this.hasAdMedia &&
      this.hasSelectedLoungeGroup &&
      this.hasSelectedPriority;

    if (!basicValid) {
      return false;
    }

    // Schedule-specific validation
    const normalizedScheduleType = this.scheduleType.toLowerCase();
    
    // For one-time schedules, validate date/time
    if (normalizedScheduleType === 'one-time') {
      if (!this.oneTimeScheduleDate || !this.oneTimeScheduleTime) {
        return false;
      }
    }

    // For recurring schedules with traffic level selection, validate time period
    if (normalizedScheduleType === 'recurring' && this.selectedTrafficLevel) {
      if (!this.selectedTimePeriodName) {
        return false; // Traffic level selected but no time period
      }
    }

    // For recurring schedules, validate basic fields
    if (normalizedScheduleType === 'recurring') {
      if (!this.startDate) {
        return false;
      }
      if (this.occursType === 'Weekly') {
        // At least one day must be selected
        const daysSelected = this.weeklyMonday || this.weeklyTuesday || this.weeklyWednesday ||
                            this.weeklyThursday || this.weeklyFriday || this.weeklySaturday || this.weeklySunday;
        if (!daysSelected) {
          return false;
        }
      }
    }

    return true;
  }
  
  viewData: any = {
    addNo: '',
    name: '',
    category: '',
    duration: '',
    scheduleType: '',
    loungeGroups: '',
    priority: '',
    version: '',
    status: '',
    fileName: '',
    mediaUrl: '',
    description: ''
  };

  openModal() {
    if (this.availableGroups.length === 0) {
      this.loadAdvertisementGroups(true);
    }

    this.isViewMode = false;
    this.isEditMode = false;
    this.isEmergencyCategory = false;
    this.showGroupDropdown = false;
    this.selectedGroups = [];
    this.groupSearchTerm = '';
    
    // Get current date in YYYY-MM-DD format for date inputs
    const today = new Date();
    const defaultDate = today.toISOString().split('T')[0]; // e.g., "2026-02-13"
    
    // Reset form fields
    this.scheduleType = 'Recurring';
    this.occursType = 'Daily';
    this.frequencyType = 'once';
    this.recurrenceInterval = 1;
    this.occursOnceTime = '12:00';
    this.startingTime = '12:00';
    this.endingTime = '23:59';
    this.startDate = defaultDate;
    
    // Set end date to one year from today
    const nextYear = new Date(today);
    nextYear.setFullYear(nextYear.getFullYear() + 1);
    this.endDate = nextYear.toISOString().split('T')[0];
    this.noEndDate = false;
    
    // Reset one-time schedule fields with today's date
    this.oneTimeScheduleDate = defaultDate;
    this.oneTimeScheduleTime = '12:00';
    
    // Reset weekly days
    this.weeklyMonday = false;
    this.weeklyTuesday = false;
    this.weeklyWednesday = false;
    this.weeklyThursday = false;
    this.weeklyFriday = false;
    this.weeklySaturday = false;
    this.weeklySunday = false;
    
    // Reset monthly fields
    this.monthlyDayOfMonth = 1;
    this.monthlyWeek = 'First';
    this.monthlyDay = 'Monday';
    this.monthlyType = 'dayOfMonth';
    
    // Reset frequency fields
    this.occursEveryInterval = 1;
    this.occursEveryUnit = 'Hourly';
    
    // Reset max idle loop duration
    this.maxIdleLoopDuration = 60;
    
    // Reset enabled state (default to enabled for new advertisements)
    this.isEnabled = true;
    
    // Reset file selection
    this.selectedFile = null;

    this.selectedTrafficLevel = '';
    this.selectedTimePeriodName = '';

    this.resetSlotHelperState();
    
    this.showModal = true;
    this.refreshAvailableSlotsForSelectedGroups();
  }

  viewAdvertisement(ad: any) {
    this.isViewMode = true;
    this.isEditMode = false;
    this.showModal = true;
    this.populateFormFromAdvertisement(ad);
  }

  editAdvertisement(ad: any) {
    this.isViewMode = false;
    this.isEditMode = true;
    this.showModal = true;
    this.populateFormFromAdvertisement(ad);
    this.syncTrafficPeriodFromCurrentTimes();
    this.refreshAvailableSlotsForSelectedGroups();
  }

  deleteAdvertisement(ad: any) {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    const adId = ad?.originalData?.id || ad?.addNo;
    const adName = ad?.name || ad?.originalData?.advertisementName || 'this advertisement';

    if (!adId) {
      alert('Unable to delete advertisement: missing advertisement ID.');
      return;
    }

    const confirmed = window.confirm(`Are you sure you want to delete "${adName}"? This action cannot be undone.`);
    if (!confirmed) {
      return;
    }

    this.advertisementService.deleteAdvertisement(adId).subscribe({
      next: () => {
        this.loadAdvertisements();
        alert('Advertisement deleted successfully.');
      },
      error: (error) => {
        console.error('Error deleting advertisement:', error);
        const msg = error?.error?.error || error?.message || 'Failed to delete advertisement';
        alert(msg);
      }
    });
  }

  populateFormFromAdvertisement(ad: any) {
    if (!ad || !ad.originalData) {
      console.error('No advertisement data or originalData found');
      return;
    }

    const data = ad.originalData;
    console.log('Populating form with advertisement data:', data);

    // Basic fields
    this.viewData = {
      addNo: ad.addNo || data.id,
      name: data.advertisementName || '',
      category: this.capitalizeFirst(data.advertisementCategory || 'commercial'),
      duration: data.mediaDuration ? `${data.mediaDuration}s` : '0s',
      scheduleType: this.capitalizeFirst(data.scheduleType || 'recurring'),
      loungeGroups: data.loungeGroupName || '',
      priority: this.capitalizeFirst(data.priority || 'medium'),
      version: data.version ? `v${data.version}.0` : 'v1.0',
      status: this.capitalizeFirst(data.status || 'active'),
      fileName: data.mediaUrl ? data.mediaUrl.split('/').pop() : '',
      mediaUrl: data.mediaUrl || '',
      description: data.description || ''
    };

    // Set enabled state based on status: enabled if active, disabled if paused (case-insensitive)
    this.isEnabled = (data.status || '').toLowerCase() === 'active';

    // Category and emergency check
    this.isEmergencyCategory = data.advertisementCategory?.toLowerCase() === 'emergency';

    // Schedule type (normalize to match form options)
    const scheduleType = data.scheduleType?.toLowerCase() || 'recurring';
    if (scheduleType === 'recurring') {
      this.scheduleType = 'Recurring';
    } else if (scheduleType === 'one-time') {
      this.scheduleType = 'One-time';
    } else if (scheduleType === 'on startup') {
      this.scheduleType = 'On startup';
    } else if (scheduleType === 'on idle') {
      this.scheduleType = 'On Idle';
    }

    // Handle Recurring schedule
    if (scheduleType === 'recurring' && data.frequency) {
      const frequency = data.frequency.toLowerCase();
      
      // Set occurs type (Daily/Weekly/Monthly)
      if (frequency === 'daily') {
        this.occursType = 'Daily';
      } else if (frequency === 'weekly') {
        this.occursType = 'Weekly';
      } else if (frequency === 'monthly') {
        this.occursType = 'Monthly';
      }

      // Recurrence interval
      this.recurrenceInterval = data.recurrenceInterval || 1;

      // Start and end dates
      if (data.startDate) {
        this.startDate = this.formatDateForInput(data.startDate);
      }
      if (data.endDate) {
        this.endDate = this.formatDateForInput(data.endDate);
        this.noEndDate = false;
      } else {
        this.noEndDate = true;
      }

      // Start and end times
      if (data.startTime) {
        this.startingTime = data.startTime.substring(0, 5); // Extract HH:MM
      }
      if (data.endTime) {
        this.endingTime = data.endTime.substring(0, 5);
      }

      // Determine frequency type (once vs every)
      if (data.occursEveryInterval && data.occursEveryInterval > 0) {
        // "Occurs every" option
        this.frequencyType = 'every';
        this.occursEveryInterval = data.occursEveryInterval;
        
        // Infer unit from the interval value (this is approximate)
        // You might want to store the unit separately in the database
        this.occursEveryUnit = 'Hourly'; // Default
      } else if (data.occursOnceAt) {
        // "Occurs once at" option
        this.frequencyType = 'once';
        const time = new Date(data.occursOnceAt);
        this.occursOnceTime = this.formatTimeForInput(time);
      } else {
        this.frequencyType = 'once';
      }

      // Weekly days (parse comma-separated string)
      if (frequency === 'weekly' && data.weeklyDays) {
        const days = data.weeklyDays.toLowerCase().split(',').map((d: string) => d.trim());
        this.weeklyMonday = days.includes('monday');
        this.weeklyTuesday = days.includes('tuesday');
        this.weeklyWednesday = days.includes('wednesday');
        this.weeklyThursday = days.includes('thursday');
        this.weeklyFriday = days.includes('friday');
        this.weeklySaturday = days.includes('saturday');
        this.weeklySunday = days.includes('sunday');
      }

      // Monthly fields
      if (frequency === 'monthly') {
        if (data.monthlyDayOfMonth) {
          this.monthlyType = 'dayOfMonth';
          this.monthlyDayOfMonth = data.monthlyDayOfMonth;
        } else if (data.monthlyWeek && data.monthlyDay) {
          this.monthlyType = 'weekday';
          this.monthlyWeek = this.capitalizeFirst(data.monthlyWeek);
          this.monthlyDay = this.capitalizeFirst(data.monthlyDay);
        }
      }
    }

    // Handle One-time schedule
    if (scheduleType === 'one-time' && data.occursOnceAt) {
      const dateTime = new Date(data.occursOnceAt);
      this.oneTimeScheduleDate = this.formatDateForInput(dateTime);
      this.oneTimeScheduleTime = this.formatTimeForInput(dateTime);
    }

    // Handle On Idle
    if (scheduleType === 'on idle' && data.maxIdleLoopDuration) {
      this.maxIdleLoopDuration = data.maxIdleLoopDuration;
    }

    // Lounge groups
    if (data.loungeGroupName) {
      this.selectedGroups = [data.loungeGroupName];
    }

    // Update description
    this.updateDescription();
  }

  formatDateForInput(dateValue: any): string {
    if (!dateValue) return '';
    const date = new Date(dateValue);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  formatTimeForInput(timeValue: any): string {
    if (!timeValue) return '12:00';
    const date = new Date(timeValue);
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${hours}:${minutes}`;
  }

  // Convert local date and time to UTC ISO string
  convertLocalToUTC(dateStr: string, timeStr: string): string {
    // Create a date object in local time
    const localDateTime = new Date(`${dateStr}T${timeStr}:00`);
    // Convert to ISO string (this automatically converts to UTC)
    return localDateTime.toISOString();
  }

  closeModal() {
    this.showModal = false;
    this.isViewMode = false;
    this.isEditMode = false;
    this.selectedFile = null;
    this.scheduleType = 'Recurring';
    this.occursType = 'Daily';
    this.frequencyType = 'once';
    this.recurrenceInterval = 1;
    
    // Reset one-time schedule fields
    this.oneTimeScheduleDate = '2026-01-16';
    this.oneTimeScheduleTime = '00:00';
    
    // Reset weekly days
    this.weeklyMonday = false;
    this.weeklyTuesday = false;
    this.weeklyWednesday = false;
    this.weeklyThursday = false;
    this.weeklyFriday = false;
    this.weeklySaturday = false;
    this.weeklySunday = false;
    
    // Reset monthly fields
    this.monthlyDayOfMonth = 1;
    this.monthlyWeek = 'First';
    this.monthlyDay = 'Monday';
    this.monthlyType = 'dayOfMonth';
    
    // Reset frequency fields
    this.occursEveryInterval = 1;
    this.occursEveryUnit = 'Hourly';
    this.recurrenceInterval = 1;
    
    // Reset max idle loop duration
    this.maxIdleLoopDuration = 60;
    
    this.viewData = {
      addNo: '',
      name: '',
      category: '',
      duration: '',
      scheduleType: '',
      loungeGroups: '',
      priority: '',
      version: '',
      status: '',
      fileName: '',
      mediaUrl: '',
      description: ''
    };

    this.selectedTrafficLevel = '';
    this.selectedTimePeriodName = '';

    this.resetSlotHelperState();
  }

  onFileSelected(event: Event) {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      const selected = input.files[0];
      this.selectedFile = selected;
      this.viewData.fileName = selected.name;
      console.log('Selected file:', this.selectedFile.name);
    }
  }

  onScheduleTypeChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    this.scheduleType = select.value;
    this.syncTrafficPeriodFromCurrentTimes();
    this.updateDescription();
  }

  onCategoryChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    this.viewData.category = select.value;
    this.isEmergencyCategory = select.value === 'Emergency';
    
    if (this.isEmergencyCategory) {
      // Auto-set emergency defaults
      this.viewData.priority = 'High';
      this.scheduleType = 'On startup';
      this.updateDescription();
    }
  }

  onOccursTypeChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    this.occursType = select.value;
    this.updateDescription();
  }

  onTrafficLevelChange(): void {
    if (!this.selectedTrafficLevel) {
      this.selectedTimePeriodName = '';
      this.refreshAvailableSlotsForSelectedGroups();
      this.updateDescription();
      return;
    }

    const options = this.filteredTrafficTimePeriods;
    if (!options.some((item) => item.name === this.selectedTimePeriodName)) {
      this.selectedTimePeriodName = options.length ? options[0].name : '';
    }
    this.applySelectedTrafficTimePeriod(false);
  }

  onTimePeriodChange(): void {
    this.applySelectedTrafficTimePeriod(false);
  }

  get filteredTrafficTimePeriods(): TrafficTimePeriod[] {
    if (!this.selectedTrafficLevel) {
      return this.trafficTimePeriods;
    }
    return this.trafficTimePeriods.filter((item) => item.trafficLevel === this.selectedTrafficLevel);
  }

  get selectedTrafficTimePeriod(): TrafficTimePeriod | null {
    if (!this.selectedTimePeriodName) {
      return null;
    }
    return this.trafficTimePeriods.find((item) => item.name === this.selectedTimePeriodName) || null;
  }

  formatTrafficPeriod(period: TrafficTimePeriod): string {
    const endSuffix = period.wrapsToNextDay ? ' (next day)' : '';
    return `${period.startTime} - ${period.endTime}${endSuffix}`;
  }

  updateDescription() {
    let description = '';
    
    // Emergency category override
    if (this.isEmergencyCategory) {
      description = '⚠️ EMERGENCY BROADCAST: This message will be displayed immediately on all selected lounge screens with highest priority. It will trigger automatically when the lounge display software boots up.';
      this.viewData.description = description;
      return;
    }
    
    if (this.scheduleType === 'Recurring') {
      // Format time to 12-hour format with AM/PM
      const formatTime = (time: string) => {
        const [hours, minutes] = time.split(':');
        const hour = parseInt(hours);
        const ampm = hour >= 12 ? 'PM' : 'AM';
        const hour12 = hour % 12 || 12;
        return `${hour12}:${minutes}:00 ${ampm}`;
      };
      
      if (this.occursType === 'Daily') {
        if (this.frequencyType === 'once') {
          description = `Occurs every day at ${formatTime(this.occursOnceTime)}. `;
        } else if (this.frequencyType === 'every') {
          // Format the unit to lowercase singular/plural
          let unitText = '';
          const unit = this.occursEveryUnit.toLowerCase();
          const interval = this.occursEveryInterval;
          
          if (unit === 'hourly') {
            unitText = interval === 1 ? 'hour' : 'hours';
          } else if (unit === 'minutes') {
            unitText = interval === 1 ? 'minute' : 'minutes';
          } else if (unit === 'secs') {
            unitText = interval === 1 ? 'second' : 'seconds';
          }
          
          description = `Occurs every ${interval} ${unitText}, starting at ${formatTime(this.startingTime)} and ending at ${formatTime(this.endingTime)}. `;
        }
      } else if (this.occursType === 'Weekly') {
        if (this.frequencyType === 'once') {
          description = `Occurs on selected weekdays at ${formatTime(this.occursOnceTime)}. `;
        } else if (this.frequencyType === 'every') {
          let unitText = '';
          const unit = this.occursEveryUnit.toLowerCase();
          const interval = this.occursEveryInterval;
          
          if (unit === 'hourly') {
            unitText = interval === 1 ? 'hour' : 'hours';
          } else if (unit === 'minutes') {
            unitText = interval === 1 ? 'minute' : 'minutes';
          } else if (unit === 'secs') {
            unitText = interval === 1 ? 'second' : 'seconds';
          }
          
          description = `Occurs on selected weekdays every ${interval} ${unitText}, starting at ${formatTime(this.startingTime)} and ending at ${formatTime(this.endingTime)}. `;
        }
      } else if (this.occursType === 'Monthly') {
        if (this.frequencyType === 'once') {
          description = `Occurs on the specified day of each month at ${formatTime(this.occursOnceTime)}. `;
        } else if (this.frequencyType === 'every') {
          let unitText = '';
          const unit = this.occursEveryUnit.toLowerCase();
          const interval = this.occursEveryInterval;
          
          if (unit === 'hourly') {
            unitText = interval === 1 ? 'hour' : 'hours';
          } else if (unit === 'minutes') {
            unitText = interval === 1 ? 'minute' : 'minutes';
          } else if (unit === 'secs') {
            unitText = interval === 1 ? 'second' : 'seconds';
          }
          
          description = `Occurs on the specified day of each month every ${interval} ${unitText}, starting at ${formatTime(this.startingTime)} and ending at ${formatTime(this.endingTime)}. `;
        }
      }
      
      // Format date to MM/DD/YYYY
      const formatDate = (dateStr: string) => {
        const date = new Date(dateStr);
        return `${date.getMonth() + 1}/${date.getDate()}/${date.getFullYear()}`;
      };
      
      description += `Schedule will be used starting on ${formatDate(this.startDate)}`;
      
      if (!this.noEndDate) {
        description += ` and ending on ${formatDate(this.endDate)}`;
      }
      
    } else if (this.scheduleType === 'One-time') {
      description = 'Runs once at the specified date and time.';
    } else if (this.scheduleType === 'On startup') {
      description = 'Triggers automatically when lounge display software boots up.';
    } else if (this.scheduleType === 'On Idle') {
      description = 'Plays as filler when no other ads are scheduled to run.';
    }
    
    const selectedPeriod = this.selectedTrafficTimePeriod;
    if (selectedPeriod) {
      const trafficNote = ` Traffic slot: ${selectedPeriod.name} (${this.formatTrafficPeriod(selectedPeriod)}), ${selectedPeriod.trafficLevel}.`;
      this.viewData.description = `${description}${trafficNote}`.trim();
      return;
    }

    this.viewData.description = description;
  }

  saveAdvertisement() {
    if (!this.canSaveAdvertisement) {
      const missingFields: string[] = [];
      if (!this.hasAdMedia) missingFields.push('ad media');
      if (!this.hasValidAdName) missingFields.push('ad name');
      if (!this.hasValidAdDuration) missingFields.push('duration');
      if (!this.hasSelectedLoungeGroup) missingFields.push('lounge group');
      if (!this.hasSelectedPriority) missingFields.push('priority level');

      alert(`Please provide required fields before saving: ${missingFields.join(', ')}.`);
      return;
    }

    // Update loungeGroups from selectedGroups
    this.viewData.loungeGroups = this.selectedGroups.join(',');

    // Validate recurring schedule selections
    const normalizedScheduleType = this.scheduleType.toLowerCase();
    if (normalizedScheduleType === 'recurring') {
      if (this.selectedTrafficLevel && !this.selectedTimePeriodName) {
        alert('You selected a traffic level. Please also select a time period for this traffic level.');
        return;
      }
      if (this.occursType === 'Weekly') {
        const daysSelected = this.weeklyMonday || this.weeklyTuesday || this.weeklyWednesday ||
                            this.weeklyThursday || this.weeklyFriday || this.weeklySaturday || this.weeklySunday;
        if (!daysSelected) {
          alert('Please select at least one day of the week for weekly schedule.');
          return;
        }
      }
    }

    if (this.availableExactSlotsForSelectedDuration.length > 0 && this.selectedExactSlotStartIsos.length === 0) {
      alert('Select at least one exact play slot before saving this advertisement.');
      return;
    }

    if (this.exactTimeSlotsForSelectedDuration.length > 0 && this.availableExactSlotsForSelectedDuration.length === 0) {
      alert('All exact time slots for the selected period are already booked. Please change the traffic period or date.');
      return;
    }

    if (this.effectiveAvailableSeconds > 0) {
      const selectedDuration = this.parseDurationValue(this.viewData.duration) || 0;
      if (selectedDuration > this.effectiveAvailableSeconds) {
        alert(
          `Selected duration (${selectedDuration}s) exceeds the allowed ad window (${this.effectiveAvailableSeconds}s) for the selected traffic period.`
        );
        return;
      }
    }

    if (this.selectedTrafficLevel && !this.selectedTimePeriodName) {
      alert('Please select a time period for the chosen traffic level.');
      return;
    }

    console.log('=== Checking for conflicts ===');
    console.log('Selected Groups:', this.selectedGroups);
    console.log('Schedule Time:', this.getScheduleTimeSlot());
    console.log('Schedule Type:', this.scheduleType);

    // In exact-slot mode, conflicts are shown directly by color on each time slot.
    // Save directly without the submit-time conflict dialog.
    if (this.exactTimeSlotsForSelectedDuration.length > 0) {
      this.proceedWithSave();
      return;
    }

    // Keep backend conflict checks for non-exact scheduling modes.
    this.checkScheduleConflictBackend();
  }

  checkScheduleConflictBackend() {
    if (!this.selectedGroups || this.selectedGroups.length === 0) {
      this.proceedWithSave();
      return;
    }

    // Reset per-run conflict state
    this.excludedLoungesByGroup = {};
    this.currentConflictGroupIndex = -1;

    this.checkConflictForGroup(0);
  }

  checkConflictForGroup(index: number) {
    if (index >= this.selectedGroups.length) {
      // All groups passed conflict check
      this.proceedWithSave();
      return;
    }

    const groupName = this.selectedGroups[index];
    const payload = this.buildConflictPayload(groupName);

    this.advertisementService.checkAdvertisementConflicts(payload).subscribe({
      next: (res) => {
        const excludedLounges = this.excludedLoungesByGroup[groupName] || new Set<string>();

        const sanitize = (list: string[] | undefined) =>
          (list || [])
            .map((l: string) => (l || '').trim())
            .filter((l: string) => l !== '' && l !== '[]' && !excludedLounges.has(l));

        const activeAffectedLounges = sanitize(res?.affectedLounges);

        // Fallback: use group's lounges minus excluded ones if backend returned none
        const groupRecord = this.advertisementGroups.find(g => g.groupName === groupName);
        const groupLounges = sanitize(groupRecord ? this.getLoungesArray(groupRecord.lounges) : []);

        const candidateLounges = activeAffectedLounges.length > 0 ? activeAffectedLounges : groupLounges;

        const sanitizedResponse = sanitize(res?.affectedLounges);
        const firstCandidate = (candidateLounges[0] || '').trim();
        const firstResponse = (sanitizedResponse[0] || '').trim();
        let conflictingLounge = firstCandidate || firstResponse || groupName || 'selected lounge';

        const displayLounges = (candidateLounges.length > 0 ? candidateLounges : sanitizedResponse);

        console.log('Conflict detected:', {
          backendAffectedLounges: res?.affectedLounges,
          sanitizedResponse,
          candidateLounges,
          conflictingLounge,
          groupName
        });

        if (res?.hasConflict && (displayLounges.length > 0 || groupName)) {
          const formattedSlot = this.formatConflictTimeSlot(res?.conflictTimeSlot || '');
          const loungeName = conflictingLounge || groupName || 'the selected lounge';
          
          // Check if we have an actual lounge name (not falling back to group name)
          const isActualLounge = conflictingLounge && conflictingLounge !== groupName;
          const loungePrefix = isActualLounge ? 'Lounge ' : '';
          const loungeSuffix = isActualLounge ? ' lounge' : '';

          this.conflictData = {
            message: `${loungePrefix}${loungeName} already has an advertisement scheduled from ${formattedSlot} time slot. This advertisement will overlap.`,
            removeOption: `Remove ${loungeName}${loungeSuffix} from ${groupName}`,
            conflictingLounge,
            conflictingGroup: groupName,
            conflictingGroups: res?.conflictingAds || [],
            affectedLounges: displayLounges.length > 0 ? displayLounges : [conflictingLounge],
            timeSlot: formattedSlot
          };
          this.selectedConflictOption = '';
          this.currentConflictGroupIndex = index;
          this.showConflictModal = true;
        } else {
          this.checkConflictForGroup(index + 1);
        }
      },
      error: (err) => {
        console.error('Error checking conflicts:', err);
        alert('Failed to check schedule conflicts. Proceeding to save.');
        this.proceedWithSave();
      }
    });
  }

  buildConflictPayload(groupName: string) {
    const normalizedScheduleType = this.scheduleType.toLowerCase();

    const payload: any = {
      loungeGroupName: groupName,
      scheduleType: normalizedScheduleType,
      frequency: normalizedScheduleType === 'recurring' ? this.occursType.toLowerCase() : undefined,
      occursOnceAt: undefined as string | undefined,
      startTime: undefined as string | undefined,
      endTime: undefined as string | undefined,
      startDate: normalizedScheduleType === 'recurring' ? this.startDate : undefined,
      endDate: normalizedScheduleType === 'recurring' && !this.noEndDate ? this.endDate : undefined,
      weeklyDays: normalizedScheduleType === 'recurring' && this.occursType === 'Weekly' ? this.getWeeklyDaysString() : undefined,
      monthlyDayOfMonth: normalizedScheduleType === 'recurring' && this.occursType === 'Monthly' && this.monthlyType === 'dayOfMonth' ? this.monthlyDayOfMonth : undefined,
      recurrenceInterval: normalizedScheduleType === 'recurring' ? this.recurrenceInterval : undefined,
      excludeAdvertisementId: this.isEditMode && this.viewData.addNo ? this.viewData.addNo : undefined
    };

    if (normalizedScheduleType === 'one-time') {
      payload.occursOnceAt = this.convertLocalToUTC(this.oneTimeScheduleDate, this.oneTimeScheduleTime);
    } else if (normalizedScheduleType === 'recurring' && this.frequencyType === 'once' && this.occursOnceTime && this.startDate) {
      payload.occursOnceAt = this.convertLocalToUTC(this.startDate, this.occursOnceTime);
    }

    if (normalizedScheduleType === 'recurring' && this.frequencyType === 'every') {
      payload.startTime = this.startingTime;
      payload.endTime = this.endingTime;
    }

    return payload;
  }

  getWeeklyDaysString(): string | undefined {
    if (this.occursType !== 'Weekly') return undefined;
    const days: string[] = [];
    if (this.weeklyMonday) days.push('monday');
    if (this.weeklyTuesday) days.push('tuesday');
    if (this.weeklyWednesday) days.push('wednesday');
    if (this.weeklyThursday) days.push('thursday');
    if (this.weeklyFriday) days.push('friday');
    if (this.weeklySaturday) days.push('saturday');
    if (this.weeklySunday) days.push('sunday');
    return days.length > 0 ? days.join(',') : undefined;
  }

  // Group Modal Methods
  openGroupModal() {
    if (this.availableLounges.length === 0) {
      this.loadLounges(true);
    }

    this.showGroupModal = true;
    this.showLoungeDropdown = true;
    this.isViewGroupMode = false;
    this.isEditGroupMode = false;
    this.selectedLounges = [];
    this.groupData = { id: '', groupName: '', lounges: [] };
    this.loungeSearchTerm = '';
  }

  viewGroup(group: AdvertisementGroup) {
    this.showGroupModal = true;
    this.showLoungeDropdown = false;
    this.isViewGroupMode = true;
    this.isEditGroupMode = false;
    this.groupData = { 
      id: group.id,
      groupName: group.groupName, 
      lounges: [] 
    };
    // Parse the lounges JSON string
    try {
      const loungesArray = JSON.parse(group.lounges);
      this.selectedLounges = Array.isArray(loungesArray) ? loungesArray : [];
    } catch {
      this.selectedLounges = [];
    }
    this.loungeSearchTerm = '';
    
    // Get advertisements for this group
    this.groupAdvertisements = this.advertisements
      .filter(ad => ad.loungeGroups.includes(group.groupName))
      .map(ad => ad.name);
  }

  editGroup(group: AdvertisementGroup) {
    this.showGroupModal = true;
    this.showLoungeDropdown = true;
    this.isViewGroupMode = false;
    this.isEditGroupMode = true;
    this.groupData = { 
      id: group.id,
      groupName: group.groupName, 
      lounges: [] 
    };
    // Parse the lounges JSON string
    try {
      const loungesArray = JSON.parse(group.lounges);
      this.selectedLounges = Array.isArray(loungesArray) ? loungesArray : [];
    } catch {
      this.selectedLounges = [];
    }
    this.loungeSearchTerm = '';
  }

  closeGroupModal() {
    this.showGroupModal = false;
    this.showLoungeDropdown = false;
    this.isViewGroupMode = false;
    this.isEditGroupMode = false;
    this.selectedLounges = [];
    this.groupData = { id: '', groupName: '', lounges: [] };
  }

  toggleLoungeDropdown() {
    if (this.availableLounges.length === 0) {
      this.loadLounges(true);
    }
    this.showLoungeDropdown = !this.showLoungeDropdown;
  }

  trackByAdvertisement = (_: number, ad: any): string => ad.addNo;

  trackByGroup = (_: number, group: AdvertisementGroup): string => group.id;

  trackByString = (_: number, value: string): string => value;

  trackByDurationSlot = (_: number, slot: { durationSeconds: number }): number => slot.durationSeconds;

  trackByExactSlot = (_: number, slot: ExactPlayableSlot): string => slot.startIso;

  toggleLounge(lounge: string) {
    const index = this.selectedLounges.indexOf(lounge);
    if (index > -1) {
      this.selectedLounges.splice(index, 1);
    } else {
      this.selectedLounges.push(lounge);
    }
  }

  removeLounge(lounge: string) {
    const index = this.selectedLounges.indexOf(lounge);
    if (index > -1) {
      this.selectedLounges.splice(index, 1);
    }
  }

  isLoungeSelected(lounge: string): boolean {
    return this.selectedLounges.includes(lounge);
  }

  get sortedLounges(): string[] {
    const selected = this.availableLounges.filter(lounge => this.selectedLounges.includes(lounge));
    const unselected = this.availableLounges.filter(lounge => !this.selectedLounges.includes(lounge));
    return [...selected, ...unselected];
  }

  get filteredSortedLounges(): string[] {
    if (!this.loungeSearchTerm.trim()) {
      return this.sortedLounges;
    }
    const searchTerm = this.loungeSearchTerm.toLowerCase();
    return this.sortedLounges.filter(lounge => lounge.toLowerCase().includes(searchTerm));
  }

  isAllLoungesSelected(): boolean {
    return this.selectedLounges.length === this.availableLounges.length;
  }

  toggleSelectAll() {
    if (this.isAllLoungesSelected()) {
      this.selectedLounges = [];
    } else {
      this.selectedLounges = [...this.availableLounges];
    }
  }

  clearAllLounges() {
    this.selectedLounges = [];
  }

  saveGroup() {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    console.log('saveGroup called');
    console.log('isEditGroupMode:', this.isEditGroupMode);
    console.log('groupData:', this.groupData);
    console.log('selectedLounges:', this.selectedLounges);

    if (this.isEditGroupMode) {
      // Update existing group
      // Validate group name
      if (!this.groupData.groupName || this.groupData.groupName.trim() === '') {
        alert('Please enter a group name.');
        return;
      }

      console.log('Updating group with ID:', this.groupData.id);
      this.advertisementService.updateGroup(this.groupData.id, {
        groupName: this.groupData.groupName.trim(),
        lounges: this.selectedLounges
      }).subscribe({
        next: (response) => {
          console.log('Update successful:', response);
          alert('Successfully updated group details');
          this.activeTab = 'groups';
          this.loadAdvertisementGroups();
          this.closeGroupModal();
        },
        error: (error) => {
          console.error('Error updating group:', error);
          console.error('Error details:', {
            status: error.status,
            statusText: error.statusText,
            message: error.message,
            error: error.error
          });
          
          // Parse error message to provide user-friendly feedback
          let errorMsg = 'Failed to update group. Please try again.';
          if (error.error?.error) {
            const errStr = error.error.error.toLowerCase();
            if (errStr.includes('duplicate') || errStr.includes('unique constraint')) {
              errorMsg = `A group with the name "${this.groupData.groupName}" already exists. Please use a different name.`;
            } else {
              errorMsg = error.error.error;
            }
          } else if (error.message) {
            errorMsg = error.message;
          }
          
          alert(errorMsg);
        }
      });
    } else {
      // Create new group
      console.log('Validating groupName:', this.groupData.groupName);
      console.log('groupName type:', typeof this.groupData.groupName);
      console.log('groupName length:', this.groupData.groupName?.length);
      console.log('groupName after trim:', this.groupData.groupName?.trim());
      
      // Validate group name
      const trimmedName = (this.groupData.groupName || '').trim();
      if (!trimmedName) {
        alert('Please enter a group name.');
        return;
      }

      const payload = {
        groupName: trimmedName,
        lounges: this.selectedLounges
      };
      console.log('Creating new group with payload:', payload);
      
      this.advertisementService.createGroup(payload).subscribe({
        next: (response) => {
          console.log('Create successful:', response);
          alert('Successfully created the group');
          this.activeTab = 'groups';
          this.loadAdvertisementGroups();
          this.closeGroupModal();
        },
        error: (error) => {
          console.error('Error creating group:', error);
          console.error('Error details:', {
            status: error.status,
            statusText: error.statusText,
            message: error.message,
            error: error.error,
            url: error.url
          });
          
          // Parse error message to provide user-friendly feedback
          let errorMsg = 'Failed to create group. Please try again.';
          if (error.error?.error) {
            const errStr = error.error.error.toLowerCase();
            if (errStr.includes('duplicate') || errStr.includes('unique constraint')) {
              errorMsg = `A group with the name "${this.groupData.groupName}" already exists. Please use a different name.`;
            } else {
              errorMsg = error.error.error;
            }
          } else if (error.message) {
            errorMsg = error.message;
          }
          
          alert(errorMsg);
        }
      });
    }
  }

  getLoungesDisplay(loungesJson: string): string {
    try {
      const loungesArray = JSON.parse(loungesJson);
      if (Array.isArray(loungesArray)) {
        return loungesArray.join(', ');
      }
    } catch {
      // fall through to comma-split fallback
    }

    // Fallback: handle comma-separated string
    if (loungesJson) {
      const parts = loungesJson.split(',').map(l => l.trim()).filter(Boolean);
      return parts.join(', ');
    }
    return '';
  }

  getLoungesArray(loungesJson: string): string[] {
    try {
      const loungesArray = JSON.parse(loungesJson);
      if (Array.isArray(loungesArray)) {
        return loungesArray;
      }
    } catch {
      // fall through to comma-split fallback
    }

    if (!loungesJson) return [];
    return loungesJson.split(',').map(l => l.trim()).filter(Boolean);
  }

  // Advertisement Group Selection Methods
  toggleGroup(group: string) {
    const index = this.selectedGroups.indexOf(group);
    if (index > -1) {
      this.selectedGroups.splice(index, 1);
    } else {
      this.selectedGroups.push(group);
    }
    this.refreshAvailableSlotsForSelectedGroups();
  }

  removeGroup(group: string) {
    const index = this.selectedGroups.indexOf(group);
    if (index > -1) {
      this.selectedGroups.splice(index, 1);
    }
    this.refreshAvailableSlotsForSelectedGroups();
  }

  isGroupSelected(group: string): boolean {
    return this.selectedGroups.includes(group);
  }

  get sortedGroups(): string[] {
    const selected = this.availableGroups.filter(group => this.selectedGroups.includes(group));
    const unselected = this.availableGroups.filter(group => !this.selectedGroups.includes(group));
    return [...selected, ...unselected];
  }

  get filteredSortedGroups(): string[] {
    if (!this.groupSearchTerm.trim()) {
      return this.sortedGroups;
    }
    const searchTerm = this.groupSearchTerm.toLowerCase();
    return this.sortedGroups.filter(group => group.toLowerCase().includes(searchTerm));
  }

  isAllGroupsSelected(): boolean {
    return this.selectedGroups.length === this.availableGroups.length;
  }

  toggleSelectAllGroups() {
    if (this.isAllGroupsSelected()) {
      this.selectedGroups = [];
    } else {
      this.selectedGroups = [...this.availableGroups];
    }

    this.refreshAvailableSlotsForSelectedGroups();
  }

  clearAllGroups() {
    this.selectedGroups = [];
    this.refreshAvailableSlotsForSelectedGroups();
  }

  onDurationInputChange(): void {
    this.selectedExactSlotStartIsos = [];
  }

  pickDurationFromAvailableSlot(seconds: number): void {
    const normalized = Math.max(1, Math.min(this.commonAvailableSeconds || this.maxAllowedAdDurationSeconds, Math.floor(seconds || 0)));
    this.viewData.duration = `${normalized}s`;
    this.selectedExactSlotStartIsos = [];
  }

  pickExactTimeSlot(startIso: string): void {
    if (this.isExactSlotBooked(startIso)) {
      return;
    }

    if (this.selectedExactSlotStartIsos.includes(startIso)) {
      this.selectedExactSlotStartIsos = this.selectedExactSlotStartIsos.filter((item) => item !== startIso);
      return;
    }
    this.selectedExactSlotStartIsos = [...this.selectedExactSlotStartIsos, startIso];
  }

  get selectedDurationSeconds(): number {
    return this.parseDurationValue(this.viewData.duration) || 0;
  }

  get selectedDurationEndSecond(): number {
    const allowed = Math.max(0, Math.min(this.selectedDurationSeconds, this.effectiveAvailableSeconds));
    return this.effectiveAdWindowStartSecond + allowed;
  }

  get isDurationWithinAdWindow(): boolean {
    return this.selectedDurationSeconds > 0 && this.selectedDurationSeconds <= this.maxAllowedAdDurationSeconds;
  }

  get exactTimeSlotsForSelectedDuration(): ExactPlayableSlot[] {
    if (!this.isDurationWithinAdWindow || this.effectiveAvailableSeconds <= 0) {
      return [];
    }

    if (this.selectedDurationSeconds > this.effectiveAvailableSeconds) {
      return [];
    }

    const range = this.getSelectedTimePeriodRange();
    const slots: ExactPlayableSlot[] = [];

    for (let cycleStartMs = range.start.getTime(); cycleStartMs < range.end.getTime(); cycleStartMs += 30000) {
      const adWindowStart = new Date(cycleStartMs + this.selectedTrafficScheduleSeconds * 1000);
      const adWindowEnd = new Date(cycleStartMs + this.cycleDurationSeconds * 1000);

      if (adWindowEnd.getTime() > range.end.getTime()) {
        continue;
      }

      slots.push({
        startIso: adWindowStart.toISOString(),
        endIso: adWindowEnd.toISOString(),
        startLabel: this.formatDateTime(adWindowStart),
        endLabel: this.formatDateTime(adWindowEnd),
      });
    }

    return slots;
  }

  get selectedExactSlots(): ExactPlayableSlot[] {
    if (!this.selectedExactSlotStartIsos.length) {
      return [];
    }
    const selectedSet = new Set(this.selectedExactSlotStartIsos);
    return this.exactTimeSlotsForSelectedDuration.filter((slot) => selectedSet.has(slot.startIso));
  }

  get bookedExactSlotLabelsSet(): Set<string> {
    const booked = new Set<string>();
    const selectedLounges = this.getLoungeSetFromGroups(this.selectedGroups);
    if (!selectedLounges.size) {
      return booked;
    }

    for (const ad of this.advertisements) {
      const adId = ad?.originalData?.id || ad?.addNo;
      if (this.isEditMode && this.viewData?.addNo && adId === this.viewData.addNo) {
        continue;
      }

      const adGroups = String(ad?.loungeGroups || '')
        .split(',')
        .map((g: string) => g.trim())
        .filter((g: string) => !!g && g.toLowerCase() !== 'n/a');

      if (!adGroups.length) {
        continue;
      }

      const adLounges = this.getLoungeSetFromGroups(adGroups);
      if (!this.hasLoungeOverlap(selectedLounges, adLounges)) {
        continue;
      }

      for (const label of this.extractBookedSlotLabelsFromAd(ad)) {
        booked.add(label);
      }
    }

    return booked;
  }

  isExactSlotBooked(startIso: string): boolean {
    const slot = this.exactTimeSlotsForSelectedDuration.find((item) => item.startIso === startIso);
    if (!slot) {
      return false;
    }
    return this.bookedExactSlotLabelsSet.has(slot.startLabel);
  }

  get bookedExactSlotsForSelectedDuration(): ExactPlayableSlot[] {
    const bookedSet = this.bookedExactSlotLabelsSet;
    return this.exactTimeSlotsForSelectedDuration.filter((slot) => bookedSet.has(slot.startLabel));
  }

  get availableExactSlotsForSelectedDuration(): ExactPlayableSlot[] {
    const bookedSet = this.bookedExactSlotLabelsSet;
    return this.exactTimeSlotsForSelectedDuration.filter((slot) => !bookedSet.has(slot.startLabel));
  }

  get slotBaseDateLabel(): string {
    const base = this.getCycleBaseDateTime();
    const year = base.getFullYear();
    const month = String(base.getMonth() + 1).padStart(2, '0');
    const day = String(base.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  get slotBusScheduleWindowLabel(): string {
    return `${this.formatClockFromOffset(0)} - ${this.formatClockFromOffset(this.selectedTrafficScheduleSeconds)}`;
  }

  get slotAdWindowLabel(): string {
    return `${this.formatClockFromOffset(this.effectiveAdWindowStartSecond)} - ${this.formatClockFromOffset(this.cycleDurationSeconds)}`;
  }

  get slotAvailableWindowLabel(): string {
    return this.slotAdWindowLabel;
  }

  get selectedExactSlotLabel(): string {
    const selected = this.selectedExactSlots;
    if (!selected.length) {
      return '';
    }
    return selected.map((slot) => `${slot.startLabel} - ${slot.endLabel}`).join(', ');
  }

  get selectedSlotLabelsForView(): string[] {
    return this.selectedExactSlots.map((slot) => `${slot.startLabel} - ${slot.endLabel}`);
  }

  formatExactSlot(slot: ExactPlayableSlot): string {
    return `${slot.startLabel} - ${slot.endLabel}`;
  }

  get timePeriodSlotCount(): number {
    return this.exactTimeSlotsForSelectedDuration.length;
  }

  get bookedTimePeriodSlotCount(): number {
    return this.bookedExactSlotsForSelectedDuration.length;
  }

  get bookedExactSlotLegendRows(): Array<{ slot: ExactPlayableSlot; advertisementIds: string[] }> {
    const rows: Array<{ slot: ExactPlayableSlot; advertisementIds: string[] }> = [];
    const slotAdMap = new Map<string, Set<string>>();

    const selectedLounges = this.getLoungeSetFromGroups(this.selectedGroups);
    if (!selectedLounges.size) {
      return rows;
    }

    for (const ad of this.advertisements) {
      const adId = ad?.originalData?.id || ad?.addNo;
      if (this.isEditMode && this.viewData?.addNo && adId === this.viewData.addNo) {
        continue;
      }

      const adGroups = String(ad?.loungeGroups || '')
        .split(',')
        .map((g: string) => g.trim())
        .filter((g: string) => !!g && g.toLowerCase() !== 'n/a');

      if (!adGroups.length) {
        continue;
      }

      const adLounges = this.getLoungeSetFromGroups(adGroups);
      if (!this.hasLoungeOverlap(selectedLounges, adLounges)) {
        continue;
      }

      const bookedLabels = this.extractBookedSlotLabelsFromAd(ad);
      for (const slot of this.bookedExactSlotsForSelectedDuration) {
        if (bookedLabels.includes(slot.startLabel)) {
          const key = slot.startLabel;
          if (!slotAdMap.has(key)) {
            slotAdMap.set(key, new Set<string>());
          }
          slotAdMap.get(key)?.add(String(adId));
        }
      }
    }

    for (const slot of this.bookedExactSlotsForSelectedDuration) {
      const adIds = slotAdMap.get(slot.startLabel);
      if (adIds) {
        rows.push({
          slot,
          advertisementIds: Array.from(adIds),
        });
      }
    }

    return rows;
  }

  trackByBookedLegendRow(index: number, row: { slot: ExactPlayableSlot; advertisementIds: string[] }): string {
    return row.slot.startLabel;
  }

  get availableTimePeriodSlotCount(): number {
    return this.availableExactSlotsForSelectedDuration.length;
  }

  private getLoungeSetFromGroups(groupNames: string[]): Set<string> {
    const lounges = new Set<string>();
    for (const groupName of groupNames) {
      const group = this.advertisementGroups.find((g) => g.groupName === groupName);
      if (!group) {
        continue;
      }

      for (const lounge of this.getLoungesArray(group.lounges)) {
        const normalized = this.normalizeLoungeToken(lounge);
        if (normalized) {
          lounges.add(normalized);
        }
      }
    }
    return lounges;
  }

  private hasLoungeOverlap(a: Set<string>, b: Set<string>): boolean {
    for (const item of a) {
      if (b.has(item)) {
        return true;
      }
    }
    return false;
  }

  private extractBookedSlotLabelsFromAd(ad: any): string[] {
    const sourceSlots: string[] = [];
    const rawSlots = ad?.originalData?.playTimeSlots;

    if (Array.isArray(rawSlots)) {
      for (const item of rawSlots) {
        if (typeof item === 'string') {
          sourceSlots.push(item);
        }
      }
    }

    const rawLegacy = ad?.originalData?.playTimeSlot;
    if (typeof rawLegacy === 'string' && rawLegacy.trim()) {
      sourceSlots.push(...rawLegacy.split(','));
    }

    const labels = new Set<string>();

    for (const slotText of sourceSlots) {
      const text = String(slotText || '').trim();
      const match = text.match(/^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\s*-\s*(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})$/);
      if (!match) {
        continue;
      }

      labels.add(match[1]);
    }

    return Array.from(labels);
  }

  private formatClockFromOffset(offsetSeconds: number): string {
    const base = this.getCycleBaseDateTime();
    const dt = new Date(base.getTime() + offsetSeconds * 1000);
    const hh = String(dt.getHours()).padStart(2, '0');
    const mm = String(dt.getMinutes()).padStart(2, '0');
    const ss = String(dt.getSeconds()).padStart(2, '0');
    return `${hh}:${mm}:${ss}`;
  }

  private buildSelectedPlayTimeSlotsPayload(): string[] {
    const slots: string[] = [];

    if (this.selectedTrafficLevel) {
      slots.push(this.selectedTrafficLevel);
    }

    const selectedPeriod = this.selectedTrafficTimePeriod;
    if (selectedPeriod) {
      slots.push(`${selectedPeriod.name} (${selectedPeriod.startTime}-${selectedPeriod.endTime})`);
    }

    if (this.selectedExactSlots.length > 0) {
      for (const slot of this.selectedExactSlots) {
        slots.push(`${slot.startLabel}-${slot.endLabel}`);
      }
    }

    return Array.from(new Set(slots.map((item) => item.trim()).filter((item) => item.length > 0)));
  }

  private formatDateTime(dt: Date): string {
    const year = dt.getFullYear();
    const month = String(dt.getMonth() + 1).padStart(2, '0');
    const day = String(dt.getDate()).padStart(2, '0');
    const hh = String(dt.getHours()).padStart(2, '0');
    const mm = String(dt.getMinutes()).padStart(2, '0');
    const ss = String(dt.getSeconds()).padStart(2, '0');
    return `${year}-${month}-${day} ${hh}:${mm}:${ss}`;
  }

  private getSelectedTimePeriodRange(): { start: Date; end: Date } {
    const base = this.getCycleBaseDateTime();
    const period = this.selectedTrafficTimePeriod;

    if (!period) {
      return {
        start: base,
        end: new Date(base.getTime() + 60000),
      };
    }

    const start = new Date(base);
    const [startH, startM] = period.startTime.split(':').map((part) => Number(part));
    start.setHours(startH, startM, 0, 0);

    const end = new Date(base);
    const [endH, endM] = period.endTime.split(':').map((part) => Number(part));
    end.setHours(endH, endM, 0, 0);

    if (period.wrapsToNextDay || end.getTime() <= start.getTime()) {
      end.setDate(end.getDate() + 1);
    }

    return { start, end };
  }

  private getCycleBaseDateTime(): Date {
    const today = new Date();
    const datePart =
      (this.scheduleType === 'One-time' && this.oneTimeScheduleDate)
        ? this.oneTimeScheduleDate
        : (this.startDate || today.toISOString().split('T')[0]);

    let timePart = '07:00';
    const selectedPeriod = this.selectedTrafficTimePeriod;
    if (selectedPeriod) {
      timePart = selectedPeriod.startTime;
    }
    if (this.scheduleType === 'One-time' && this.oneTimeScheduleTime) {
      timePart = this.oneTimeScheduleTime;
    } else if (this.frequencyType === 'once' && this.occursOnceTime) {
      timePart = this.occursOnceTime;
    } else if (this.startingTime) {
      timePart = this.startingTime;
    }

    const normalizedTime = `${timePart}:00`.slice(0, 8);
    const composed = `${datePart}T${normalizedTime}`;
    const parsed = new Date(composed);
    if (Number.isNaN(parsed.getTime())) {
      const fallback = new Date();
      fallback.setHours(7, 0, 0, 0);
      return fallback;
    }

    parsed.setSeconds(0, 0);
    return parsed;
  }

  private applySelectedTrafficTimePeriod(syncLevel: boolean): void {
    const period = this.selectedTrafficTimePeriod;
    if (!period) {
      this.refreshAvailableSlotsForSelectedGroups();
      this.updateDescription();
      return;
    }

    if (syncLevel) {
      this.selectedTrafficLevel = period.trafficLevel;
    }

    if (this.scheduleType === 'One-time') {
      this.oneTimeScheduleTime = period.startTime;
    } else {
      this.occursOnceTime = period.startTime;
      this.startingTime = period.startTime;
      this.endingTime = period.endTime;
    }

    this.selectedExactSlotStartIsos = [];
    this.refreshAvailableSlotsForSelectedGroups();
    this.updateDescription();
  }

  private syncTrafficPeriodFromCurrentTimes(): void {
    const currentTime =
      this.scheduleType === 'One-time'
        ? this.oneTimeScheduleTime
        : (this.frequencyType === 'once' ? this.occursOnceTime : this.startingTime);

    if (!currentTime) {
      return;
    }

    const match = this.trafficTimePeriods.find((period) => period.startTime === currentTime);
    if (!match) {
      return;
    }

    this.selectedTrafficLevel = match.trafficLevel;
    this.selectedTimePeriodName = match.name;
  }

  get effectiveAdWindowStartSecond(): number {
    return this.commonAdWindowStartSecond;
  }

  get effectiveAvailableSeconds(): number {
    return this.commonAvailableSeconds;
  }

  get durationOverflowSeconds(): number {
    return Math.max(0, this.selectedDurationSeconds - this.effectiveAvailableSeconds);
  }

  private resetSlotHelperState(): void {
    this.slotSummariesByLounge = [];
    this.slotHelperLoading = false;
    this.slotHelperError = '';
    this.commonAdWindowStartSecond = this.selectedTrafficScheduleSeconds;
    this.commonAvailableSeconds = 0;
    this.interactiveDurationSlots = [];
    this.selectedExactSlotStartIsos = [];
  }

  refreshAvailableSlotsForSelectedGroups(): void {
    if (!this.showModal || this.isViewMode) {
      return;
    }

    if (!this.selectedGroups.length) {
      // Keep the default traffic window visible until a lounge group is selected.
      this.resetSlotHelperState();
      return;
    }

    this.fetchSlotsForCurrentSelection();
  }

  private fetchSlotsForCurrentSelection(): void {
    const selectedLoungeNames = new Set<string>();

    for (const groupName of this.selectedGroups) {
      const group = this.advertisementGroups.find((item) => item.groupName === groupName);
      if (!group) {
        continue;
      }

      for (const loungeName of this.getLoungesArray(group.lounges)) {
        if (loungeName && loungeName.trim()) {
          selectedLoungeNames.add(loungeName.trim());
        }
      }
    }

    if (!selectedLoungeNames.size) {
      this.resetSlotHelperState();
      this.slotHelperError = 'Select a lounge group that has lounges to see available ad slots.';
      return;
    }

    const selectedLounges = this.resolveSelectedLoungeRefs(Array.from(selectedLoungeNames));

    if (!selectedLounges.length) {
      // Do not hard-fail the UI; keep default 6s-30s fallback visibility in modal.
      this.slotSummariesByLounge = [];
      this.slotHelperLoading = false;
      this.slotHelperError =
        'Unable to map selected lounge group to lounge records. Showing default 6s-30s slot window.';
      this.commonAdWindowStartSecond = 6;
      this.commonAvailableSeconds = 24;
      this.interactiveDurationSlots = [];
      return;
    }

    this.slotHelperLoading = true;
    this.slotHelperError = '';

    const requests = selectedLounges.map((entry) =>
      this.advertisementService.getLoungeAdSlots(entry.loungeId).pipe(
        catchError((error) => {
          console.error(`Error loading slot summary for lounge ${entry.loungeName}:`, error);
          return of(null);
        })
      )
    );

    forkJoin(requests).subscribe((results) => {
      const summaries: LoungeSlotEntry[] = [];

      for (let index = 0; index < results.length; index++) {
        const summary = results[index];
        const lounge = selectedLounges[index];
        if (!summary || !lounge) {
          continue;
        }
        summaries.push({
          loungeId: lounge.loungeId,
          loungeName: lounge.loungeName,
          summary,
        });
      }

      this.slotSummariesByLounge = summaries;
      this.slotHelperLoading = false;

      if (!summaries.length) {
        this.commonAvailableSeconds = 0;
        this.interactiveDurationSlots = [];
        this.slotHelperError = 'Unable to load slot summary for selected lounge group(s).';
        return;
      }

      const maxBookedSeconds = Math.max(
        ...summaries.map((entry) => Math.max(0, Math.min(24, entry.summary.bookedSeconds || 0)))
      );
      this.commonAdWindowStartSecond = 6 + maxBookedSeconds;

      const minRemainingSeconds = Math.min(
        ...summaries.map((entry) => Math.max(0, Math.min(24, entry.summary.remainingSeconds || 0)))
      );
      const boundedRemaining = Math.max(0, Math.min(minRemainingSeconds, 30 - this.commonAdWindowStartSecond));
      this.commonAvailableSeconds = boundedRemaining;

      this.interactiveDurationSlots = Array.from({ length: this.commonAvailableSeconds }, (_, i) => {
        const duration = i + 1;
        return {
          durationSeconds: duration,
          startSecond: this.commonAdWindowStartSecond,
          endSecond: this.commonAdWindowStartSecond + duration,
        };
      });
    });
  }

  private resolveSelectedLoungeRefs(rawNames: string[]): LoungeRef[] {
    const normalizedCatalog = this.loungesCatalog.map((lounge) => ({
      lounge,
      normalizedName: this.normalizeLoungeToken(lounge.loungeName),
      normalizedId: this.normalizeLoungeToken(lounge.id),
    }));

    const resolved: LoungeRef[] = [];
    const seen = new Set<string>();

    for (const raw of rawNames) {
      const normalized = this.normalizeLoungeToken(raw);
      if (!normalized) {
        continue;
      }

      if (this.looksLikeUuid(normalized)) {
        const directMatch = this.loungesCatalog.find(
          (lounge) => this.normalizeLoungeToken(lounge.id) === normalized
        );

        if (directMatch) {
          const directKey = directMatch.id;
          if (!seen.has(directKey)) {
            resolved.push({ loungeId: directMatch.id, loungeName: directMatch.loungeName });
            seen.add(directKey);
          }
          continue;
        }

        if (!seen.has(normalized)) {
          resolved.push({ loungeId: normalized, loungeName: normalized });
          seen.add(normalized);
        }
        continue;
      }

      const match = normalizedCatalog.find(
        (entry) => entry.normalizedName === normalized || entry.normalizedId === normalized
      );

      if (match) {
        const key = match.lounge.id;
        if (!seen.has(key)) {
          resolved.push({ loungeId: match.lounge.id, loungeName: match.lounge.loungeName });
          seen.add(key);
        }
      }
    }

    // If direct group mapping fails but context already has a selected lounge, use it.
    if (!resolved.length && this.selectedLoungeId) {
      const byId = this.loungesCatalog.find(
        (lounge) => this.normalizeLoungeToken(lounge.id) === this.normalizeLoungeToken(this.selectedLoungeId)
      );
      if (byId) {
        return [{ loungeId: byId.id, loungeName: byId.loungeName }];
      }
    }

    if (!resolved.length && this.selectedLoungeName) {
      const byName = this.loungesCatalog.find(
        (lounge) =>
          this.normalizeLoungeToken(lounge.loungeName) === this.normalizeLoungeToken(this.selectedLoungeName)
      );
      if (byName) {
        return [{ loungeId: byName.id, loungeName: byName.loungeName }];
      }
    }

    return resolved;
  }

  private normalizeLoungeToken(value: string | null | undefined): string {
    return (value || '')
      .toLowerCase()
      .replace(/[\[\]"']/g, '')
      .replace(/\s+/g, ' ')
      .trim();
  }

  private looksLikeUuid(value: string): boolean {
    return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value);
  }

  // Conflict Detection and Resolution Methods
  checkScheduleConflict(): boolean {
    const currentTime = this.getScheduleTimeSlot();
    console.log('Current time slot:', currentTime);
    
    // Get all lounges from selected groups
    const newAdLounges = new Set<string>();
    
    this.selectedGroups.forEach((groupName: string) => {
      const group = this.advertisementGroups.find(g => g.groupName === groupName);
      if (group) {
        const loungesList = this.getLoungesArray(group.lounges);
        loungesList.forEach((lounge: string) => {
          newAdLounges.add(lounge);
        });
      }
    });

    console.log('New ad lounges:', Array.from(newAdLounges));

    // Check for conflicts with existing ads
    for (const ad of this.advertisements) {
      console.log(`Checking ad: ${ad.name} (${ad.loungeGroups}) at ${ad.scheduledTime}`);
      
      // Skip if not same schedule type
      if (ad.scheduleType !== this.scheduleType) {
        console.log('  - Different schedule type, skipping');
        continue;
      }
      
      // Skip if different time
      const adTime = ad.scheduledTime || this.extractTimeFromAd(ad);
      if (adTime !== currentTime) {
        console.log(`  - Different time (${adTime} vs ${currentTime}), skipping`);
        continue;
      }

      console.log('  - Same schedule type and time, checking lounges...');

      // Get lounges from existing ad's groups
      const existingAdGroups = ad.loungeGroups.split(',').map((g: string) => g.trim());
      const existingAdLounges = new Set<string>();
      
      existingAdGroups.forEach((groupName: string) => {
        const group = this.advertisementGroups.find(g => g.groupName === groupName);
        if (group) {
          const loungesList = this.getLoungesArray(group.lounges);
          loungesList.forEach((lounge: string) => {
            existingAdLounges.add(lounge);
          });
        }
      });

      console.log('  - Existing ad lounges:', Array.from(existingAdLounges));

      // Check for overlapping lounges
      for (const lounge of newAdLounges) {
        if (existingAdLounges.has(lounge)) {
          console.log(`  - CONFLICT FOUND! Lounge "${lounge}" overlaps`);
          // Found conflict!
          const conflictingGroup = this.selectedGroups.find((groupName: string) => {
            const group = this.advertisementGroups.find(g => g.groupName === groupName);
            if (group) {
              const loungesList = this.getLoungesArray(group.lounges);
              return loungesList.includes(lounge);
            }
            return false;
          }) || this.selectedGroups[0];

          this.showConflictDialog(lounge, conflictingGroup, currentTime);
          return true;
        }
      }
    }

    console.log('No conflicts found');
    return false;
  }

  getScheduleTimeSlot(): string {
    if (this.scheduleType === 'Recurring' && this.occursType === 'Daily') {
      return this.occursOnceTime || '12:00';
    } else if (this.scheduleType === 'One-time') {
      return this.startDate + ' ' + (this.occursOnceTime || '12:00');
    } else if (this.scheduleType === 'On startup') {
      return 'On Startup';
    } else if (this.scheduleType === 'On Idle') {
      return 'On Idle';
    }
    return this.occursOnceTime || '12:00';
  }

  extractTimeFromAd(ad: any): string {
    // Extract time from existing advertisement
    if (ad.scheduledTime) {
      return ad.scheduledTime;
    }
    // Default fallback
    return '12:00';
  }

  showConflictDialog(lounge: string, groupToRemoveFrom: string, timeSlot: string) {
    const timeDisplay = this.formatTimeSlot(timeSlot);
    
    this.conflictData = {
      message: `${lounge} already has an advertisement scheduled from ${timeDisplay} Time slot. This advertisement will overlap.`,
      removeOption: `Remove ${lounge} from this ${groupToRemoveFrom}`,
      conflictingLounge: lounge,
      conflictingGroup: groupToRemoveFrom,
      timeSlot: timeDisplay
    };

    this.selectedConflictOption = '';
    this.showConflictModal = true;
  }

  formatTimeSlot(timeSlot: string): string {
    if (timeSlot === 'On Startup' || timeSlot === 'On Idle') {
      return timeSlot;
    }

    // Convert 24-hour time to 12-hour format
    if (timeSlot.includes(':')) {
      // Format start time
      const [hours, minutes] = timeSlot.split(':');
      const hour = parseInt(hours);
      const period = hour >= 12 ? 'PM' : 'AM';
      const hour12 = hour % 12 || 12;
      
      // Format end time using the actual endingTime from form
      const endTime = this.endingTime || '23:59';
      const [endHours, endMinutes] = endTime.split(':');
      const endHour = parseInt(endHours);
      const endPeriod = endHour >= 12 ? 'PM' : 'AM';
      const endHour12 = endHour % 12 || 12;
      
      return `${hour12.toString().padStart(2, '0')}:${minutes} ${period} to ${endHour12.toString().padStart(2, '0')}:${endMinutes} ${endPeriod}`;
    }

    return timeSlot;
  }

  formatConflictTimeSlot(timeSlot: string): string {
    if (!timeSlot) {
      return this.formatTimeSlot(this.getScheduleTimeSlot());
    }

    const normalized = timeSlot.trim();
    if (normalized.toLowerCase() === 'on startup' || normalized.toLowerCase() === 'on idle') {
      return normalized;
    }

    if (normalized.toLowerCase().includes(' to ')) {
      const parts = normalized.split(/\s+to\s+/i);
      if (parts.length === 2) {
        return `${this.to12Hour(parts[0])} to ${this.to12Hour(parts[1])}`;
      }
    }

    return this.to12Hour(normalized);
  }

  private to12Hour(time: string): string {
    const trimmed = time.trim();

    // If already contains AM/PM, just normalize casing
    if (/am|pm/i.test(trimmed)) {
      return trimmed.toUpperCase();
    }

    const segments = trimmed.split(':');
    if (segments.length < 2) {
      return trimmed;
    }

    const hour = parseInt(segments[0], 10);
    const minutes = segments[1].replace(/[^0-9]/g, '').padStart(2, '0');
    if (Number.isNaN(hour)) {
      return trimmed;
    }

    const period = hour >= 12 ? 'PM' : 'AM';
    const hour12 = hour % 12 || 12;

    return `${hour12.toString().padStart(2, '0')}:${minutes} ${period}`;
  }

  selectConflictOption(option: string) {
    this.selectedConflictOption = option;
  }

  confirmConflictResolution() {
    if (!this.selectedConflictOption) {
      alert('Please select an option');
      return;
    }

    switch (this.selectedConflictOption) {
      case 'remove':
        this.removeConflictingLounge();
        break;
      case 'changeTime':
        this.changeTimeSlot();
        break;
      case 'cancel':
        this.cancelSave();
        break;
    }

    // Only close modal for cancel option, others handle it themselves
    if (this.selectedConflictOption === 'cancel') {
      this.closeConflictModal();
    }
  }

  removeConflictingLounge() {
    const group = this.conflictData.conflictingGroup;
    const lounge = this.conflictData.conflictingLounge;

    if (group && lounge) {
      if (!this.excludedLoungesByGroup[group]) {
        this.excludedLoungesByGroup[group] = new Set<string>();
      }
      const loungesToExclude = this.conflictData.affectedLounges && this.conflictData.affectedLounges.length > 0
        ? this.conflictData.affectedLounges
        : [lounge];
      loungesToExclude.forEach((l: string) => this.excludedLoungesByGroup[group].add(l));

      const groupRecord = this.advertisementGroups.find(g => g.groupName === group);
      if (groupRecord) {
        const currentLounges = this.getLoungesArray(groupRecord.lounges);
        const updatedLounges = currentLounges.filter((l: string) => !loungesToExclude.some((x: string) => x.trim().toLowerCase() === l.trim().toLowerCase()));

        // Only persist if there is an actual change
        if (updatedLounges.length !== currentLounges.length) {
          this.advertisementService.updateGroup(groupRecord.id, {
            groupName: groupRecord.groupName,
            lounges: updatedLounges
          }).subscribe({
            next: () => {
              // Update local state so the groups table reflects the removal without waiting
              groupRecord.lounges = JSON.stringify(updatedLounges);
              // Also update selectedLounges if this group is currently selected and visible
              if (this.isViewGroupMode && this.groupData.groupName === groupRecord.groupName) {
                this.selectedLounges = updatedLounges;
              }
              this.loadAdvertisementGroups();

              this.closeConflictModal();
              const nextIndex = this.currentConflictGroupIndex >= 0 ? this.currentConflictGroupIndex + 1 : 0;
              this.checkConflictForGroup(nextIndex);
            },
            error: (err) => {
              console.error('Failed to update group after removing lounge:', err);
              alert('Failed to update the advertisement group after removing the overlapping lounge.');
              this.closeConflictModal();
              const nextIndex = this.currentConflictGroupIndex >= 0 ? this.currentConflictGroupIndex + 1 : 0;
              this.checkConflictForGroup(nextIndex);
            }
          });
          return;
        }
      }
    }

    this.closeConflictModal();
    const nextIndex = this.currentConflictGroupIndex >= 0 ? this.currentConflictGroupIndex + 1 : 0;
    this.checkConflictForGroup(nextIndex);
  }

  changeTimeSlot() {
    // Close conflict modal and allow user to change the time
    this.showConflictModal = false;
    // The modal remains open for the user to change the time slot
    alert('Please adjust the schedule time to avoid conflict');
  }

  cancelSave() {
    // Simply close the conflict modal and don't save
    this.showConflictModal = false;
  }

  closeConflictModal() {
    this.showConflictModal = false;
    this.selectedConflictOption = '';
  }

  proceedWithSave() {
    // Actual save logic after conflict resolution
    this.saveAdvertisementWithoutCheck();
  }

  saveAdvertisementWithoutCheck() {
    // Get selected weekly days as comma-separated string
    const getWeeklyDays = (): string | undefined => {
      if (this.occursType !== 'Weekly') return undefined;
      const days = [];
      if (this.weeklyMonday) days.push('Monday');
      if (this.weeklyTuesday) days.push('Tuesday');
      if (this.weeklyWednesday) days.push('Wednesday');
      if (this.weeklyThursday) days.push('Thursday');
      if (this.weeklyFriday) days.push('Friday');
      if (this.weeklySaturday) days.push('Saturday');
      if (this.weeklySunday) days.push('Sunday');
      return days.length > 0 ? days.join(',') : undefined;
    };

    // Normalize schedule type for comparison (case-insensitive)
    const normalizedScheduleType = this.scheduleType.toLowerCase();

    console.log('=== Advertisement Save Debug ===');
    console.log('Schedule Type:', this.scheduleType);
    console.log('Normalized Schedule Type:', normalizedScheduleType);
    console.log('Frequency Type:', this.frequencyType);
    console.log('Occurs Type:', this.occursType);
    console.log('Recurrence Interval:', this.recurrenceInterval);
    console.log('Occurs Every Interval:', this.occursEveryInterval);
    console.log('Occurs Every Unit:', this.occursEveryUnit);
    console.log('');
    console.log('=== Date/Time Values ===');
    console.log('Start Date (recurring):', this.startDate);
    console.log('End Date (recurring):', this.endDate);
    console.log('No End Date:', this.noEndDate);
    console.log('One-time Date:', this.oneTimeScheduleDate);
    console.log('One-time Time:', this.oneTimeScheduleTime);
    console.log('Starting Time:', this.startingTime);
    console.log('Ending Time:', this.endingTime);
    console.log('Occurs Once Time:', this.occursOnceTime);
    console.log('Max Idle Duration:', this.maxIdleLoopDuration);
    console.log('');

    // Validate one-time schedule has required date/time
    if (normalizedScheduleType === 'one-time') {
      if (!this.oneTimeScheduleDate || !this.oneTimeScheduleTime) {
        alert('Please fill in both Schedule date and Schedule time for one-time advertisements');
        console.error('One-time schedule missing date or time!');
        return;
      }
      console.log('One-time schedule validation passed ✓');
      console.log(`Will create occurs_once_at: ${this.oneTimeScheduleDate}T${this.oneTimeScheduleTime}:00Z`);
    }

    // Prepare occursEveryInterval - send just the number (database column is INTEGER)
    // The unit (Hourly/Minutes/Secs) is stored separately or inferred from context
    let occursEveryValue: number | undefined = undefined;
    if (normalizedScheduleType === 'recurring' && this.frequencyType === 'every') {
      occursEveryValue = this.occursEveryInterval;
      console.log('Occurs every interval:', occursEveryValue, this.occursEveryUnit);
    }

    // Prepare occurs_once_at timestamp
    let occursOnceAtValue: string | undefined = undefined;
    
    if (normalizedScheduleType === 'one-time') {
      // For One-time: use Schedule date + Schedule time
      occursOnceAtValue = this.convertLocalToUTC(this.oneTimeScheduleDate, this.oneTimeScheduleTime);
      console.log('One-time occurs_once_at:', occursOnceAtValue);
    } else if (normalizedScheduleType === 'recurring' && this.frequencyType === 'once' && this.occursOnceTime) {
      // For Recurring with "Occurs once at": use Start date + Occurs once at time
      if (this.startDate) {
        occursOnceAtValue = this.convertLocalToUTC(this.startDate, this.occursOnceTime);
        console.log('Recurring "occurs once at" → occurs_once_at:', occursOnceAtValue);
      } else {
        console.warn('Recurring schedule with "occurs once at" but no start date - using today');
        const today = new Date().toISOString().split('T')[0];
        occursOnceAtValue = this.convertLocalToUTC(today, this.occursOnceTime);
      }
    }

    const persistAdvertisement = (resolvedMediaUrl: string | undefined, resolvedMediaType: string | undefined) => {
      // Keep the form model in sync so edit mode always retains the latest media URL.
      this.viewData.mediaUrl = resolvedMediaUrl || '';

      const selectedSlotNote = this.selectedExactSlots.length
        ? ` Preferred cycle slots: ${this.selectedExactSlots.map((slot) => `${slot.startLabel}-${slot.endLabel}`).join(', ')} in 30s cycle (00-${this.selectedTrafficScheduleSeconds.toString().padStart(2, '0')} reserved for bus schedules).`
        : '';
      const selectedPeriod = this.selectedTrafficTimePeriod;
      const selectedPeriodNote = selectedPeriod
        ? ` Time period: ${selectedPeriod.name} (${this.formatTrafficPeriod(selectedPeriod)}), ${selectedPeriod.trafficLevel}.`
        : '';
      const descriptionWithSlot = `${this.viewData.description || ''}${selectedPeriodNote}${selectedSlotNote}`.trim();
      const playTimeSlots = this.buildSelectedPlayTimeSlotsPayload();

      // Map UI fields to API payload
      const payload = {
      advertisementName: this.viewData.name.trim(),
      advertisementCategory: (this.viewData.category || 'commercial').toLowerCase(),
      priority: this.viewData.priority.toLowerCase(),
      scheduleType: normalizedScheduleType,
      loungeGroupName: this.selectedGroups.join(',') || undefined,
      description: descriptionWithSlot || undefined,
      mediaDuration: this.parseDurationValue(this.viewData.duration),
      mediaUrl: resolvedMediaUrl,
      mediaType: resolvedMediaType,
      // Frequency only for Recurring schedules
      frequency: normalizedScheduleType === 'recurring' ? (this.occursType ? this.occursType.toLowerCase() : undefined) : undefined,
      // Recurrence interval - for ALL Recurring schedules (every X days/weeks/months)
      recurrenceInterval: normalizedScheduleType === 'recurring' ? this.recurrenceInterval : undefined,
      // Occurs once at timestamp - for BOTH One-time and Recurring (when "occurs once at" is selected)
      occursOnceAt: occursOnceAtValue,
      // Occurs every interval - only for Recurring with "every" frequency type (number only)
      occursEveryInterval: occursEveryValue,
      // Start time - for Recurring schedules (time of day)
      startTime: normalizedScheduleType === 'recurring' ? (this.frequencyType === 'once' ? this.occursOnceTime : this.startingTime) : undefined,
      // End time - for all Recurring schedules (when endingTime is provided)
      endTime: normalizedScheduleType === 'recurring' && this.endingTime ? this.endingTime : undefined,
      // Multiple play time slots (preferred) + single-string legacy compatibility
      playTimeSlots,
      playTimeSlot: playTimeSlots.length > 0 ? playTimeSlots.join(',') : undefined,
      // Start date and end date - for Recurring schedules (date range)
      startDate: normalizedScheduleType === 'recurring' ? this.startDate : undefined,
      endDate: normalizedScheduleType === 'recurring' && !this.noEndDate ? this.endDate : undefined,
      // Weekly days - only for Weekly recurring schedules
      weeklyDays: getWeeklyDays(),
      // Monthly fields - only for Monthly recurring schedules
      monthlyDayOfMonth: normalizedScheduleType === 'recurring' && this.occursType === 'Monthly' && this.monthlyType === 'dayOfMonth' ? this.monthlyDayOfMonth : undefined,
      monthlyWeek: normalizedScheduleType === 'recurring' && this.occursType === 'Monthly' && this.monthlyType === 'weekday' ? this.monthlyWeek.toLowerCase() : undefined,
      monthlyDay: normalizedScheduleType === 'recurring' && this.occursType === 'Monthly' && this.monthlyType === 'weekday' ? this.monthlyDay.toLowerCase() : undefined,
      // Max idle duration - only for On Idle schedules
      maxIdleLoopDuration: normalizedScheduleType === 'on idle' ? this.maxIdleLoopDuration : undefined
    };

      // Force high priority for emergency category
      if (payload.advertisementCategory === 'emergency') {
        payload.priority = 'high';
      }

      // Set status based on enabled checkbox: 'active' when enabled, 'paused' when disabled
      const status = this.isEnabled ? 'active' : 'paused';
      (payload as any).status = status;

      console.log('=== Payload Being Sent ===');
      console.log('Complete Payload:', JSON.stringify(payload, null, 2));
      console.log('');
      console.log('Specific Fields:');
      console.log('- scheduleType:', payload.scheduleType);
      console.log('- frequency:', payload.frequency);
      console.log('- recurrenceInterval:', payload.recurrenceInterval);
      console.log('- occursEveryInterval:', payload.occursEveryInterval);
      console.log('- startDate:', payload.startDate);
      console.log('- endDate:', payload.endDate);
      console.log('- startTime:', payload.startTime);
      console.log('- endTime:', payload.endTime);
      console.log('- occursOnceAt:', payload.occursOnceAt);
      console.log('- playTimeSlots:', payload.playTimeSlots);
      console.log('- playTimeSlot (legacy):', payload.playTimeSlot);
      console.log('- maxIdleLoopDuration:', payload.maxIdleLoopDuration);

      // Check if editing existing advertisement or creating new one
      if (this.isEditMode && this.viewData.addNo) {
        const advertisementId = this.viewData.addNo;
        console.log('Updating advertisement with ID:', advertisementId);

        this.advertisementService.updateAdvertisement(advertisementId, payload as any).subscribe({
          next: (ad) => {
            console.log('Advertisement updated:', ad);
            this.isSaving = false;
            this.loadAdvertisements();
            this.closeModal();
            alert('Advertisement updated successfully');
          },
          error: (error) => {
            this.isSaving = false;
            console.error('Error updating advertisement:', error);
            const msg = error?.error?.error || error?.message || 'Failed to update advertisement';
            alert(msg);
          }
        });
      } else {
        console.log('Creating new advertisement');

        this.advertisementService.createAdvertisement(payload as any).subscribe({
          next: (ad) => {
            console.log('Advertisement created:', ad);
            this.isSaving = false;
            this.loadAdvertisements();
            this.closeModal();
            alert('Advertisement created successfully');
          },
          error: (error) => {
            this.isSaving = false;
            console.error('Error creating advertisement:', error);
            const msg = error?.error?.error || error?.message || 'Failed to create advertisement';
            alert(msg);
          }
        });
      }
    };

    this.isSaving = true;

    if (this.selectedFile) {
      this.advertisementService.uploadMedia(this.selectedFile).subscribe({
        next: (uploadRes: UploadMediaResponse) => {
          this.viewData.fileName = uploadRes.fileName;
          this.viewData.mediaUrl = uploadRes.mediaUrl;
          persistAdvertisement(uploadRes.mediaUrl, uploadRes.mediaType || this.inferMediaTypeFromUrl(uploadRes.mediaUrl));
        },
        error: (error) => {
          this.isSaving = false;
          console.error('Error uploading media:', error);
          const msg = error?.error?.error || error?.message || 'Failed to upload media file';
          alert(msg);
        }
      });
      return;
    }

    const existingMediaUrl = this.resolveExistingMediaUrl();
    const existingMediaType = existingMediaUrl ? this.inferMediaTypeFromUrl(existingMediaUrl) : undefined;
    persistAdvertisement(existingMediaUrl, existingMediaType);
  }

  private resolveExistingMediaUrl(): string | undefined {
    const mediaUrl = (this.viewData.mediaUrl || '').trim();
    if (mediaUrl) {
      return mediaUrl;
    }

    // Backward compatibility for records that only retained the file name.
    const fileName = (this.viewData.fileName || '').trim();
    if (!fileName) {
      return undefined;
    }

    if (fileName.startsWith('http://') || fileName.startsWith('https://') || fileName.startsWith('/media/')) {
      return fileName;
    }

    return `/media/${fileName}`;
  }

  private inferMediaTypeFromUrl(mediaUrl: string): string {
    const url = mediaUrl.toLowerCase();
    if (url.endsWith('.mp4') || url.endsWith('.webm') || url.endsWith('.ogg') || url.endsWith('.mov') || url.endsWith('.avi')) {
      return 'video';
    }
    return 'image';
  }
}
