import { motion } from "motion/react";

interface SeatGridProps {
  totalSeats: number;
  bookedSeats: Set<number>;
  lockedSeats: Set<number>;
}

export function SeatGrid({ totalSeats, bookedSeats, lockedSeats }: SeatGridProps) {
  const rows = 20;
  const seatsPerRow = 50;

  return (
    <div className="bg-slate-900/50 backdrop-blur-sm border border-slate-700/50 rounded-xl p-6">
      <div className="mb-6">
        <h3 className="text-lg font-semibold text-white mb-4">Seat Availability</h3>

        {/* Legend */}
        <div className="flex flex-wrap gap-4 text-sm">
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-green-500 rounded" />
            <span className="text-gray-300">Available</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-yellow-500 rounded" />
            <span className="text-gray-300">Locked</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-red-500 rounded" />
            <span className="text-gray-300">Booked</span>
          </div>
        </div>
      </div>

      {/* Seat Grid */}
      <div className="overflow-x-auto">
        <div className="inline-block min-w-full">
          <div className="space-y-2">
            {Array.from({ length: rows }, (_, rowIndex) => (
              <div key={rowIndex} className="flex gap-1.5">
                {/* Row label */}
                <div className="w-8 flex items-center justify-center text-gray-500 text-xs">
                  {String.fromCharCode(65 + rowIndex)}
                </div>

                {/* Seats */}
                {Array.from({ length: seatsPerRow }, (_, seatIndex) => {
                  const seatNumber = rowIndex * seatsPerRow + seatIndex;
                  const isBooked = bookedSeats.has(seatNumber);
                  const isLocked = lockedSeats.has(seatNumber);

                  let seatColor = "bg-green-500";
                  if (isBooked) seatColor = "bg-red-500";
                  else if (isLocked) seatColor = "bg-yellow-500";

                  return (
                    <motion.div
                      key={seatNumber}
                      className={`w-3 h-3 rounded-sm ${seatColor} transition-colors`}
                      initial={{ scale: 1 }}
                      animate={isBooked || isLocked ? { scale: [1, 1.2, 1] } : {}}
                      transition={{ duration: 0.3 }}
                    />
                  );
                })}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Stats below grid */}
      <div className="mt-6 grid grid-cols-3 gap-4">
        <div className="text-center">
          <div className="text-2xl font-bold text-green-400">
            {totalSeats - bookedSeats.size}
          </div>
          <div className="text-xs text-gray-400">Available</div>
        </div>
        <div className="text-center">
          <div className="text-2xl font-bold text-yellow-400">
            {lockedSeats.size}
          </div>
          <div className="text-xs text-gray-400">Locked</div>
        </div>
        <div className="text-center">
          <div className="text-2xl font-bold text-red-400">
            {bookedSeats.size}
          </div>
          <div className="text-xs text-gray-400">Booked</div>
        </div>
      </div>
    </div>
  );
}
