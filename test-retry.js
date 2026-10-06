import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 50,
  duration: '30s',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const userId = __VU;
  const headers = {
    'X-User-Id': `${userId}`,
    'Content-Type': 'application/json',
  };

  const first = http.post(`${BASE_URL}/book`, null, { headers, timeout: '10s' });
  check(first, { 'first ok or sold out': (r) => r.status === 200 || r.status === 404 });

  const retry = http.post(`${BASE_URL}/book`, null, { headers, timeout: '10s' });
  if (first.status === 200 && retry.status === 200) {
    const a = JSON.parse(first.body);
    const b = JSON.parse(retry.body);
    check(retry, {
      'idempotent ticket_id': () => a.ticket_id === b.ticket_id,
    });
  }

  sleep(0.05);
}
