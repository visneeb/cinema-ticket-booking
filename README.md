
# Cinema Ticket Booking System

A real-time cinema seat booking system built with Go, Vue 3, Redis, RabbitMQ, and MongoDB Atlas.

---

## 1. System Architecture Diagram

```
Browser (Vue 3 SPA)
        │
        │  HTTP REST  /api/*
        │  WebSocket  /ws/showtimes/:id
        ▼
  ┌─────────────┐
  │   nginx     │  :3000  (Docker frontend)
  │  (reverse   │────────────────────────────┐
  │   proxy)    │                            │
  └─────────────┘                            │
        │  proxy_pass http://backend:8080    │
        ▼                                    │
  ┌─────────────────────────────────────────┐│
  │         Go Backend (Gin)          :8080 ││
  │                                         ││
  │  Handler → Service → Repository         ││
  │                                         ││
  │  ┌────────────┐  ┌──────────────────┐   ││
  │  │ WebSocket  │  │ RabbitMQ         │   ││
  │  │ Hub        │  │ Publisher        │   ││
  │  │ (per room) │  │ + 3 Consumers    │   ││
  │  └────────────┘  └──────────────────┘   ││
  └─────────────────────────────────────────┘│
        │            │              │         │
        ▼            ▼              ▼         │
   ┌─────────┐ ┌──────────┐ ┌──────────┐     │
   │  Redis  │ │ RabbitMQ │ │ MongoDB  │     │
   │  :6379  │ │  :5672   │ │  Atlas   │     │
   │ (locks) │ │ (events) │ │ (data)   │     │
   └─────────┘ └──────────┘ └──────────┘     │
                                              │
   Firebase Auth (external) ◄────────────────┘
```

---

## 2. Tech Stack Overview

| Layer | Technology | Purpose |
|---|---|---|
| **Frontend** | Vue 3 + TypeScript + Vite | SPA with Composition API |
| **State** | Pinia | Auth store, reactive state |
| **HTTP Client** | Axios | REST API calls with Firebase JWT interceptor |
| **Auth** | Firebase Authentication (Google OAuth) | User identity, JWT tokens |
| **Backend** | Go 1.26 + Gin | REST API, WebSocket hub, RabbitMQ workers |
| **Real-time** | WebSocket (gorilla/websocket) | Push seat status to all browser tabs |
| **Seat Locking** | Redis 8 | Distributed lock with 5-min TTL (`SET NX EX`) |
| **Database** | MongoDB Atlas | Bookings, seats, showtimes, movies, audit logs, users |
| **Message Queue** | RabbitMQ 4 (topic exchange) | Async notifications and audit logging |
| **Reverse Proxy** | nginx | SPA serving, API + WebSocket proxy |
| **Container** | Docker + Docker Compose | One-command full-stack startup |

---

## 3. Booking Flow — Step by Step

```
Step 1 — Browse Showtimes
  GET /api/showtimes
  └─ Returns showtimes joined with movie title from MongoDB

Step 2 — View Seats
  GET /api/showtimes/:id/seats
  └─ Reads seats from MongoDB
  └─ Overlays Redis statuses (MGET) on top of MongoDB status
  └─ Redis wins for LOCKED (live state), MongoDB wins for BOOKED (permanent)

Step 3 — User Selects a Seat (Phase 1: Select)
  POST /api/showtimes/:st/seats/:seat/lock   [requires Firebase JWT]
  └─ Redis SET NX EX → acquire distributed lock (5 min TTL)
  └─ Redis SET seat:status → "LOCKED" (5 min TTL)
  └─ Redis SADD user:lock → reverse index (used by /my-lock)
  └─ MongoDB seats.status → "LOCKED" (write-through cache)
  └─ WebSocket BroadcastSeatEvent → all other tabs see LOCKED instantly

Step 4 — User Confirms Reservation (Phase 2: Reserve)
  POST /api/showtimes/:st/seats/:seat/lock  (re-lock to refresh TTL)
  └─ If same user calls lock again → returns remaining TTL from Redis

Step 5 — User Confirms Booking (Phase 3: Book)
  POST /api/showtimes/:st/seats/:seat/book  [requires Firebase JWT]
  └─ Verify lock owner in Redis == caller UID
  └─ Insert booking document into MongoDB (status: BOOKED)
  └─ Redis SET seat:status → "BOOKED" (no TTL — permanent)
  └─ Redis DEL lock key + remove from user:lock set
  └─ MongoDB seats.status → "BOOKED"
  └─ WebSocket BroadcastSeatEvent → all tabs see BOOKED
  └─ Publish booking.confirmed to RabbitMQ exchange (async)

Step 6 — Cancellation (optional)
  DELETE /api/showtimes/:st/seats/:seat/lock
  └─ Verify lock owner == caller
  └─ Delete Redis lock + status + reverse index
  └─ MongoDB seats.status → "AVAILABLE"
  └─ WebSocket BroadcastSeatEvent → all tabs see AVAILABLE

Step 7 — Lock Expiry (timeout)
  └─ Redis TTL expires automatically after 5 min
  └─ booking.timeout published to RabbitMQ
  └─ AuditConsumer: clears Redis status, resets MongoDB, broadcasts AVAILABLE
```

---

## 4. Redis Lock Strategy

Three Redis key types are written atomically per seat operation:

```
lock:seat:{showtimeID}:{seatID}      → userID            (TTL: 5 min, SET NX EX)
seat:status:{showtimeID}:{seatID}    → "LOCKED"/"BOOKED" (TTL: 5 min or permanent)
user:lock:{showtimeID}:{userID}      → Set{seatID, ...}  (TTL: 5 min, Redis Set)
```

**Key behaviors:**

| Scenario | Behavior |
|---|---|
| First lock attempt | `SET NX EX` — atomic, only succeeds if key does not exist |
| Another user tries to lock | `SET NX` returns false → 409 Conflict |
| Same user refreshes page | Lock key found with same UID → return remaining TTL |
| Lock TTL expires | Key auto-deleted by Redis; status key cleaned up lazily on next request |
| Confirm booking | Status set to `BOOKED` with **no TTL** (permanent), lock key deleted |
| Cancel | Lua compare-and-delete script — only owner can release (atomic) |

**Why two keys per seat?**
- `lock:seat:*` is the **ownership mutex** — only the holder's UID is stored
- `seat:status:*` is the **fast read cache** — `GET /seats` does one `MGET` for all seats instead of checking ownership per seat
- `user:lock:*` is the **reverse index** — lets `/my-lock` recover session after a page refresh

---

## 5. Message Queue — What RabbitMQ Does

**Exchange:** `booking` (topic, durable)

```
booking.confirmed ──► notify      queue  ──► NotificationConsumer
                  └──► audit_log  queue  ──► AuditLogConsumer

booking.timeout   ──► audit_timeout queue ──► AuditConsumer
```

| Consumer | Queue | Event | Action |
|---|---|---|---|
| `StartNotificationConsumer` | `notify` | `booking.confirmed` | Mock email log (swap for SMTP/SendGrid) |
| `StartAuditLogConsumer` | `audit_log` | `booking.confirmed` | Write `BOOKING_SUCCESS` to `audit_logs` collection |
| `StartAuditConsumer` | `audit_timeout` | `booking.timeout` | Clear Redis keys, reset MongoDB seat to AVAILABLE, broadcast AVAILABLE via WebSocket, write `BOOKING_TIMEOUT` audit log |

**Why RabbitMQ instead of calling these inline?**
- Email and audit logging are **not on the critical path** — the user should not wait for them
- If the notification service fails, the booking is already saved — no data loss
- Durable queues + persistent delivery mode survive a RabbitMQ restart
- Manual ACK ensures messages are not lost if a consumer crashes mid-processing

---

## 6. How to Run

### Prerequisites

- Docker & Docker Compose installed
- MongoDB Atlas cluster (free tier works)
- Firebase project with Google sign-in enabled
- `firebase-service-account.json` at the project root

### Setup

```bash
# 1. Clone the repo
git clone <repo-url>
cd ctb

# 2. Copy and fill in the environment file
cp .env.example .env
# Edit .env — fill in MONGO_URI, RABBITMQ_USER, RABBITMQ_PASS, all VITE_FIREBASE_* keys

# 3. Place your Firebase service account at the project root
#    (download from Firebase Console → Project Settings → Service Accounts)
mv ~/Downloads/firebase-service-account.json ./firebase-service-account.json

# 4. Start everything
docker compose up --build
```

### Access

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| Swagger UI | http://localhost:8080/swagger/index.html |
| RabbitMQ Management | http://localhost:15672 |

### Local Development (without Docker)

```bash
# Start infrastructure only
docker compose up redis rabbitmq

# Backend (from cinema-ticket-booking-server/)
go run ./cmd/server/main.go

# Frontend (from cinema-ticket-booking-client/)
npm install
npm run dev       # → http://localhost:5173
```

### Run Tests

```bash
# Frontend unit tests
cd cinema-ticket-booking-client
npm run test:unit

# Backend tests
cd cinema-ticket-booking-server
go test ./...
```

---

## 7. Assumptions & Trade-offs

| # | Decision | Rationale / Trade-off |
|---|---|---|
| 1 | **Redis as distributed lock** | Atomic `SET NX EX` prevents double-booking without a database transaction. Trade-off: Redis is a single point of failure (no Redlock used — acceptable for a single-node deployment). |
| 2 | **5-minute lock TTL** | Long enough for a user to complete payment, short enough to not block other users indefinitely. TTL is hardcoded — not configurable per showtime. |
| 3 | **MongoDB write-through on every lock** | `seats.status` is updated in MongoDB whenever Redis is updated, so `GET /seats` stays accurate even after a Redis restart. Trade-off: one extra MongoDB write per lock/unlock operation. |
| 4 | **WebSocket hub is in-process** | Simple and fast for a single backend instance. Trade-off: does not scale horizontally — multiple backend instances would need a shared pub/sub (e.g., Redis Pub/Sub) to broadcast across nodes. |
| 5 | **RabbitMQ for async events** | Decouples notification and audit logging from the booking transaction. Trade-off: adding RabbitMQ as a required dependency increases operational complexity. |
| 6 | **Firebase Auth (no custom auth)** | Eliminates password management, email verification, and session storage. Trade-off: creates a hard dependency on Google's infrastructure. |
| 7 | **MongoDB Atlas (no self-hosted Mongo)** | No Mongo container to manage. Trade-off: requires internet access and an Atlas account; not fully air-gapped. |
| 8 | **Single backend instance** | Keeps architecture simple. Trade-off: the WebSocket hub, RabbitMQ connection, and Redis connection are all per-process — horizontal scaling requires additional work. |
| 9 | **Admin dashboard shows only confirmed bookings** | `LOCKED` state exists only in Redis (not in the `bookings` collection), so the admin table cannot show in-progress seat holds by design. |
