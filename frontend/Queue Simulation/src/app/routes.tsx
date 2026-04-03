import { createBrowserRouter } from "react-router";
import { RootLayout } from "./components/RootLayout";
import { QueueSimulation } from "./components/QueueSimulation";
import { SeatBooking } from "./components/SeatBooking";
import { VenueInfo } from "./components/VenueInfo";
import { PaymentsInfo } from "./components/PaymentsInfo";

export const router = createBrowserRouter([
  {
    path: "/",
    Component: RootLayout,
    children: [
      { index: true, Component: QueueSimulation },
      { path: "queue", Component: QueueSimulation },
      { path: "seats", Component: SeatBooking },
      { path: "venue", Component: VenueInfo },
      { path: "payments", Component: PaymentsInfo },
    ],
  },
]);
