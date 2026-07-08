import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from './api.service';

export interface DepartureInfo {
  loungeId: string;
  loungeName: string;
  activeTripId: string;
  busNo: string;
  routeNumber: string;
  origin: string;
  destination: string;
  currentLatitude: number | null;
  currentLongitude: number | null;
  actualDeparture: string | null;
  eta: string | null;
  status: string;
  remarks: string;
  time: string;
}

export interface LoungeDepartureResponse {
  loungeId: string;
  loungeName: string;
  departures: DepartureInfo[];
}

@Injectable({ providedIn: 'root' })
export class DepartureService {
  constructor(private api: ApiService) {}

  getAllLoungeDepartures(): Observable<LoungeDepartureResponse[]> {
    return this.api.get<LoungeDepartureResponse[]>('/departures');
  }

  getDeparturesByLoungeId(loungeId: string): Observable<LoungeDepartureResponse> {
    return this.api.get<LoungeDepartureResponse>(`/departures/lounge/${loungeId}`);
  }
}