import { motion } from "motion/react";
import { CheckCircle2, Zap, Shield } from "lucide-react";

interface InsightsFooterProps {
  ticketsBooked: number;
  totalTickets: number;
}

export function InsightsFooter({ ticketsBooked, totalTickets }: InsightsFooterProps) {
  return (
    <div className="bg-gradient-to-r from-slate-900/80 via-purple-900/50 to-slate-900/80 backdrop-blur-sm border border-slate-700/50 rounded-xl p-8">
      <div className="text-center mb-6">
        <h3 className="text-2xl font-bold text-white mb-2">System Performance Insights</h3>
        <p className="text-gray-400">Real-time metrics and achievements</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* No Double Booking */}
        <motion.div
          className="flex flex-col items-center text-center p-6 bg-slate-900/50 rounded-xl"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <div className="w-16 h-16 rounded-full bg-green-500/20 flex items-center justify-center mb-4">
            <CheckCircle2 className="w-8 h-8 text-green-400" />
          </div>
          <h4 className="text-xl font-bold text-green-400 mb-2">Zero Double Booking</h4>
          <p className="text-sm text-gray-400">
            100% consistency maintained across all transactions
          </p>
        </motion.div>

        {/* Requests Handled */}
        <motion.div
          className="flex flex-col items-center text-center p-6 bg-slate-900/50 rounded-xl"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          <div className="w-16 h-16 rounded-full bg-purple-500/20 flex items-center justify-center mb-4">
            <Zap className="w-8 h-8 text-purple-400" />
          </div>
          <h4 className="text-xl font-bold text-purple-400 mb-2">
            {ticketsBooked.toLocaleString()}+ Processed
          </h4>
          <p className="text-sm text-gray-400">
            Requests successfully handled with high throughput
          </p>
        </motion.div>

        {/* Technology Stack */}
        <motion.div
          className="flex flex-col items-center text-center p-6 bg-slate-900/50 rounded-xl"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
        >
          <div className="w-16 h-16 rounded-full bg-blue-500/20 flex items-center justify-center mb-4">
            <Shield className="w-8 h-8 text-blue-400" />
          </div>
          <h4 className="text-xl font-bold text-blue-400 mb-2">Powered by Redis + Go + MySQL</h4>
          <p className="text-sm text-gray-400">
            Battle-tested technologies for reliability
          </p>
        </motion.div>
      </div>

      {/* Tech badges */}
      <div className="mt-8 flex justify-center gap-4 flex-wrap">
        <div className="px-4 py-2 bg-slate-800/50 border border-slate-700/50 rounded-full text-sm text-gray-300">
          Redis Queue
        </div>
        <div className="px-4 py-2 bg-slate-800/50 border border-slate-700/50 rounded-full text-sm text-gray-300">
          Go Workers
        </div>
        <div className="px-4 py-2 bg-slate-800/50 border border-slate-700/50 rounded-full text-sm text-gray-300">
          MySQL Database
        </div>
        <div className="px-4 py-2 bg-slate-800/50 border border-slate-700/50 rounded-full text-sm text-gray-300">
          REST API
        </div>
      </div>
    </div>
  );
}
