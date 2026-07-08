import { Injectable } from '@angular/core';
import { combineLatest, map, Observable, of, startWith, catchError } from 'rxjs';
import { ArrivalService } from './arrival.service';
import { DepartureService } from './departure.service';

export interface DisplayDepartureRow {
  time: string;
  busNo: string;
  routeNo: string;
  destination: string;
  remarks: string;
  status: string;
  indicator: boolean;
}

export interface DisplayArrivalRow {
  time: string;
  busNo: string;
  routeNo: string;
  origin: string;
  remarks: string;
  status: string;
  indicator: boolean;
}

export interface LoungeScheduleData {
  loungeName: string;
  departures: DisplayDepartureRow[];
  arrivals: DisplayArrivalRow[];
}

@Injectable({
  providedIn: 'root'
})
export class ScheduleService {
  constructor(
    private departureService: DepartureService,
    private arrivalService: ArrivalService
  ) {}

  getScheduleByLounge(loungeId: string): Observable<LoungeScheduleData> {
    return combineLatest({
      departures: this.departureService.getDeparturesByLoungeId(loungeId).pipe(
        startWith({ loungeId, loungeName: '', departures: [] }),
        catchError(() => of({ loungeId, loungeName: '', departures: [] }))
      ),
      arrivals: this.arrivalService.getArrivalsByLoungeId(loungeId).pipe(
        startWith({ loungeId, loungeName: '', arrivals: [] }),
        catchError(() => of({ loungeId, loungeName: '', arrivals: [] }))
      )
    }).pipe(
      map(({ departures, arrivals }) => ({
        loungeName: departures.loungeName || arrivals.loungeName || '',
        departures: departures.departures.map((dep) => ({
          time: this.toDisplayTime(dep.time),
          busNo: dep.busNo,
          routeNo: dep.routeNumber,
          destination: dep.destination,
          remarks: dep.remarks || dep.status || '',
          status: this.getStatusClass(dep.remarks || dep.status || ''),
          indicator: this.hasIndicator(dep.remarks || dep.status || '')
        })),
        arrivals: arrivals.arrivals.map((arr) => ({
          time: this.toDisplayTime(arr.time),
          busNo: arr.busNo,
          routeNo: arr.routeNumber,
          origin: arr.origin,
          remarks: arr.remarks || arr.status || '',
          status: this.getStatusClass(arr.remarks || arr.status || ''),
          indicator: this.hasIndicator(arr.remarks || arr.status || '')
        }))
      }))
    );
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
    const lowerRemarks = remarks.toLowerCase();
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
    const lowerRemarks = remarks.toLowerCase();
    return (
      lowerRemarks.includes('check in') ||
      lowerRemarks.includes('boarding') ||
      lowerRemarks.includes('delay') ||
      lowerRemarks.includes('expected') ||
      lowerRemarks.includes('on time') ||
      lowerRemarks.includes('arriving')
    );
  }
}
