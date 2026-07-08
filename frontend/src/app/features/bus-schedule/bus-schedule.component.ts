import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { forkJoin, map, of } from 'rxjs';
import { HeaderComponent } from '../../shared/components/header/header.component';
import { LoungeService } from '../../core/services/lounge.service';
import { ArrivalService, LoungeArrivalResponse } from '../../core/services/arrival.service';
import { DepartureService, LoungeDepartureResponse } from '../../core/services/departure.service';

interface LoungeTableRow {
  id: string;
  name: string;
  district: string;
  arrivals: string[];
  departures: string[];
}

@Component({
  selector: 'app-bus-schedule',
  standalone: true,
  imports: [HeaderComponent, CommonModule, FormsModule],
  templateUrl: './bus-schedule.component.html',
  styleUrl: './bus-schedule.component.scss'
})
export class BusScheduleComponent implements OnInit {
  searchTerm: string = '';
  lounges: LoungeTableRow[] = [];
  loading: boolean = true;
  error: string = '';

  get filteredLounges() {
    if (!this.searchTerm.trim()) {
      return this.lounges;
    }
    const term = this.searchTerm.toLowerCase();
    return this.lounges.filter(lounge => 
      lounge.id.toLowerCase().includes(term) ||
      lounge.name.toLowerCase().includes(term) ||
      lounge.district.toLowerCase().includes(term)
    );
  }

  constructor(
    private router: Router,
    private loungeService: LoungeService,
    private arrivalService: ArrivalService,
    private departureService: DepartureService
  ) {}

  ngOnInit() {
    this.loadLoungeSchedules();
  }

  loadLoungeSchedules() {
    this.loading = true;
    this.error = '';

    forkJoin({
      lounges: this.loungeService.getAllLounges(),
      arrivals: this.arrivalService.getAllLoungeArrivals(),
      departures: this.departureService.getAllLoungeDepartures()
    }).pipe(
      map(({ lounges, arrivals, departures }) => {
        const arrivalMap = new Map<string, LoungeArrivalResponse>();
        arrivals.forEach(a => arrivalMap.set(a.loungeId, a));

        const departureMap = new Map<string, LoungeDepartureResponse>();
        departures.forEach(d => departureMap.set(d.loungeId, d));

        return lounges.map(lounge => {
          const loungeArrivals = arrivalMap.get(lounge.id);
          const loungeDepartures = departureMap.get(lounge.id);

          return {
            id: lounge.id,
            name: lounge.loungeName,
            district: lounge.district || 'N/A',
            arrivals: loungeArrivals ? loungeArrivals.arrivals.map(a => a.time).filter(t => t) : [],
            departures: loungeDepartures ? loungeDepartures.departures.map(d => d.time).filter(t => t) : []
          };
        });
      })
    ).subscribe({
      next: (combinedLounges) => {
        this.lounges = combinedLounges;
        this.loading = false;
      },
      error: (err) => {
        console.error('Error loading lounge schedules:', err);
        this.error = 'Failed to load lounge schedules. Please try again.';
        this.loading = false;
      }
    });
  }

  viewSchedules(lounge: LoungeTableRow) {
    this.router.navigate(['/bids-display'], {
      queryParams: {
        loungeId: lounge.id,
        loungeName: lounge.name,
        kiosk: 'true'
      }
    });
  }

  viewAds(lounge: LoungeTableRow) {
    this.router.navigate(['/advertisements'], {
      queryParams: {
        loungeId: lounge.id,
        loungeName: lounge.name
      }
    });
  }
}
