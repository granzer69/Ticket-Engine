import { useState } from "react";
import { MapPin, Users, Calendar } from "lucide-react";

interface Venue {
  id: string;
  name: string;
  location: string;
  capacity: string;
  mapQuery: string;
}

const venues: Venue[] = [
  {
    id: "stadium1",
    name: "Eden Gardens",
    location: "Kolkata, India",
    capacity: "66,000",
    mapQuery: "Eden+Gardens,Kolkata",
  },
  {
    id: "stadium2",
    name: "Wankhede Stadium",
    location: "Mumbai, India",
    capacity: "33,000",
    mapQuery: "Wankhede+Stadium,Mumbai",
  },
  {
    id: "stadium3",
    name: "O2 Arena",
    location: "London, UK",
    capacity: "20,000",
    mapQuery: "O2+Arena,London,UK",
  },
];

export function VenueInfo() {
  const [selectedVenue, setSelectedVenue] = useState<Venue>(venues[0]);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Venue Information</h2>
        <p className="text-gray-600">Select a venue to view details and location</p>
      </div>

      {/* Venue Selector */}
      <div className="bg-white p-6 rounded-lg shadow">
        <label className="block text-sm font-medium text-gray-700 mb-3">
          Select Venue
        </label>
        <select
          value={selectedVenue.id}
          onChange={(e) => {
            const venue = venues.find((v) => v.id === e.target.value);
            if (venue) setSelectedVenue(venue);
          }}
          className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
          {venues.map((venue) => (
            <option key={venue.id} value={venue.id}>
              {venue.name} - {venue.location}
            </option>
          ))}
        </select>
      </div>

      {/* Venue Details */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-blue-50 p-4 rounded-lg border border-blue-200">
          <div className="flex items-center gap-2 text-blue-700 mb-2">
            <MapPin className="w-5 h-5" />
            <span className="text-sm font-medium">Location</span>
          </div>
          <div className="font-semibold text-blue-900">{selectedVenue.location}</div>
        </div>
        <div className="bg-purple-50 p-4 rounded-lg border border-purple-200">
          <div className="flex items-center gap-2 text-purple-700 mb-2">
            <Users className="w-5 h-5" />
            <span className="text-sm font-medium">Capacity</span>
          </div>
          <div className="font-semibold text-purple-900">{selectedVenue.capacity}</div>
        </div>
        <div className="bg-green-50 p-4 rounded-lg border border-green-200">
          <div className="flex items-center gap-2 text-green-700 mb-2">
            <Calendar className="w-5 h-5" />
            <span className="text-sm font-medium">Next Event</span>
          </div>
          <div className="font-semibold text-green-900">April 15, 2026</div>
        </div>
      </div>

      {/* Map */}
      <div className="bg-white p-6 rounded-lg shadow">
        <h3 className="font-semibold text-gray-900 mb-4">Venue Location</h3>
        <div className="w-full h-96 rounded-lg overflow-hidden border border-gray-200">
          <iframe
            src={`https://www.google.com/maps?q=${selectedVenue.mapQuery}&output=embed`}
            width="100%"
            height="100%"
            style={{ border: 0 }}
            allowFullScreen
            loading="lazy"
            title={`Map of ${selectedVenue.name}`}
          />
        </div>
      </div>
    </div>
  );
}