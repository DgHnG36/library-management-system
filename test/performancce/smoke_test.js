import http from 'k6/http'
import { check, group, sleep } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'

const loginDuration = new Trend('login_duration', true)
const bookListsDuration = new Trend('book_lists_duration', true)
const orderDuration = new Trend('order_duration', true)
const errorRate = new Rate('error_rate')
const authErrors = new Counter('auth_errors')

export const options = {
    smoke: {
        executor: 'constant-vus',
        vus: 2,
        duration: '30s',
        tags: { scenario: 'smoke' }
    },
    load: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 10 },
            { duration: '2m', target: 20 },
            { duration: '1m', target: 0 },
        ],
        tags: { scenario: 'load' },
        startTime: '35s',
    },
    thresholds: {
        'http_req_duration': ['p(95)<500,p(99)<1000'],
        'http_req_failed': ['rate<0.01'],
        'login_duration': ['p(95)<300'],
        'book_lists_duration': ['p(95)<400'],
        'order_duration': ['p(95)<600'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const JWT_SECRET = __ENV.JWT_SECRET || 'local-dev-secret-key'

function baseHeaders(token) {
    return {
        "Content-type": "application/json",
        "Accept": "application/json",
        "Origin": "http://localhost:3000",
        "X-Request-ID": `k6-${Date.now()} - ${Math.random().toString(36).slice(2, 7)}`,
        ...(token ? { "Authorization": `Bearer ${token}` } : {}),
    };
}

function randomSuffix() {
    return Math.random().toString(36).slice(2, 10);
}




export function setup() {
    console.log('Checking health, readiness, and metrics of the system...')

    // Health check
    let healthRes = http.get(`${BASE_URL}/health`)
    check(healthRes, {
        'Health check status is 200': (r) => r.status === 200,
    });
    sleep(1);

    // Readiness check
    let readinessRes = http.get(`${BASE_URL}/ready`)
    check(readinessRes, {
        'Readiness check status is 200': (r) => r.status === 200,
    });
    sleep(1);

    // Metrics check
    let metricsRes = http.get(`${BASE_URL}/metrics`)
    check(metricsRes, {
        'Metrics check status is 200': (r) => r.status === 200,
    });
    sleep(1);
    console.log('System is healthy, ready, and metrics are accessible.')

}

export default function () {
    console.log('Running smoke test...')
    const paramsHeader = {

    }
}