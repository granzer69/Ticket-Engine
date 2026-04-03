import { useState } from "react";
import { Login } from "./components/Login";
import { QueueSimulation } from "./components/QueueSimulation";
import { Dashboard } from "./components/Dashboard";

export default function App() {
  const [currentPage, setCurrentPage] = useState<'login' | 'queue' | 'dashboard'>('login');

  return (
    <div className="size-full">
      {currentPage === 'login' && (
        <Login onLogin={() => setCurrentPage('queue')} />
      )}
      {currentPage === 'queue' && (
        <QueueSimulation onBooked={() => setCurrentPage('dashboard')} />
      )}
      {currentPage === 'dashboard' && (
        <Dashboard />
      )}
    </div>
  );
}