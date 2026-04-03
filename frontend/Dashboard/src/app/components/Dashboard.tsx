import { useState, useEffect } from "react";
import { StatsPanel } from "./StatsPanel";
import { ActivityFeed } from "./ActivityFeed";
import { QueuePanel } from "./QueuePanel";
import { SeatGrid } from "./SeatGrid";
import { ArchitectureDiagram } from "./ArchitectureDiagram";
import { InsightsFooter } from "./InsightsFooter";
import { Activity } from "./types";

export function Dashboard() {
  const [totalTickets] = useState(15000);
  const [ticketsBooked, setTicketsBooked] = useState(0);
  const [activeUsers, setActiveUsers] = useState(0);
  const [requestsPerSecond, setRequestsPerSecond] = useState(0);
  const [queueSize, setQueueSize] = useState(0);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [bookedSeats, setBookedSeats] = useState<Set<number>>(new Set());
  const [lockedSeats, setLockedSeats] = useState<Set<number>>(new Set());

  useEffect(() => {
    // Simulate initial burst
    setActiveUsers(Math.floor(Math.random() * 1000) + 500);
    setQueueSize(Math.floor(Math.random() * 200) + 50);

    // Fetch real ticket counts and metrics from backend
    const fetchTicketsInterval = setInterval(async () => {
      try {
        const response = await fetch('http://localhost:8080/tickets/count');
        const data = await response.json();
        if (response.ok && data.remaining !== undefined) {
           // We cap tickets booked to avoid overflowing negative if totalTickets changes
           setTicketsBooked(Math.max(0, totalTickets - data.remaining));
        }
      } catch (e) {
        console.error("Failed to fetch tickets:", e);
      }
      
      try {
        const mResp = await fetch('http://localhost:8080/metrics');
        const mData = await mResp.json();
        if (mResp.ok) {
           // Use total requests as an indicator for requests per second to show activity
           setRequestsPerSecond(mData.total_requests);
        }
      } catch (e) {
        console.error("Failed to fetch metrics:", e);
      }
    }, 1000);

    // Simulate locked seats
    const lockInterval = setInterval(() => {
      setLockedSeats((prev) => {
        const newLocked = new Set<number>();
        const availableSeats = Array.from({ length: totalTickets }, (_, i) => i).filter(
          (seat) => !bookedSeats.has(seat)
        );
        // Lock 2-5 random seats
        const numToLock = Math.floor(Math.random() * 4) + 2;
        for (let i = 0; i < numToLock && availableSeats.length > 0; i++) {
          const randomIndex = Math.floor(Math.random() * availableSeats.length);
          newLocked.add(availableSeats[randomIndex]);
          availableSeats.splice(randomIndex, 1);
        }
        return newLocked;
      });
    }, 1500);

    // Simulate active users fluctuation
    const usersInterval = setInterval(() => {
      setActiveUsers((prev) => {
        const change = Math.floor(Math.random() * 200) - 100;
        return Math.max(100, Math.min(2000, prev + change));
      });
    }, 2000);

    // Removed Simulated RPS since we pull it from real metrics now

    // Simulate queue size
    const queueInterval = setInterval(() => {
      setQueueSize((prev) => {
        const change = Math.floor(Math.random() * 40) - 20;
        return Math.max(0, Math.min(500, prev + change));
      });
    }, 1500);

    // Add some queue activities
    const queueActivityInterval = setInterval(() => {
      if (Math.random() > 0.5) {
        const userId = `User${Math.floor(Math.random() * 9999) + 1000}`;
        setActivities((prev) => [
          {
            id: Date.now() + Math.random(),
            user: userId,
            action: "queued",
            timestamp: new Date(),
          },
          ...prev.slice(0, 49),
        ]);
      }
    }, 2000);

    return () => {
      clearInterval(fetchTicketsInterval);
      clearInterval(lockInterval);
      clearInterval(usersInterval);
      clearInterval(queueInterval);
      clearInterval(queueActivityInterval);
    };
  }, [totalTickets, bookedSeats]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950">
      {/* Header */}
      <div className="border-b border-slate-800 bg-slate-900/50 backdrop-blur-sm">
        <div className="max-w-[1920px] mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-2xl font-bold text-white">Live Dashboard</h2>
              <p className="text-gray-400 text-sm">Real-time monitoring</p>
            </div>
            <div className="flex items-center gap-2 px-4 py-2 bg-green-500/20 border border-green-500/30 rounded-full">
              <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse" />
              <span className="text-green-400 text-sm">Live</span>
            </div>
          </div>
        </div>
      </div>

      {/* Main Dashboard */}
      <div className="max-w-[1920px] mx-auto px-6 py-8">
        {/* Three Column Layout */}
        <div className="grid grid-cols-1 xl:grid-cols-12 gap-6 mb-8">
          <div className="xl:col-span-3">
            <StatsPanel
              totalTickets={totalTickets}
              ticketsBooked={ticketsBooked}
              activeUsers={activeUsers}
              requestsPerSecond={requestsPerSecond}
            />
          </div>

          <div className="xl:col-span-6">
            <ActivityFeed activities={activities} />
          </div>

          <div className="xl:col-span-3">
            <QueuePanel queueSize={queueSize} />
          </div>
        </div>

        {/* Seat Grid */}
        <div className="mb-8">
          <SeatGrid
            totalSeats={Math.min(1000, totalTickets)} // Capped rendering to avoid browser freeze
            bookedSeats={bookedSeats}
            lockedSeats={lockedSeats}
          />
        </div>

        {/* Architecture Diagram */}
        <div className="mb-8">
          <ArchitectureDiagram />
        </div>

        {/* Insights Footer */}
        <InsightsFooter ticketsBooked={ticketsBooked} totalTickets={totalTickets} />
      </div>
    </div>
  );
}
