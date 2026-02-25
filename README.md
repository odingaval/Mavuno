# 🌾 Mvuno
## Harvest Without Limits.

Mavuno is a Local-First Progressive Web Application (PWA) built with Go.

It enables smallholder farmers to:
- 🌽 Manage produce inventory offline
- 📚 Access agricultural learning content offline
- 🛒 Create market listings without internet
- 🔄 Sync automatically when connectivity returns

Built for the “LOCAL FIRST: Build Our Reality” Hackathon.

---

# 🌍 Why Mavuno?

Most digital platforms assume:
- Stable internet
- Constant backend availability
- Unlimited data
- Always-on devices

In reality:
- Connections drop
- Data is expensive
- Power fails
- Devices are shared

Mavuno is designed for these conditions.

---

# 🧠 Local-First Architecture

Mavuno follows a strict Local-First principle:

1. All writes occur locally first (IndexedDB)
2. The UI never blocks waiting for network
3. Sync runs in the background
4. Failures are handled gracefully
5. Conflicts are detected using version control

The network is treated as an enhancement — not a dependency.

---

# 🏗 System Architecture

Browser (PWA)
- IndexedDB
- Service Worker
- Background Sync Queue
- Optimistic UI Updates

⬇ REST API

Go Backend
- Versioned records
- Conflict detection (409)
- Partial (delta) updates
- SQLite persistence

---

# 📁 Project Structure

```
mavuno/
├── cmd/server/         # Application entry
├── internal/
│   ├── api/            # HTTP handlers
│   ├── models/         # Domain models
│   ├── services/       # Business logic
│   ├── storage/        # Database layer
│   └── middleware/     # HTTP middleware
├── web/                # PWA frontend
└── go.mod
```

---

# 📦 Data Model Example

Each record is version-controlled:

```go
type Produce struct {
    ID        string
    Name      string
    Quantity  int
    Price     float64
    Version   int
    UpdatedAt time.Time
}
```

Updates must include a matching `Version`.

If versions mismatch:
- Server returns HTTP 409 Conflict
- Client triggers conflict resolution

---

# 🔄 Sync Strategy

Client stores offline mutations in a Sync Queue:

```
{
  id,
  entityType,
  operation,
  payload,
  retryCount,
  status
}
```

When online:
- Sync queue is processed sequentially
- Failed requests retry with exponential backoff
- Delta updates reduce payload size
- Idempotent endpoints prevent duplication

---

# ⚠ Failure Handling

| Scenario | System Response |
|-----------|----------------|
| Network drops mid-sync | Resume from last successful operation |
| Server returns 500 | Retry with backoff |
| App closed during write | IndexedDB atomic transaction |
| Same record edited on two devices | 409 conflict detection |
| Duplicate submission | Idempotent update handling |

---

# 🛠 Tech Stack

Backend:
- Go (net/http)
- SQLite
- Clean layered architecture

Frontend:
- HTML/CSS
- Vanilla JavaScript
- IndexedDB
- Service Worker
- Background Sync API

---

# 🚀 Running the Project

## 1. Clone repository

```
git clone https://github.com/your-org/mvuno.git
cd mavuno
```

## 2. Install dependencies

```
go mod tidy
```

## 3. Run server

```
go run ./cmd/server
```

Server runs at:
```
http://localhost:8080
```

Frontend is served from `/web`.

---

# 🧪 Testing Offline Mode

1. Open app in browser
2. Open DevTools → Network
3. Select "Offline"
4. Add produce
5. Create listing
6. Re-enable internet
7. Observe background sync

---

# 🎤 Hackathon Demo Flow

1. Turn internet OFF
2. Add produce
3. Access learning content
4. Create listing
5. Close and reopen app
6. Turn internet ON
7. Show sync processing
8. Demonstrate conflict handling

---

# 🏆 Judging Alignment

| Category | Implementation |
|------------|----------------|
| Local-First Architecture | IndexedDB + Service Worker |
| Reliability Under Failure | Retry + Resume + Idempotent API |
| Technical Depth | Versioning + Conflict Handling |
| UX & Usability | Optimistic UI |
| Code Quality | Clean Go architecture |

---

# 🔮 Future Improvements

- SMS fallback notifications
- Buyer-side PWA
- Cooperative aggregation
- Battery-aware sync scheduling
- Payload compression
- AI-driven yield forecasting

---

# 📄 License

MIT
