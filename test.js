import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const bookingSuccess = new Counter('booking_success');
const bookingFailed = new Counter('booking_failed');
const soldOut = new Counter('sold_out');
const errorRate = new Rate('error_rate');
const bookingDuration = new Trend('booking_duration');

export const options = {
  // Staged ramp-up: simulates 10K total unique users arriving over time.
  // On Windows localhost, sustained 10K concurrent sockets exhaust
  // ephemeral ports. This profile achieves 10K+ total requests safely.
  stages: [
    { duration: '10s', target: 500 },    // Warm up
    { duration: '15s', target: 2000 },   // Ramp to 2K concurrent
    { duration: '20s', target: 5000 },   // Ramp to 5K concurrent
    { duration: '15s', target: 5000 },   // Hold at 5K
    { duration: '10s', target: 0 },      // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],    // 95% of requests under 1s
    error_rate: ['rate<0.10'],            // Less than 10% unexpected errors
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // Each VU+iteration combo gets a unique user ID
  const userId = __VU * 100000 + __ITER;

  const res = http.post(`${BASE_URL}/book`, null, {
    headers: {
      'X-User-Id': `${userId}`,
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  });

  bookingDuration.add(res.timings.duration);

  const isOk = check(res, {
    'response received': (r) =>
      r.status === 200 || r.status === 404 || r.status === 429,
  });

  if (res.status === 200) {
    bookingSuccess.add(1);
    errorRate.add(false);
  } else if (res.status === 404) {
    soldOut.add(1);
    errorRate.add(false); // Sold out is expected, not an error
  } else if (res.status === 429) {
    bookingFailed.add(1);
    errorRate.add(false); // Rate limit is expected
  } else {
    errorRate.add(true);
  }

  sleep(0.1); // 100ms think time
}

// Query final state at test end
export function teardown() {
  const countRes = http.get(`${BASE_URL}/tickets/count`);
  console.log(`\n=== FINAL STATE ===`);
  console.log(`Remaining tickets: ${countRes.body}`);

  const metricsRes = http.get(`${BASE_URL}/metrics`);
  console.log(`Server metrics: ${metricsRes.body}`);
}
