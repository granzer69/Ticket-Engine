import { motion } from "motion/react";
import { Ticket, Users, Activity, TrendingUp } from "lucide-react";

interface StatsPanelProps {
  totalTickets: number;
  ticketsBooked: number;
  activeUsers: number;
  requestsPerSecond: number;
}

export function StatsPanel({ totalTickets, ticketsBooked, activeUsers, requestsPerSecond }: StatsPanelProps) {
  const availableTickets = totalTickets - ticketsBooked;
  const bookingPercentage = (ticketsBooked / totalTickets) * 100;

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-white mb-4">System Stats</h3>

      {/* Available Tickets */}
      <motion.div
        className="p-5 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl"
        whileHover={{ scale: 1.02 }}
        transition={{ type: "spring", stiffness: 300 }}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Ticket className="w-5 h-5 text-green-400" />
            <span className="text-gray-300 text-sm">Available</span>
          </div>
        </div>
        <div className="text-3xl font-bold text-green-400">{availableTickets.toLocaleString()}</div>
        <div className="text-xs text-gray-500 mt-1">of {totalTickets.toLocaleString()} total</div>
      </motion.div>

      {/* Tickets Booked */}
      <motion.div
        className="p-5 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl"
        whileHover={{ scale: 1.02 }}
        transition={{ type: "spring", stiffness: 300 }}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Ticket className="w-5 h-5 text-purple-400" />
            <span className="text-gray-300 text-sm">Booked</span>
          </div>
        </div>
        <div className="text-3xl font-bold text-purple-400">{ticketsBooked.toLocaleString()}</div>
        <div className="mt-3 bg-slate-800 rounded-full h-2 overflow-hidden">
          <motion.div
            className="h-full bg-gradient-to-r from-purple-500 to-pink-500"
            initial={{ width: 0 }}
            animate={{ width: `${bookingPercentage}%` }}
            transition={{ duration: 0.5 }}
          />
        </div>
        <div className="text-xs text-gray-500 mt-1">{bookingPercentage.toFixed(1)}% booked</div>
      </motion.div>

      {/* Active Users */}
      <motion.div
        className="p-5 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl"
        whileHover={{ scale: 1.02 }}
        transition={{ type: "spring", stiffness: 300 }}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Users className="w-5 h-5 text-blue-400" />
            <span className="text-gray-300 text-sm">Active Users</span>
          </div>
        </div>
        <div className="text-3xl font-bold text-blue-400">{activeUsers.toLocaleString()}</div>
        <div className="text-xs text-gray-500 mt-1">concurrent connections</div>
      </motion.div>

      {/* Requests Per Second */}
      <motion.div
        className="p-5 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl"
        whileHover={{ scale: 1.02 }}
        transition={{ type: "spring", stiffness: 300 }}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-yellow-400" />
            <span className="text-gray-300 text-sm">RPS</span>
          </div>
        </div>
        <div className="text-3xl font-bold text-yellow-400">{requestsPerSecond.toLocaleString()}</div>
        <div className="text-xs text-gray-500 mt-1">requests/second</div>
      </motion.div>
    </div>
  );
}
