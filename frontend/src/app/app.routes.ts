import { Routes } from '@angular/router';
import { HomeComponent } from './features/home/home.component';
import { DashboardComponent } from './features/dashboard/dashboard.component';
import { DeparturesComponent } from './features/departures/departures.component';
import { ArrivalComponent } from './features/Arrivals/Arrival.component';
import { AdvertisementsComponent } from './features/advertisements/advertisements.component';
import { BusScheduleComponent } from './features/bus-schedule/bus-schedule.component';
import { BidsDisplayComponent } from './features/bids-display/bids-display.component';
import { LoungeComponent } from './features/lounge/lounge.component';
import { LoginComponent } from './features/auth/login.component';
import { LogoutComponent } from './features/auth/logout.component';
import { RouteManagementComponent } from './features/route-management/route-management.component';
import { RouteEditorComponent } from './features/route-management/route-editor.component';
import { TVSyncAgentComponent } from './features/tv-sync-agent/tv-sync-agent.component';
import { AdCostReportComponent } from './features/ad-cost-report/ad-cost-report.component';
import { authGuard } from './core/guards/auth.guard';

export const routes: Routes = [
	{ path: '', pathMatch: 'full', component: HomeComponent },
	{ path: 'home', component: HomeComponent },
	{ path: 'dashboard', component: DashboardComponent, canActivate: [authGuard] },
	{ path: 'departures', component: DeparturesComponent, canActivate: [authGuard] },
	{ path: 'arrivals', component: ArrivalComponent, canActivate: [authGuard] },
	{ path: 'bus-schedule', component: BusScheduleComponent, canActivate: [authGuard] },
	{ path: 'bids-display', component: BidsDisplayComponent },
	{ path: 'advertisements', component: AdvertisementsComponent, canActivate: [authGuard] },
	{ path: 'lounge', component: LoungeComponent, canActivate: [authGuard] },
	{ path: 'tv-sync-agent', component: TVSyncAgentComponent, canActivate: [authGuard] },
	{ path: 'ad-cost-report', component: AdCostReportComponent, canActivate: [authGuard] },
	{ path: 'route-management', component: RouteManagementComponent, canActivate: [authGuard] },
	{ path: 'route-management/editor', component: RouteEditorComponent, canActivate: [authGuard] },
	{ path: 'route-management/editor/:id', component: RouteEditorComponent, canActivate: [authGuard] },
	{ path: 'login', component: LoginComponent },
	{ path: 'logout', component: LogoutComponent },
	{ path: '**', redirectTo: '' }
];
