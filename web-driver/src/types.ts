export interface Trip {
  trip: Trip;
  id: string;
  userID: string;
  status: string;
  selectedFare: RouteFare;
  route: Route;
  driver?: Driver;
}

export interface Coordinate {
  latitude: number;
  longitude: number;
}

export interface Route {
  geometry: {
    coordinates: Coordinate[];
  }[];
  duration: number;
  distance: number;
}

export enum CarPackageSlug {
  SEDAN = "sedan",
  SUV = "suv",
  VAN = "van",
  LUXURY = "luxury",
}

export interface RouteFare {
  id: string;
  packageSlug: CarPackageSlug;
  basePrice: number;
  totalPriceInCents?: number;
  expiresAt: Date;
  route: Route;
}

export interface Driver {
  id: string;
  location: Coordinate;
  geohash: string;
  name: string;
  profilePicture: string;
  carPlate: string;
  packageSlug?: CarPackageSlug;
}

export interface DriverProfile {
  id: string;
  name: string;
  email: string;
  phone: string;
  packageSlug: CarPackageSlug;
  carPlate: string;
  profilePicture?: string;
  bonusPoints?: number;
}

export interface DriverTripSummary {
  id: string;
  userID: string;
  status: string;
  fare?: number;
  packageSlug?: string;
  currency?: string;
  completedAt?: string;
  driver?: { id: string; name: string };
}

export interface DriverReview {
  id: string;
  tripID: string;
  userID: string;
  driverID: string;
  rating: number;
  comment?: string;
  bonusPoints: number;
  createdAt: string;
}

export interface DriverDashboard {
  tripCount: number;
  bonusPoints: number;
  averageRating: number;
  recentTrips: DriverTripSummary[];
  recentReviews: DriverReview[];
}

export interface RegisterDriverPayload {
  name: string;
  email: string;
  phone: string;
  password: string;
  packageSlug: CarPackageSlug;
  carPlate: string;
}

export interface LoginDriverPayload {
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  driver: DriverProfile;
}
