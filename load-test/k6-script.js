import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    // Ramp up to 500 VUs over 20s, hold for 30s, ramp down over 10s
    stages: [
        { duration: '20s', target: 500 },
        { duration: '30s', target: 500 },
        { duration: '10s', target: 0 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms (we expect <50ms since it's just queuing)
        http_req_failed: ['rate<0.01'],    // less than 1% errors
    },
};

const BASE_URL = 'http://localhost:8080';

export default function () {
    const payload = JSON.stringify({
        device_id: `device-k6-${__VU}`,
        sensor_type: Math.random() > 0.5 ? 'temperature' : 'vibration',
        nature: 'analog',
        value: Math.random() * 100,
        // skipping timestamp, backend will auto-add it
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const res = http.post(`${BASE_URL}/telemetry`, payload, params);

    check(res, {
        'status is 202': (r) => r.status === 202,
    });

    // small sleep to not completely bombard localhost from a single VU
    // roughly 5 requests per second per VU
    sleep(0.2); 
}
