export interface Schedule {
  id: string;
  route: string;
  departureTime: string;
  arrivalTime: string;
  status: 'on-time' | 'delayed' | 'cancelled';
}

export interface User {
  id: string;
  username: string;
  email: string;
  role: 'admin' | 'user';
}

export interface Lounge {
  id: string;
  loungeOwnerId?: string;
  loungeName: string;
  description?: string;
  address?: string;
  district?: string;
  state?: string;
  country?: string;
  postalCode?: string;
  latitude?: number;
  longitude?: number;
  contactPhone?: string;
  price1Hour?: number;
  price2Hours?: number;
  price3Hours?: number;
  priceUntilBus?: number;
  amenities?: any;
  images?: any;
  status?: string;
  isOperational?: boolean;
  averageRating?: number;
  capacity?: number;
  ownerId?: string;
  marketplaceCategoryId?: string;
  verificationNote?: string;
  totalStaff?: number;
  createdAt?: string;
  updatedAt?: string;
}

export * from './tv-sync-agent.model';
