import { motion } from "motion/react";
import { Zap, Users, Shield } from "lucide-react";

interface LandingPageProps {
  onStartSimulation: () => void;
}

export function LandingPage({ onStartSimulation }: LandingPageProps) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-purple-950 to-slate-900 flex items-center justify-center relative overflow-hidden">
      {/* Animated background elements */}
      <div className="absolute inset-0 overflow-hidden">
        <motion.div
          className="absolute top-20 left-20 w-72 h-72 bg-purple-500/20 rounded-full blur-3xl"
          animate={{
            scale: [1, 1.2, 1],
            opacity: [0.3, 0.5, 0.3],
          }}
          transition={{
            duration: 4,
            repeat: Infinity,
            ease: "easeInOut",
          }}
        />
        <motion.div
          className="absolute bottom-20 right-20 w-96 h-96 bg-blue-500/20 rounded-full blur-3xl"
          animate={{
            scale: [1, 1.3, 1],
            opacity: [0.3, 0.5, 0.3],
          }}
          transition={{
            duration: 5,
            repeat: Infinity,
            ease: "easeInOut",
          }}
        />
      </div>

      <div className="relative z-10 text-center px-8 max-w-5xl">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8 }}
        >
          <div className="mb-6 inline-flex items-center gap-2 px-4 py-2 bg-purple-500/20 border border-purple-500/30 rounded-full">
            <Zap className="w-4 h-4 text-purple-400" />
            <span className="text-purple-300 text-sm">High-Performance System</span>
          </div>

          <h1 className="text-6xl md:text-7xl font-bold mb-6 bg-gradient-to-r from-purple-400 via-pink-400 to-blue-400 bg-clip-text text-transparent">
            Real-Time Ticket Booking Engine
          </h1>

          <p className="text-xl md:text-2xl text-gray-300 mb-12 max-w-3xl mx-auto">
            Handles <span className="text-purple-400 font-semibold">100,000+ concurrent users</span> with{" "}
            <span className="text-green-400 font-semibold">zero double booking</span>
          </p>

          <motion.button
            onClick={onStartSimulation}
            className="px-8 py-4 bg-gradient-to-r from-purple-600 to-pink-600 hover:from-purple-500 hover:to-pink-500 text-white rounded-xl font-semibold text-lg shadow-lg shadow-purple-500/50 transition-all"
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
          >
            Start Simulation
          </motion.button>
        </motion.div>

        {/* Feature highlights */}
        <motion.div
          className="mt-20 grid grid-cols-1 md:grid-cols-3 gap-6"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.3 }}
        >
          <div className="p-6 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl">
            <Zap className="w-8 h-8 text-yellow-400 mb-3 mx-auto" />
            <h3 className="text-lg font-semibold text-white mb-2">Lightning Fast</h3>
            <p className="text-gray-400 text-sm">Sub-millisecond response times</p>
          </div>

          <div className="p-6 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl">
            <Users className="w-8 h-8 text-purple-400 mb-3 mx-auto" />
            <h3 className="text-lg font-semibold text-white mb-2">High Concurrency</h3>
            <p className="text-gray-400 text-sm">100K+ simultaneous requests</p>
          </div>

          <div className="p-6 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl">
            <Shield className="w-8 h-8 text-green-400 mb-3 mx-auto" />
            <h3 className="text-lg font-semibold text-white mb-2">Zero Conflicts</h3>
            <p className="text-gray-400 text-sm">Guaranteed data consistency</p>
          </div>
        </motion.div>
      </div>
    </div>
  );
}
