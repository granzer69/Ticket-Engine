import { motion } from "motion/react";
import { ListOrdered, Zap, Database } from "lucide-react";

interface QueuePanelProps {
  queueSize: number;
}

export function QueuePanel({ queueSize }: QueuePanelProps) {
  const processingSpeed = Math.floor(Math.random() * 50) + 150; // 150-200 ms
  const queueHealth = queueSize < 100 ? "healthy" : queueSize < 300 ? "moderate" : "busy";

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-white mb-4">Queue & Performance</h3>

      {/* Queue Size */}
      <motion.div
        className="p-5 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl"
        whileHover={{ scale: 1.02 }}
        transition={{ type: "spring", stiffness: 300 }}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <ListOrdered className="w-5 h-5 text-orange-400" />
            <span className="text-gray-300 text-sm">Queue Size</span>
          </div>
        </div>
        <div className="text-3xl font-bold text-orange-400">{queueSize}</div>
        <div className="flex items-center gap-2 mt-2">
          <div
            className={`px-2 py-1 rounded text-xs font-semibold ${
              queueHealth === "healthy"
                ? "bg-green-500/20 text-green-400"
                : queueHealth === "moderate"
                ? "bg-yellow-500/20 text-yellow-400"
                : "bg-red-500/20 text-red-400"
            }`}
          >
            {queueHealth}
          </div>
        </div>
      </motion.div>

      {/* Processing Speed */}
      <motion.div
        className="p-5 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl"
        whileHover={{ scale: 1.02 }}
        transition={{ type: "spring", stiffness: 300 }}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Zap className="w-5 h-5 text-yellow-400" />
            <span className="text-gray-300 text-sm">Avg. Processing</span>
          </div>
        </div>
        <div className="text-3xl font-bold text-yellow-400">{processingSpeed}ms</div>
        <div className="text-xs text-gray-500 mt-1">per request</div>
      </motion.div>

      {/* Redis Visualization */}
      <motion.div
        className="p-5 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl"
        whileHover={{ scale: 1.02 }}
        transition={{ type: "spring", stiffness: 300 }}
      >
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Database className="w-5 h-5 text-red-400" />
            <span className="text-gray-300 text-sm">Redis Queue</span>
          </div>
        </div>

        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((_, index) => (
            <motion.div
              key={index}
              className="h-3 bg-slate-800 rounded-full overflow-hidden"
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <motion.div
                className="h-full bg-gradient-to-r from-red-500 to-orange-500"
                animate={{
                  width: ["0%", "100%"],
                }}
                transition={{
                  duration: 2,
                  repeat: Infinity,
                  delay: index * 0.2,
                  ease: "easeInOut",
                }}
              />
            </motion.div>
          ))}
        </div>

        <p className="text-xs text-gray-500 mt-3 text-center">Queue processing visualization</p>
      </motion.div>

      {/* System Health */}
      <motion.div
        className="p-5 bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl"
        whileHover={{ scale: 1.02 }}
        transition={{ type: "spring", stiffness: 300 }}
      >
        <div className="space-y-3">
          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-400">CPU Usage</span>
              <span className="text-green-400">42%</span>
            </div>
            <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
              <div className="h-full w-[42%] bg-green-500 rounded-full" />
            </div>
          </div>

          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-400">Memory</span>
              <span className="text-blue-400">68%</span>
            </div>
            <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
              <div className="h-full w-[68%] bg-blue-500 rounded-full" />
            </div>
          </div>

          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-400">Network I/O</span>
              <span className="text-purple-400">55%</span>
            </div>
            <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
              <div className="h-full w-[55%] bg-purple-500 rounded-full" />
            </div>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
