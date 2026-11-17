import { Routes } from '@angular/router';
import { BidsDisplayComponent } from './features/bids-display/bids-display.component';

// The Electron shell loads /bids-display?loungeId=...&kiosk=true
export const routes: Routes = [
	{ path: 'bids-display', component: BidsDisplayComponent },
	{ path: '**', redirectTo: 'bids-display' }
];
