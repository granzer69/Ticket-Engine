import { useMemo } from "react";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from "recharts";

export function QueueSimulation() {
  const queueData = useMemo(() => {
    const data = [];
    const totalPeople = 100000;
    let booked = 0;

    for (let i = 0; i <= 10; i++) {
      const remaining = totalPeople - booked;
      const bookingNow = Math.floor(Math.random() * 10000);
      booked += bookingNow;
      data.push({
        time: `T+${i}`,
        waiting: Math.max(remaining - bookingNow, 0),
      });
    }

    return data;
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Queue Simulation</h2>
        <p className="text-gray-600">
          Real-time visualization of people waiting in the booking queue
        </p>
      </div>

      <div className="bg-white p-6 rounded-lg shadow">
        <ResponsiveContainer width="100%" height={400}>
          <BarChart data={queueData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Bar dataKey="waiting" fill="#3b82f6" name="People Waiting" />
          </BarChart>
        </ResponsiveContainer>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-blue-50 p-4 rounded-lg border border-blue-200">
          <div className="text-sm text-blue-600 font-medium">Total Capacity</div>
          <div className="text-2xl font-bold text-blue-900 mt-1">100,000</div>
        </div>
        <div className="bg-green-50 p-4 rounded-lg border border-green-200">
          <div className="text-sm text-green-600 font-medium">Current Queue</div>
          <div className="text-2xl font-bold text-green-900 mt-1">
            {queueData[queueData.length - 1]?.waiting.toLocaleString()}
          </div>
        </div>
        <div className="bg-purple-50 p-4 rounded-lg border border-purple-200">
          <div className="text-sm text-purple-600 font-medium">Time Intervals</div>
          <div className="text-2xl font-bold text-purple-900 mt-1">11</div>
        </div>
      </div>
    </div>
  );
}
