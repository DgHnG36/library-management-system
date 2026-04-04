/**
 * LOAD TEST - LIBRARY MANAGEMENT SYSTEM
 * DESCRIPTION:
 * Simulates realistic concurrent traffic across three user roles:
 * 1. Regular Users: Register, login, browse books, create/cancel orders, view profile (70%)
 * 2. Managers: Login, view all orders, update order status, manage books (20%)
 * 3. Public Readers: Browse books without authentication (10%)
 * RUN: k6 run load_test.js --env BASE_URL=http://localhost:8080
 */

import http from 'k6/http'
import { check, group, sleep } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const JWT_SECRET = __ENV.JWT_SECRET || 'local-lms-secret-key'
const RUN_ID = Date.now()

const reqDuration = new Trend('req_duration', true)
const errorRate = new Rate('error_rate')
const successCount = new Counter('success_count')
const failedCount = new Counter('failed_count')


/**
 * CUSTOM METRICS
 * - Request durations for key operations (register, login, create order, list books, get book, manager login, list all orders, update order status)
 * - Error counts for authentication failures, order creation failures, and manager operation failures
 * - Success counts for successful operations
 */
const registerDuration = new Trend('register_duration', true)
const loginDuration = new Trend('login_duration', true)
const createOrderDuration = new Trend('create_order_duration', true)
const listBooksDuration = new Trend('list_books_duration', true)
const getBookDuration = new Trend('get_book_duration', true)
const managerLoginDuration = new Trend('manager_login_duration', true)
const listAllOrdersDuration = new Trend('list_all_orders_duration', true)
const updateOrderStatusDuration = new Trend('update_order_status_duration', true)

/**
 * ERROR METRICS
 * - auth_errors: Counts authentication failures during user and manager login attempts
 * - create_order_errors: Counts failures when regular users attempt to create orders
 * - manager_errors: Counts failures during manager-specific operations (login, listing all orders, updating order status)
 */
const authErrors = new Counter('auth_errors')
const createOrderErrors = new Counter('create_order_errors')
const managerErrors = new Counter('manager_errors')

export const options = {
    scenarios: {
        regular_users: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '1m', target: 20 },
                { duration: '3m', target: 20 },
                { duration: '1m', target: 50 },
                { duration: '3m', target: 50 },
                { duration: '1m', target: 100 },
                { duration: '3m', target: 100 },
                { duration: '1m', target: 300 },
                { duration: '3m', target: 300 },
                { duration: '1m', target: 500 },
                { duration: '3m', target: 500 },
                { duration: '5m', target: 0 }
            ],
            exec: 'regularUserFlow',
        },
        manager: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '1m', target: 5 },
                { duration: '1m', target: 15 },
                { duration: '1m', target: 30 },
                { duration: '2m', target: 50 },
                { duration: '2m', target: 0 },
            ],
            exec: 'managerFlow',
        },
        public_readers: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '30s', target: 5 },
                { duration: '1m', target: 10 },
                { duration: '1m', target: 20 },
                { duration: '2m', target: 50 },
                { duration: '2m', target: 0 },
            ],
            exec: 'publicReaderFlow',
        }
    },
    thresholds: {
        // Overall error rate must stay below 1 %
        error_rate: ['rate<0.01'],

        // Mixed metric — covers all request types under 500 VUs
        req_duration: ['p(95)<2000', 'p(99)<4000'],

        // Auth operations hit bcrypt + DB write: slow under high concurrency
        register_duration: ['p(95)<2000', 'p(99)<4000'],
        login_duration: ['p(95)<2000', 'p(99)<4000'],

        // Order write path under 500 VUs
        create_order_duration: ['p(50)<20', 'p(95)<1500', 'p(99)<3000'],

        // Read operations — fast even under load
        list_books_duration: ['p(50)<20', 'p(95)<500', 'p(99)<1000'],
        get_book_duration: ['p(50)<20', 'p(95)<500', 'p(99)<1000'],

        // Manager paths — low concurrency
        manager_login_duration: ['p(50)<20', 'p(95)<500', 'p(99)<1000'],
        list_all_orders_duration: ['p(50)<20', 'p(95)<1000', 'p(99)<3000'],
        update_order_status_duration: ['p(50)<20', 'p(95)<500', 'p(99)<1000'],

        // Error counters — allow ~1 % of total order volume for 500 VUs
        auth_errors: ['count<50'],
        create_order_errors: ['count<1000'],
        manager_errors: ['count<20'],
    }
}

/**
 * 
 * HELPER FUNCTIONS
 * - jsonHeaders: Constructs headers for JSON requests, optionally including an Authorization token
 * - randomPhoneNumber: Generates a random phone number for user registration
 */
function jsonHeader(token = '') {
    const headers = {
        "Content-Type": "application/json"
    }

    if (token) {
        headers["Authorization"] = `Bearer ${token}`
    }
    return headers
}

function randomPhoneNumber() {
    return `09${Math.floor(100000000 + Math.random() * 899999999)}`
}

/**
 * 
 * API INTERACTION FUNCTIONS
 * Each function corresponds to a specific API endpoint and includes:
 * - Timing the request and recording it in the appropriate Trend metric
 * - Checking the response for expected status codes and response structure
 * - Updating the errorRate, successCount, and failedCount metrics based on the outcome
 * - Logging detailed error information for failed requests to aid in debugging 
 */
function registerUser() {
    const uid = `${RUN_ID}-${__VU}-${__ITER}`
    const username = `load-user-${uid}`
    const password = `pass$${uid}`
    const email = `load-user-${uid}@test.com`
    const phone_number = randomPhoneNumber()

    const payload = JSON.stringify({
        username,
        password,
        email,
        phone_number
    })

    const start = Date.now()
    let resp = http.post(`${BASE_URL}/api/v1/auth/register`, payload, {
        headers: jsonHeader()
    })

    // Retry once on rate limit with a short back-off
    if (resp.status === 429) {
        sleep(2 + Math.random() * 3)
        resp = http.post(`${BASE_URL}/api/v1/auth/register`, payload, {
            headers: jsonHeader()
        })
    }

    registerDuration.add(Date.now() - start)

    const ok = check(resp, {
        'register status is 201': (r) => r.status === 201,
        'register response has user id': (r) => { try { return !!r.json('user_id') } catch (e) { return false } }
    })

    errorRate.add(!ok)
    if (!ok) {
        authErrors.add(1)
        failedCount.add(1)
        console.error(`[register] VU=${__VU} ITER=${__ITER} status=${resp.status} body=${resp.body}`)
        return null
    }

    successCount.add(1)
    return {
        username,
        password,
        email,
    }
}

function loginUser(identifier, password, durationMetric) {
    const metric = durationMetric || loginDuration

    const start = Date.now()
    let resp = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
        identifier,
        password,
    }), {
        headers: jsonHeader()
    })

    // Retry once on rate limit with a short back-off
    if (resp.status === 429) {
        sleep(2 + Math.random() * 3)
        resp = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
            identifier,
            password,
        }), {
            headers: jsonHeader()
        })
    }

    metric.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'login status is 200': (r) => r.status === 200,
        'login response has token': (r) => { try { return !!r.json('token_pair.access_token') } catch (e) { return false } }
    })

    errorRate.add(!ok)
    if (!ok) {
        if (resp.status !== 429) {
            authErrors.add(1)
        }
        failedCount.add(1)
        console.error(`[login] identifier=${identifier} status=${resp.status} body=${resp.body}`)
        return null
    }

    successCount.add(1)
    return resp.json('token_pair.access_token')
}

function listBooks(token) {
    const start = Date.now()
    const resp = http.get(`${BASE_URL}/api/v1/books`, {
        headers: jsonHeader(token)
    })
    listBooksDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'list books status is 200': (r) => r.status === 200,
    })

    errorRate.add(!ok)
    if (!ok) {
        failedCount.add(1)
        return []
    }

    successCount.add(1)
    return (resp.json('books') || []).map((b) => b.id).filter(Boolean);
}

function getBook(token, bookID) {
    const start = Date.now()
    const resp = http.get(`${BASE_URL}/api/v1/books/${bookID}`, {
        headers: jsonHeader(token)
    })
    getBookDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'get book status is 200': (r) => r.status === 200,
        'get book response has book title or isbn': (r) => { try { return r.json('book.id') !== undefined || r.json('book.isbn') !== undefined } catch (e) { return false } }
    })

    errorRate.add(!ok)
    ok ? successCount.add(1) : failedCount.add(1)
}

function createOrder(token, bookIDs, bookBorrowDays) {
    const start = Date.now()
    const resp = http.post(`${BASE_URL}/api/v1/orders`, JSON.stringify({
        book_ids: bookIDs,
        borrow_days: bookBorrowDays
    }), {
        headers: jsonHeader(token)
    })

    createOrderDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'create order status is 201': (r) => r.status === 201,
        'create order response has order id': (r) => { try { return !!r.json('order.id') } catch (e) { return false } }
    })
    errorRate.add(!ok)
    if (!ok) {
        createOrderErrors.add(1)
        failedCount.add(1)
        console.error(`[create order] status=${resp.status} body=${resp.body}`)
        return null
    }
    successCount.add(1)
    return resp.json('order.id')
}

function cancelOrder(token, orderID) {
    const start = Date.now()
    const resp = http.post(`${BASE_URL}/api/v1/orders/${orderID}/cancel`, null, {
        headers: jsonHeader(token)
    })
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'cancel order status is 200': (r) => r.status === 200,
    })
    errorRate.add(!ok)
    ok ? successCount.add(1) : failedCount.add(1)
}

function listAllOrders(managerToken) {
    const start = Date.now()

    const resp = http.get(`${BASE_URL}/api/v1/management/orders`, {
        headers: jsonHeader(managerToken)
    })
    listAllOrdersDuration.add(Date.now() - start)
    reqDuration.add(Date.now() - start)

    const ok = check(resp, {
        'list all orders status is 200': (r) => r.status === 200,
    })
    errorRate.add(!ok)
    if (!ok) {
        managerErrors.add(1)
        failedCount.add(1)
        return []
    }

    successCount.add(1)
    try { return resp.json('orders') || [] } catch (e) { return [] }
}

function updateOrderStatus(managerToken, orderID, status) {
    const start = Date.now()
    const resp = http.patch(
        `${BASE_URL}/api/v1/management/orders/${orderID}/status`,
        JSON.stringify({ new_status: status }),
        { headers: jsonHeader(managerToken) },
    )
    const elapsed = Date.now() - start
    updateOrderStatusDuration.add(elapsed)
    reqDuration.add(elapsed)

    const ok = check(resp, { 'update order status: status 200': (r) => r.status === 200 })
    errorRate.add(!ok)
    if (!ok) {
        managerErrors.add(1)
        failedCount.add(1)
        console.error(`[update order status] orderID=${orderID} status=${resp.status} body=${resp.body}`)
        return false
    }
    successCount.add(1)
    return true
}

export function setup() {
    const managerToken = loginUser('lms-manager', 'manager@413', managerLoginDuration)
    if (!managerToken) {
        console.error('[setup] manager login failed - falling back to existing catalogs')
        return {
            bookIDs: listBooks().slice(0, 10)
        }
    }

    const suffix = Date.now() % 100000000  // 8 digits → ISBN fits in varchar(20)
    const payload = {
        books_payload: Array.from({ length: 5 }, function (_, i) {
            return {
                title: `Load Test Book ${suffix}-${i}`,
                author: 'Load Tester',
                isbn: `978-${suffix}-${i}`,
                category: 'Testing',
                description: 'Seeded for load test',
                quantity: 1000,
            }
        })
    }

    const resp = http.post(`${BASE_URL}/api/v1/management/books`, JSON.stringify(payload), {
        headers: jsonHeader(managerToken)
    })

    let bookIDs = []
    if (resp.status === 201) {
        try {
            bookIDs = (resp.json('created_books') || []).map((b) => b.id).filter(Boolean)
        } catch (e) { }
    } else {
        console.error(`[setup] create books failed: ${resp.status} - ${resp.body}`)
        bookIDs = listBooks(managerToken).slice(0, 10)
    }

    console.log(`[setup] using ${bookIDs.length} book(s): ${bookIDs.join(', ')}`)
    return { bookIDs }
}
/**
 * FLOW FUNCTIONS
 * Each flow function simulates the behavior of a specific user role, executing a series of API interactions that reflect typical usage patterns. The functions include:
 * - regularUserFlow: Simulates a regular user registering, logging in, browsing books, creating/canceling orders, and viewing their profile
 * - managerFlow: Simulates a manager logging in, viewing all orders, updating order status, and managing books
 * - publicReaderFlow: Simulates a public reader browsing books without authentication
 * Each step within the flows includes checks for response validity and updates the custom metrics accordingly to track performance and error rates for each type of operation.
 */
export function regularUserFlow(data) {
    const bookIDs = data && data.bookIDs || []
    group('regular user: register', function () {
        const credentials = registerUser()
        if (!credentials) {
            sleep(1)
            return
        }

        group('regular user: login', function () {
            const token = loginUser(credentials.username, credentials.password)
            if (!token) {
                sleep(1)
                return
            }

            group('regular user: list books', function () {
                const ids = listBooks(token)
                if (bookIDs.length == 0 && ids.length > 0) {
                    ids.forEach(function (id) {
                        bookIDs.push(id)
                    })
                }
            })

            const targets = bookIDs.length > 0 ? [bookIDs[Math.floor(Math.random() * bookIDs.length)]] : []
            if (targets.length > 0) {
                group('regular user: get book', function () {
                    getBook(token, targets[Math.floor(Math.random() * targets.length)])
                })

                group('regular user: create order', function () {
                    const borrowDays = Math.floor(Math.random() * 14) + 1
                    const orderID = createOrder(token, targets, borrowDays)

                    if (orderID && Math.random() < 0.3) {
                        group('regular user: cancel order', function () {
                            cancelOrder(token, orderID)
                        })
                    }
                })
            }

            group('regular user: get profile', function () {
                const start = Date.now()
                const resp = http.get(`${BASE_URL}/api/v1/user/profile`, { headers: jsonHeader(token) })
                reqDuration.add(Date.now() - start)
                const ok = check(resp, {
                    'get profile: status 200': (r) => r.status === 200,
                    'get profile: has user id': (r) => { try { return !!r.json('user.id') } catch (e) { return false } },
                })
                errorRate.add(!ok)
                ok ? successCount.add(1) : failedCount.add(1)
            })
        })
    })

    sleep(1 + Math.random() * 2)
}

export function managerFlow(data) {
    group('manager: login', function () {
        const managerToken = loginUser('lms-manager', 'manager@413', managerLoginDuration)
        if (!managerToken) {
            sleep(1)
            return
        }

        group('manager: list all orders', function () {
            const orders = listAllOrders(managerToken)
            const pending = orders.filter(order => order.status === 'PENDING').slice(0, 3)
            pending.forEach(function (order) {
                group('manager: update order status', function () {
                    updateOrderStatus(managerToken, order.id, 'APPROVED')
                })
            })
        })

        group('manager: list books', function () {
            listBooks(managerToken)
        })
    })

    sleep(5 + Math.random() * 5)
}

export function publicReaderFlow(data) {
    const bookIDs = (data && data.bookIDs) || []

    group('public: list books', function () {
        const ids = listBooks()
        const pool = ids.length > 0 ? ids : bookIDs

        if (pool.length > 0) {
            const pick = pool[Math.floor(Math.random() * pool.length)]
            group('public: get book', function () {
                getBook('', pick)
            })
        }
    })

    sleep(0.5 + Math.random())
}

export default function () { }
