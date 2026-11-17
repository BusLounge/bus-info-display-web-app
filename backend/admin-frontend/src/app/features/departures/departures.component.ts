import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HeaderComponent } from '../../shared/components/header/header.component';
import { DepartureService } from '../../core/services';
import type { LoungeDepartureResponse, DepartureInfo } from '../../core/services';

@Component({
  selector: 'app-departures',
  standalone: true,
  imports: [HeaderComponent, CommonModule, FormsModule],
  templateUrl: './departures.component.html',
  styleUrl: './departures.component.scss'
})
export class DeparturesComponent implements OnInit {
  searchTerm: string = '';
  lounges: any[] = [];
  isLoading: boolean = false;
  error: string | null = null;

  constructor(private departureService: DepartureService) {}

  ngOnInit(): void {
    this.loadDepartures();
  }

  loadDepartures(): void {
    this.isLoading = true;
    this.error = null;
    
    this.departureService.getAllLoungeDepartures().subscribe({
      next: (loungeDepartures: LoungeDepartureResponse[]) => {
        // Map departures to the component's structure
        this.lounges = loungeDepartures.map(lounge => ({
          id: lounge.loungeId,
          name: lounge.loungeName,
          departures: lounge.departures.map(departure => this.mapDepartureToDisplay(departure))
        }));
        this.isLoading = false;
      },
      error: (err) => {
        console.error('Error loading departures:', err);
        this.error = 'Failed to load departures. Please try again.';
        this.isLoading = false;
        
        // Fallback to empty array
        this.lounges = [];
      }
    });
  }

  private mapDepartureToDisplay(departure: DepartureInfo): any {
    const status = this.getStatusClass(departure.remarks);
    const isActive = status === 'checkin' || status === 'ontime' || departure.remarks.includes('Boarding');
    
    return {
      time: departure.time || '-',
      busNo: departure.busNo,
      routeNumber: departure.routeNumber,
      destination: departure.destination,
      status: status,
      statusText: departure.remarks,
      active: isActive
    };
  }

  private getStatusClass(remarks: string): string {
    if (!remarks) return '';
    
    const lowerRemarks = remarks.toLowerCase();
    if (lowerRemarks.includes('departed')) return 'departed';
    if (lowerRemarks.includes('check in') || lowerRemarks.includes('boarding')) return 'checkin';
    if (lowerRemarks.includes('delay')) return 'delay';
    if (lowerRemarks.includes('on time')) return 'ontime';
    
    return '';
  }

  get filteredLounges() {
    if (!this.searchTerm.trim()) {
      return this.lounges;
    }
    const term = this.searchTerm.toLowerCase();
    return this.lounges.map(lounge => ({
      ...lounge,
      departures: lounge.departures.filter((dep: any) => 
        lounge.name.toLowerCase().includes(term) ||
        dep.time.toLowerCase().includes(term) ||
        dep.busNo.toLowerCase().includes(term) ||
        dep.routeNumber.toLowerCase().includes(term) ||
        dep.destination.toLowerCase().includes(term) ||
        dep.statusText.toLowerCase().includes(term)
      )
    })).filter(lounge => lounge.departures.length > 0);
  }
}
