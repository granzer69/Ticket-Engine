import { motion } from "motion/react";
import { Users, Server, Database, Layers, ArrowRight } from "lucide-react";

export function ArchitectureDiagram() {
  return (
    <div className="bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl p-6">
      <h3 className="text-lg font-semibold text-white mb-6">System Architecture</h3>

      <div className="flex items-center justify-between flex-wrap gap-6">
        {/* Users */}
        <motion.div
          className="flex flex-col items-center"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.1 }}
        >
          <div className="w-20 h-20 rounded-xl bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center mb-3">
            <Users className="w-10 h-10 text-white" />
          </div>
          <p className="text-sm font-semibold text-white">Users</p>
          <p className="text-xs text-gray-400">100K+ concurrent</p>
        </motion.div>

        {/* Arrow */}
        <motion.div
          initial={{ opacity: 0, scale: 0.5 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.2 }}
        >
          <ArrowRight className="w-8 h-8 text-purple-400" />
        </motion.div>

        {/* API Gateway */}
        <motion.div
          className="flex flex-col items-center"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.3 }}
        >
          <div className="w-20 h-20 rounded-xl bg-gradient-to-br from-purple-500 to-purple-600 flex items-center justify-center mb-3">
            <Server className="w-10 h-10 text-white" />
          </div>
          <p className="text-sm font-semibold text-white">API Gateway</p>
          <p className="text-xs text-gray-400">Load balancing</p>
        </motion.div>

        {/* Arrow */}
        <motion.div
          initial={{ opacity: 0, scale: 0.5 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.4 }}
        >
          <ArrowRight className="w-8 h-8 text-purple-400" />
        </motion.div>

        {/* Redis Queue - Highlighted */}
        <motion.div
          className="flex flex-col items-center relative"
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.5 }}
        >
          <div className="absolute -inset-2 bg-gradient-to-r from-red-500/20 to-orange-500/20 rounded-xl blur-lg" />
          <div className="relative w-20 h-20 rounded-xl bg-gradient-to-br from-red-500 to-orange-500 flex items-center justify-center mb-3">
            <Layers className="w-10 h-10 text-white" />
          </div>
          <p className="text-sm font-semibold text-white">Redis Queue</p>
          <p className="text-xs text-red-400 font-semibold">Concurrency Engine</p>
        </motion.div>

        {/* Arrow */}
        <motion.div
          initial={{ opacity: 0, scale: 0.5 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.6 }}
        >
          <ArrowRight className="w-8 h-8 text-purple-400" />
        </motion.div>

        {/* Worker Pool */}
        <motion.div
          className="flex flex-col items-center"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.7 }}
        >
          <div className="w-20 h-20 rounded-xl bg-gradient-to-br from-green-500 to-green-600 flex items-center justify-center mb-3">
            <Layers className="w-10 h-10 text-white" />
          </div>
          <p className="text-sm font-semibold text-white">Worker Pool</p>
          <p className="text-xs text-gray-400">Processing</p>
        </motion.div>

        {/* Arrow */}
        <motion.div
          initial={{ opacity: 0, scale: 0.5 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.8 }}
        >
          <ArrowRight className="w-8 h-8 text-purple-400" />
        </motion.div>

        {/* Database */}
        <motion.div
          className="flex flex-col items-center"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.9 }}
        >
          <div className="w-20 h-20 rounded-xl bg-gradient-to-br from-cyan-500 to-cyan-600 flex items-center justify-center mb-3">
            <Database className="w-10 h-10 text-white" />
          </div>
          <p className="text-sm font-semibold text-white">MySQL DB</p>
          <p className="text-xs text-gray-400">Persistent storage</p>
        </motion.div>
      </div>

      {/* Key Features */}
      <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/30">
          <h4 className="text-sm font-semibold text-purple-400 mb-2">Queue-Based Processing</h4>
          <p className="text-xs text-gray-400">
            Redis ensures ordered, atomic operations preventing race conditions
          </p>
        </div>
        <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/30">
          <h4 className="text-sm font-semibold text-green-400 mb-2">Distributed Workers</h4>
          <p className="text-xs text-gray-400">
            Horizontally scalable worker pool handles high throughput
          </p>
        </div>
        <div className="p-4 bg-slate-800/50 rounded-lg border border-slate-700/30">
          <h4 className="text-sm font-semibold text-blue-400 mb-2">ACID Compliance</h4>
          <p className="text-xs text-gray-400">
            Database transactions guarantee data consistency
          </p>
        </div>
      </div>
    </div>
  );
}
