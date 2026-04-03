import { useMemo, useState } from "react";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from "recharts";

export function QueueSimulation({ onBooked }: { onBooked: () => void }) {
  const [status, setStatus] = useState<'idle' | 'booking' | 'success' | 'error'>('idle');
  const [message, setMessage] = useState('');
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

  const handleJoinQueue = async () => {
    setStatus('booking');
    setMessage('Joining queue...');
    const username = localStorage.getItem('username') || 'DummyUser'; 
    let userId = 0;
    for (let i = 0; i < username.length; i++) {
        userId += username.charCodeAt(i);
    }
    userId = userId + Math.floor(Math.random() * 1000);

    try {
      const response = await fetch('http://localhost:8080/book', {
        method: 'POST',
        headers: {
          'X-User-Id': userId.toString(),
        }
      });
      const data = await response.json();
      if (response.ok && data.status === 'success') {
         setStatus('success');
         setMessage('Booking successful! Redirecting...');
         setTimeout(() => {
           onBooked();
         }, 1500);
      } else {
         setStatus('error');
         setMessage(data.message || 'Booking failed');
      }
    } catch (e: any) {
      setStatus('error');
      setMessage(e.message || 'Error connecting to server');
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Queue Simulation</h2>
        <div className="flex justify-between items-center w-full">
          <p className="text-gray-600">
            Real-time visualization of people waiting in the booking queue
          </p>
          <button 
             onClick={handleJoinQueue} 
             disabled={status === 'booking' || status === 'success'}
             className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white font-semibold rounded-lg shadow disabled:opacity-50"
          >
             {status === 'booking' ? 'Processing...' : 'Book Ticket'}
          </button>
        </div>
        {message && (
          <div className={`mt-4 p-3 rounded-md ${status === 'success' ? 'bg-green-100 text-green-800' : status === 'error' ? 'bg-red-100 text-red-800' : 'bg-blue-100 text-blue-800'}`}>
             {message}
          </div>
        )}
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
