export interface Activity {
  id: number;
  user: string;
  action: "booked" | "queued" | "failed";
  seat?: string;
  timestamp: Date;
}
