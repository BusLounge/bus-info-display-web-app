import { Injectable } from '@angular/core';

export interface TrafficLevelSchedule {
  level: 'Peak' | 'Moderate' | 'Off-Peak';
  startTime: string; // HH:mm format
  endTime: string;   // HH:mm format
  daysOfWeek: string[]; // ['Mon', 'Tue', ...] or empty for all days
}

@Injectable({ providedIn: 'root' })
export class TrafficLevelService {
  // Default traffic level schedule
  private readonly defaultSchedule: TrafficLevelSchedule[] = [
    {
      level: 'Peak',
      startTime: '06:00',
      endTime: '09:00',
      daysOfWeek: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri']
    },
    {
      level: 'Peak',
      startTime: '16:00',
      endTime: '19:00',
      daysOfWeek: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri']
    },
    {
      level: 'Moderate',
      startTime: '09:00',
      endTime: '16:00',
      daysOfWeek: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri']
    },
    {
      level: 'Moderate',
      startTime: '10:00',
      endTime: '20:00',
      daysOfWeek: ['Sat', 'Sun']
    },
    {
      level: 'Off-Peak',
      startTime: '19:00',
      endTime: '06:00',
      daysOfWeek: []
    }
  ];

  private customSchedule: TrafficLevelSchedule[] | null = null;

  /**
   * Get the current traffic level based on time of day
   */
  getCurrentTrafficLevel(): 'Peak' | 'Moderate' | 'Off-Peak' {
    return this.getTrafficLevelAt(new Date());
  }

  /**
   * Get traffic level at a specific date/time
   */
  getTrafficLevelAt(date: Date): 'Peak' | 'Moderate' | 'Off-Peak' {
    const schedule = this.customSchedule || this.defaultSchedule;
    const dayName = this.getDayName(date);
    const timeStr = this.formatTime(date);

    for (const slot of schedule) {
      // Check if day matches (if specified)
      if (slot.daysOfWeek.length > 0 && !slot.daysOfWeek.includes(dayName)) {
        continue;
      }

      // Check if time falls within this slot
      if (this.isTimeInRange(timeStr, slot.startTime, slot.endTime)) {
        return slot.level;
      }
    }

    return 'Moderate'; // Default fallback
  }

  /**
   * Set custom traffic level schedule
   */
  setCustomSchedule(schedule: TrafficLevelSchedule[]): void {
    this.customSchedule = schedule;
  }

  /**
   * Reset to default schedule
   */
  resetToDefault(): void {
    this.customSchedule = null;
  }

  /**
   * Helper: Check if time falls within range
   */
  private isTimeInRange(time: string, start: string, end: string): boolean {
    const timeMs = this.timeToMinutes(time);
    const startMs = this.timeToMinutes(start);
    const endMs = this.timeToMinutes(end);

    // Handle wrap-around (e.g., 19:00-06:00)
    if (startMs > endMs) {
      return timeMs >= startMs || timeMs < endMs;
    }

    return timeMs >= startMs && timeMs < endMs;
  }

  /**
   * Convert time string (HH:mm) to minutes since midnight
   */
  private timeToMinutes(timeStr: string): number {
    const [hours, minutes] = timeStr.split(':').map(Number);
    return hours * 60 + minutes;
  }

  /**
   * Format date to HH:mm
   */
  private formatTime(date: Date): string {
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${hours}:${minutes}`;
  }

  /**
   * Get day name (Mon, Tue, etc.)
   */
  private getDayName(date: Date): string {
    const dayIndex = date.getDay();
    const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    return days[dayIndex];
  }
}
