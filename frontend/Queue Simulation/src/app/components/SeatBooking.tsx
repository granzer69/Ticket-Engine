import { useState } from "react";
import { Check, X } from "lucide-react";

interface Seat {
  id: number;
  status: "available" | "booked";
}

export function SeatBooking() {
  const [seats, setSeats] = useState<Seat[]>(() => {
    return Array.from({ length: 100 }, (_, i) => ({
      id: i + 1,
      status: "available" as const,
    }));
  });

  const bookSeat = (id: number) => {
    setSeats((prevSeats) =>
      prevSeats.map((seat) =>
        seat.id === id && seat.status === "available"
          ? { ...seat, status: "booked" }
          : seat
      )
    );
  };

  const bookedCount = seats.filter((s) => s.status === "booked").length;
  const availableCount = seats.filter((s) => s.status === "available").length;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Seat Booking</h2>
        <p className="text-gray-600">Click on any available seat to book it</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-green-50 p-4 rounded-lg border border-green-200">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 bg-green-500 rounded"></div>
            <span className="text-sm text-green-700 font-medium">Available</span>
          </div>
          <div className="text-2xl font-bold text-green-900 mt-2">{availableCount}</div>
        </div>
        <div className="bg-red-50 p-4 rounded-lg border border-red-200">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 bg-red-500 rounded"></div>
            <span className="text-sm text-red-700 font-medium">Booked</span>
          </div>
          <div className="text-2xl font-bold text-red-900 mt-2">{bookedCount}</div>
        </div>
      </div>

      {/* Seat Grid */}
      <div className="bg-white p-6 rounded-lg shadow">
        <div className="mb-6 text-center">
          <div className="inline-block bg-gray-200 px-8 py-3 rounded">
            <span className="text-sm font-medium text-gray-700">STAGE</span>
          </div>
        </div>

        <div className="grid grid-cols-10 gap-2">
          {seats.map((seat) => (
            <button
              key={seat.id}
              onClick={() => bookSeat(seat.id)}
              disabled={seat.status === "booked"}
              className={`
                aspect-square rounded-md flex items-center justify-center text-sm font-medium
                transition-all
                ${
                  seat.status === "available"
                    ? "bg-green-500 text-white hover:bg-green-600 cursor-pointer"
                    : "bg-red-500 text-white cursor-not-allowed opacity-75"
                }
              `}
            >
              {seat.status === "booked" ? (
                <X className="w-4 h-4" />
              ) : (
                <span>{seat.id}</span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="bg-gray-50 p-4 rounded-lg border border-gray-200">
        <div className="flex flex-wrap gap-6">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-green-500 rounded flex items-center justify-center text-white text-xs">
              1
            </div>
            <span className="text-sm text-gray-700">Available - Click to book</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-red-500 rounded flex items-center justify-center text-white">
              <X className="w-4 h-4" />
            </div>
            <span className="text-sm text-gray-700">Already booked</span>
          </div>
        </div>
      </div>
    </div>
  );
}
