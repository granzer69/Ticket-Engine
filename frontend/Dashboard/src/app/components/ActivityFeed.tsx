import { motion, AnimatePresence } from "motion/react";
import { CheckCircle2, Clock, XCircle } from "lucide-react";
import { Activity } from "./types";

interface ActivityFeedProps {
  activities: Activity[];
}

export function ActivityFeed({ activities }: ActivityFeedProps) {
  return (
    <div className="h-[600px] bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl p-6">
      <h3 className="text-lg font-semibold text-white mb-4">Live Activity Feed</h3>

      <div className="h-[calc(100%-3rem)] overflow-hidden relative">
        {/* Fade overlay at top */}
        <div className="absolute top-0 left-0 right-0 h-8 bg-gradient-to-b from-slate-900/80 to-transparent z-10 pointer-events-none" />

        <div className="h-full overflow-y-auto space-y-2 pr-2 scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent">
          <AnimatePresence initial={false}>
            {activities.map((activity) => (
              <motion.div
                key={activity.id}
                initial={{ opacity: 0, x: -20, height: 0 }}
                animate={{ opacity: 1, x: 0, height: "auto" }}
                exit={{ opacity: 0, height: 0 }}
                transition={{ duration: 0.3 }}
                className="flex items-start gap-3 p-3 bg-slate-800/50 rounded-lg border border-slate-700/30"
              >
                <div className="mt-0.5">
                  {activity.action === "booked" && (
                    <CheckCircle2 className="w-4 h-4 text-green-400" />
                  )}
                  {activity.action === "queued" && (
                    <Clock className="w-4 h-4 text-yellow-400" />
                  )}
                  {activity.action === "failed" && (
                    <XCircle className="w-4 h-4 text-red-400" />
                  )}
                </div>

                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-300">
                    <span className="font-semibold text-purple-400">{activity.user}</span>
                    {activity.action === "booked" && (
                      <>
                        {" "}
                        booked{" "}
                        <span className="font-semibold text-green-400">Seat {activity.seat}</span>
                      </>
                    )}
                    {activity.action === "queued" && " added to queue"}
                    {activity.action === "failed" && " request failed"}
                  </p>
                  <p className="text-xs text-gray-500 mt-0.5">
                    {activity.timestamp.toLocaleTimeString()}
                  </p>
                </div>
              </motion.div>
            ))}
          </AnimatePresence>
        </div>

        {/* Fade overlay at bottom */}
        <div className="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-slate-900/80 to-transparent pointer-events-none" />
      </div>
    </div>
  );
}
