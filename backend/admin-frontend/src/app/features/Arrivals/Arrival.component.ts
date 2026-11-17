import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HeaderComponent } from '../../shared/components/header/header.component';
import { ArrivalService } from '../../core/services';
import type { LoungeArrivalResponse, ArrivalInfo } from '../../core/services';

@Component({
  selector: 'app-arrivals',
  standalone: true,
  imports: [HeaderComponent, CommonModule, FormsModule],
  templateUrl: './Arrival.component.html',
  styleUrl: './Arrival.component.scss'
})
export class ArrivalComponent implements OnInit, OnDestroy {
  searchTerm: string = '';
  lounges: any[] = [];
  isLoading: boolean = false;
  error: string | null = null;
  private refreshInterval: any;
  private readonly REFRESH_INTERVAL_MS = 45000; // 45 seconds

  constructor(private arrivalService: ArrivalService) {}

  ngOnInit(): void {
    this.loadArrivals();
    
    // Set up auto-refresh every 45 seconds
    this.refreshInterval = setInterval(() => {
      this.loadArrivals(true); // Silent refresh (no loading indicator)
    }, this.REFRESH_INTERVAL_MS);
  }

  ngOnDestroy(): void {
    // Clean up interval on component destroy
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }
  }

  loadArrivals(silentRefresh: boolean = false): void {
    if (!silentRefresh) {
      this.isLoading = true;
    }
    this.error = null;
    
    this.arrivalService.getAllLoungeArrivals().subscribe({
      next: (loungeArrivals: LoungeArrivalResponse[]) => {
        // Map arrivals to the component's structure
        this.lounges = loungeArrivals.map(lounge => ({
          id: lounge.loungeId,
          name: lounge.loungeName,
          arrivals: lounge.arrivals.map(arrival => this.mapArrivalToDisplay(arrival))
        }));
        this.isLoading = false;
      },
      error: (err) => {
        console.error('Error loading arrivals:', err);
        if (!silentRefresh) {
          this.error = 'Failed to load arrivals. Please try again.';
          this.lounges = [];
        }
        this.isLoading = false;
      }
    });
  }

  private mapArrivalToDisplay(arrival: ArrivalInfo): any {
    const status = this.getStatusClass(arrival.remarks);
    const isActive = status === 'expected' || status === 'delayed' || arrival.remarks.includes('Arriving');
    
    return {
      time: arrival.time || '-',
      busNo: arrival.busNo,
      routeNumber: arrival.routeNumber,
      origin: arrival.origin,
      status: status,
      statusText: arrival.remarks,
      active: isActive
    };
  }

  private getStatusClass(remarks: string): string {
    if (!remarks) return '';
    
    const lowerRemarks = remarks.toLowerCase();
    if (lowerRemarks.includes('arrived')) return 'arrived';
    if (lowerRemarks.includes('delayed')) return 'delayed';
    if (lowerRemarks.includes('expected') || lowerRemarks.includes('arriving')) return 'expected';
    
    return '';
  }

  get filteredLounges() {
    if (!this.searchTerm.trim()) {
      return this.lounges;
    }
    const term = this.searchTerm.toLowerCase();
    return this.lounges.map(lounge => ({
      ...lounge,
      arrivals: lounge.arrivals.filter((arr: any) => 
        lounge.name.toLowerCase().includes(term) ||
        arr.time.toLowerCase().includes(term) ||
        arr.busNo.toLowerCase().includes(term) ||
        arr.routeNumber.toLowerCase().includes(term) ||
        arr.origin.toLowerCase().includes(term) ||
        arr.statusText.toLowerCase().includes(term)
      )
    })).filter(lounge => lounge.arrivals.length > 0);
  }

  // Get current date for display
  getCurrentDate(): string {
    return new Date().toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric', 
      month: 'long',
      day: 'numeric'
    });
  }
}