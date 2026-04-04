/**
 * STRESS TEST — Library Management System
 *
 * Goal: find the breaking point beyond the 500-VU load test baseline.
 * Strategy: push to 1500 VUs (3× load test peak) in stages, observe
 * where error rate and latency degrade unacceptably.
 *
 * Acceptance criteria (looser than load test — we expect some degradation):
 *   - error_rate        < 5 %
 *   - register p(95)    < 5 s
 *   - login    p(95)    < 5 s
 *   - create_order p(95)< 3 s
 *   - list_books p(95)  < 1 s
 *   - get_book p(95)    < 1 s
 *
 * Run:
 *   k6 run --env BASE_URL=http://localhost:8080 stress_test.js
 */

import http from 'k6/http'
import { check, group, sleep } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const RUN_ID = Date.now()

/* ------------------------------------------------------------------ */
/*  Custom metrics                                                       */
/* ------------------------------------------------------------------ */

const reqDuration          = new Trend('req_duration', true)
const errorRate            = new Rate('error_rate')
const successCount         = new Counter('success_count')
const failedCount          = new Counter('failed_count')

const registerDuration     = new Trend('register_duration', true)
const loginDuration        = new Trend('login_duration', true)
const createOrderDuration  = new Trend('create_order_duration', true)
const listBooksDuration    = new Trend('list_books_duration', true)
const getBookDuration      = new Trend('get_book_duration', true)

const authErrors           = new Counter('auth_errors')
const createOrderErrors    = new Counter('create_order_errors')

/* ------------------------------------------------------------------ */
/*  Test options                                                        */
/*                                                                      */
/*  Stage design (total ~27 min):                                       */
/*   500 VU  — confirmed stable in load test                           */
/*   750 VU  — 1.5× load test peak                                     */
/*  1000 VU  — 2× load test peak                                       */
/*  1250 VU  — 2.5× load test peak                                     */
/*  1500 VU  — 3× load test peak  (expected breaking point)            */
/* ------------------------------------------------------------------ */

export const options = {
    scenarios: {
        stress_users: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                // Warm-up to load test baseline
                { duration: '2m',  target: 100  },
                { duration: '2m',  target: 300  },
                { duration: '2m',  target: 500  },
                { duration: '3m',  target: 500  }, // hold — confirm stable

                // Push beyond load test
                { duration: '2m',  target: 750  },
                { duration: '3m',  target: 750  }, // hold — observe

                // Double the load
                { duration: '2m',  target: 1000 },
                { duration: '3m',  target: 1000 }, // hold — observe

                // 2.5×
                { duration: '2m',  target: 1250 },
                { duration: '3m',  target: 1250 }, // hold — observe

                // 3× — breaking point test
                { duration: '2m',  target: 1500 },
                { duration: '3m',  target: 1500 }, // hold — measure failure mode

                // Recovery — system should recover after load drops
                { duration: '2m',  target: 500  },
                { duration: '2m',  target: 0    },
            ],
            exec: 'stressFlow',
        },
    },
    thresholds: {
        // Stress test allows higher error rate than load test
        error_rate: ['rate<0.05'],

        // Latency degrades under stress — thresholds reflect expected slowdown
        req_duration:         ['p(95)<5000'],
        register_duration:    ['p(95)<5000'],
        login_duration:       ['p(95)<5000'],
        create_order_duration:['p(95)<3000'],
        list_books_duration:  ['p(95)<1000'],
        get_book_duration:    ['p(95)<1000'],

        // Error counters
        auth_errors:         ['count<500'],
        create_order_errors: ['count<2000'],
    },
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                             */
/* ------------------------------------------------------------------ */

function jsonHeader(token) {
    const h = { 'Content-Type': 'application/json' }
    if (token) h['Authorization'] = `Bearer ${token}`
    return h
}

function randomPhone() {
    return `09${Math.floor(10000000 + Math.random() * 89999999)}`
}

function registerUser() {
    const uid      = `${RUN_ID}_${__VU}_${__ITER}`
    const username = `stress_${uid}`
    const password = `stresspass_${__VU}`
    const email    = `stress_${uid}@test.com`

    const start = Date.now()
    let resp = http.post(
        `${BASE_URL}/api/v1/auth/register`,
        JSON.stringify({ username, password, email, phone_number: randomPhone() }),
        { headers: jsonHeader() },
    )
    if (resp.status === 429) {
        sleep(2 + Math.random() * 3)
        resp = http.post(
            `${BASE_URL}/api/v1/auth/register`,
            JSON.stringify({ username, password, email, phone_number: randomPhone() }),
            { headers: jsonHeader() },
        )
    }
    registerDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'register: status 201':  (r) => r.status === 201,
        'register: has user_id': (r) => { try { return !!r.json('user_id') } catch (e) { return false } },
    })
    errorRate.add(!ok)
    if (!ok) {
        if (resp.status !== 429) authErrors.add(1)
        failedCount.add(1)
        return null
    }
    successCount.add(1)
    return { username, password }
}

function loginUser(identifier, password) {
    const start = Date.now()
    let resp = http.post(
        `${BASE_URL}/api/v1/auth/login`,
        JSON.stringify({ identifier, password }),
        { headers: jsonHeader() },
    )
    if (resp.status === 429) {
        sleep(2 + Math.random() * 3)
        resp = http.post(
            `${BASE_URL}/api/v1/auth/login`,
            JSON.stringify({ identifier, password }),
            { headers: jsonHeader() },
        )
    }
    loginDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'login: status 200':        (r) => r.status === 200,
        'login: has access_token':  (r) => { try { return !!r.json('token_pair.access_token') } catch (e) { return false } },
    })
    errorRate.add(!ok)
    if (!ok) {
        if (resp.status !== 429) authErrors.add(1)
        failedCount.add(1)
        return null
    }
    successCount.add(1)
    return resp.json('token_pair.access_token')
}

function listBooks(token) {
    const start = Date.now()
    const resp  = http.get(`${BASE_URL}/api/v1/books`, { headers: jsonHeader(token) })
    listBooksDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, { 'list books: status 200': (r) => r.status === 200 })
    errorRate.add(!ok)
    if (!ok) { failedCount.add(1); return [] }
    successCount.add(1)
    try { return (resp.json('books') || []).map((b) => b.id).filter(Boolean) } catch (e) { return [] }
}

function getBook(bookID, token) {
    const start = Date.now()
    const resp  = http.get(`${BASE_URL}/api/v1/books/${bookID}`, { headers: jsonHeader(token) })
    getBookDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, { 'get book: status 200': (r) => r.status === 200 })
    errorRate.add(!ok)
    ok ? successCount.add(1) : failedCount.add(1)
}

function createOrder(token, bookIDs, borrowDays) {
    const start = Date.now()
    const resp  = http.post(
        `${BASE_URL}/api/v1/orders`,
        JSON.stringify({ book_ids: bookIDs, borrow_days: borrowDays }),
        { headers: jsonHeader(token) },
    )
    createOrderDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'create order: status 201':   (r) => r.status === 201,
        'create order: has order id': (r) => { try { return !!r.json('order.id') } catch (e) { return false } },
    })
    errorRate.add(!ok)
    if (!ok) {
        createOrderErrors.add(1)
        failedCount.add(1)
        return null
    }
    successCount.add(1)
    return resp.json('order.id')
}

/* ------------------------------------------------------------------ */
/*  setup — seed shared book catalog                                   */
/* ------------------------------------------------------------------ */

export function setup() {
    let token = loginUser('lms-manager', 'manager@413')
    if (!token) {
        console.error('[setup] manager login failed')
        return { bookIDs: [] }
    }

    const suffix = Date.now() % 100000000
    const payload = {
        books_payload: Array.from({ length: 10 }, function(_, i) {
            return {
                title:       `Stress Test Book ${suffix}-${i}`,
                author:      'Stress Tester',
                isbn:        `978-${suffix}-${i}`,
                category:    'Testing',
                description: 'Seeded for stress test',
                quantity:    2000,
            }
        }),
    }

    const resp = http.post(
        `${BASE_URL}/api/v1/management/books`,
        JSON.stringify(payload),
        { headers: jsonHeader(token) },
    )

    let bookIDs = []
    if (resp.status === 201) {
        try { bookIDs = (resp.json('created_books') || []).map((b) => b.id).filter(Boolean) } catch (e) {}
    } else {
        console.error(`[setup] seed books failed: ${resp.status} ${resp.body}`)
        bookIDs = listBooks(token).slice(0, 10)
    }

    console.log(`[setup] seeded ${bookIDs.length} book(s)`)
    return { bookIDs }
}

/* ------------------------------------------------------------------ */
/*  Stress flow — simplified version of load test regular user flow   */
/*  (skip profile fetch to reduce request count per iteration)        */
/* ------------------------------------------------------------------ */

export function stressFlow(data) {
    const bookIDs = (data && data.bookIDs) || []

    // Register
    const creds = registerUser()
    if (!creds) { sleep(1 + Math.random()); return }

    // Login
    const token = loginUser(creds.username, creds.password)
    if (!token) { sleep(1 + Math.random()); return }

    // Browse books
    const ids  = listBooks(token)
    const pool = ids.length > 0 ? ids : bookIDs

    if (pool.length > 0) {
        // Get a specific book
        const pick = pool[Math.floor(Math.random() * pool.length)]
        getBook(pick, token)

        // Create order
        const borrowDays = Math.floor(3 + Math.random() * 11)
        createOrder(token, [pick], borrowDays)
    }

    sleep(1 + Math.random() * 2)
}

export default function () {}
