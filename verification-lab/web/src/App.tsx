import { StatusBar } from "./components/StatusBar";
import { Pipeline } from "./components/Pipeline";
import { Inspectors } from "./components/Inspectors";
import { Charts } from "./components/Charts";
import { Timeline } from "./components/Timeline";
import { RunControl } from "./components/RunControl";
import { ClaimsPanel } from "./components/ClaimsPanel";
import { FailureLabPanel } from "./components/FailureLabPanel";
import { useTelemetrySSE } from "./hooks/useTelemetrySSE";

export function App() {
  useTelemetrySSE();

  return (
    <div className="console">
      <StatusBar />
      <div className="console-main">
        <div className="col-left">
          <Pipeline />
          <Inspectors />
          <Charts />
        </div>
        <div className="col-right">
          <RunControl />
          <Timeline />
          <ClaimsPanel />
          <FailureLabPanel />
        </div>
      </div>
    </div>
  );
}
