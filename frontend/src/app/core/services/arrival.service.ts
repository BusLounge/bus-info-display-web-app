import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from './api.service';

export interface ArrivalInfo {
  loungeId: string;
  loungeName: string;
  activeTripId: string;
  busNo: string;
  routeNumber: string;
  origin: string;
  destination: string;
  currentLatitude: number | null;
  currentLongitude: number | null;
  eta: string | null;
  actualArrival: string | null;
  status: string;
  remarks: string;
  time: string;
}

export interface LoungeArrivalResponse {
  loungeId: string;
  loungeName: string;
  arrivals: ArrivalInfo[];
}

@Injectable({ providedIn: 'root' })
export class ArrivalService {
  constructor(private api: ApiService) {}

  getAllLoungeArrivals(): Observable<LoungeArrivalResponse[]> {
    return this.api.get<LoungeArrivalResponse[]>('/arrivals');
  }

  getArrivalsByLoungeId(loungeId: string): Observable<LoungeArrivalResponse> {
    return this.api.get<LoungeArrivalResponse>(`/arrivals/lounge/${loungeId}`);
  }
}
