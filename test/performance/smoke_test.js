/**
 * SMOKE TEST - LIBRARY MANAGEMENT SYSTEM 
 * DESCRIPTION:
 * This smoke test script is designed to perform a basic 
 * health check and functionality test of the Library Management System (LMS) API. 
 * It simulates a simple user flow that includes:
 * - Checking the health and readiness endpoints
 * - Registering a new user
 * - Logging in with the new user
 * - Retrieving the user's profile
 * - Listing available books
 * RUN: k6 run smoke_test.js --env BASE_URL=http://localhost:8080
 */

import http from 'k6/http'
import { check, group, sleep } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const JWT_SECRET = __ENV.JWT_SECRET || 'local-lms-secret-key'
const RUN_ID = Date.now()

/*
 * CUSTOM METRICS
 */

const smokeDuration = new Trend('smoke_duration', true)
const errorRate = new Rate('error_rate')
const successCount = new Counter('success_count')
const failedCount = new Counter('failed_count')

export const options = {
    vus: 3,
    iterations: 30,
    thresholds: {
        error_rate: ['rate==0'],
        smoke_duration: ['p(95)<5000']
    }
};

function jsonHeaders(token = '') {
    const headers = {
        "Content-Type": "application/json"
    }

    if (token) {
        headers["Authorization"] = `Bearer ${token}`
    }
    return headers
}

function checkHealth() {
    const resp = http.get(`${BASE_URL}/healthy`)

    const ok = check(resp, {
        'health check status is 200': (r) => r.status === 200,
        'health check response is healthy': (r) => r.json('status') == 'healthy'
    })

    if (!ok) {
        console.error(`Health check failed: ${resp.status} - ${resp.body}`)
    }

    errorRate.add(!ok)
    return ok
}

function checkReadiness() {
    const resp = http.get(`${BASE_URL}/ready`)

    const ok = check(resp, {
        'readiness check status is 200': (r) => r.status === 200,
        'readiness check response is ready': (r) => r.json('status') === 'ready'
    })

    if (!ok) {
        console.error(`Readiness check failed: ${resp.status} - ${resp.body}`)
    }

    errorRate.add(!ok)
    return ok
}

function checkMetricsEndpoint() {
    const resp = http.get(`${BASE_URL}/metrics`)

    const ok = check(resp, {
        'metrics endpoint status is 200': (r) => r.status === 200,
    })

    if (!ok) {
        console.error(`Metrics endpoint check failed: ${resp.status} - ${resp.body}`)
    }

    errorRate.add(!ok)
    return ok
}

export default function () {
    const start = Date.now()

    // Check health and readiness endpoints
    if (!checkHealth()) {
        failedCount.add(1)
        return
    }

    if (!checkReadiness()) {
        failedCount.add(1)
        return
    }

    // Check metrics endpoint
    if (!checkMetricsEndpoint()) {
        failedCount.add(1)
        return
    }

    successCount.add(1)
    const duration = Date.now() - start
    smokeDuration.add(duration)

    // Start basic flow for smoke testing
    // Step 1: Register new user
    const registerPayload = JSON.stringify({
        username: `smoke_${RUN_ID}_${__VU}_${__ITER}`,
        password: `smokeuser${__VU}`,
        email: `smoke_${RUN_ID}_${__VU}_${__ITER}@test.com`,
        phone_number: `0${Math.floor(1000000000 + Math.random() * 9000000000)}`
    })

    const registerResp = http.post(`${BASE_URL}/api/v1/auth/register`, registerPayload, {
        headers: jsonHeaders()
    })

    const registerOk = check(registerResp, {
        'register status is 201': (r) => r.status === 201,
        'register response has user id': (r) => r.json('user_id') !== undefined
    })

    if (!registerOk) {
        console.error(`User registration failed: ${registerResp.status} - ${registerResp.body}`)
        errorRate.add(1)
        failedCount.add(1)
        return
    }

    successCount.add(1)

    // Step 2: Login with the new user
    const loginPayload = JSON.stringify({
        identifier: `smoke_${RUN_ID}_${__VU}_${__ITER}`,
        password: `smokeuser${__VU}`
    })

    const loginResp = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
        headers: jsonHeaders()
    })

    const loginOk = check(loginResp, {
        'login status is 200': (r) => r.status === 200,
        'login response has token': (r) => r.json().hasOwnProperty('token_pair'),
        'login response has access token': (r) => r.json('token_pair.access_token') !== undefined,
        'login response has user': (r) => r.json('user.id') !== undefined
    })

    if (!loginOk) {
        console.error(`User login failed: ${loginResp.status} - ${loginResp.body}`)
        errorRate.add(1)
        failedCount.add(1)
        return
    }

    successCount.add(1)

    const userID = loginResp.json('user.id')
    const accessToken = loginResp.json('token_pair.access_token')

    // Step 3: Get user profile
    const profileResp = http.get(`${BASE_URL}/api/v1/user/profile`, {
        headers: jsonHeaders(accessToken)
    })

    const profileOk = check(profileResp, {
        'profile status is 200': (r) => r.status === 200,
        'profile response has user': (r) => r.json('user.id') === userID,
        'profile response has correct username': (r) => r.json('user.username') === `smoke_${RUN_ID}_${__VU}_${__ITER}`,
        'profile response has correct email': (r) => r.json('user.email') === `smoke_${RUN_ID}_${__VU}_${__ITER}@test.com`
    })

    if (!profileOk) {
        console.error(`Get profile failed: ${profileResp.status} - ${profileResp.body}`)
        errorRate.add(1)
        failedCount.add(1)
        return
    }

    successCount.add(1)

    // Step 4: List books
    const booksResp = http.get(`${BASE_URL}/api/v1/books`, {
        headers: jsonHeaders(accessToken)
    })

    const booksOk = check(booksResp, {
        'list books status is 200': (r) => r.status === 200,
    })

    if (!booksOk) {
        console.error(`List books failed: ${booksResp.status} - ${booksResp.body}`)
        errorRate.add(1)
        failedCount.add(1)
        return
    }

    successCount.add(1)

    // Total duration
    smokeDuration.add(Date.now() - start)

    sleep(1)
}