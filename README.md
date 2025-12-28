# Concurrent Job Deduplication System

## Overview

This code contains logic for the concurrent job deduplication system using Go. 

It ensures that when multiple requests ask for the same work:
- The work is executed **only once**
- All callers **share the same result given the same company ID**
- Results are cached at **Initial Base Calculations** and **API Specific Calculations** for the same company
- Results are **cached with TTL**
- The system is **safe under concurrency**


## How i designed it

The system has two layers:

### Base Jobs
These are the base calculations needed for the api specific computations to proceed. We ensure we have One `BaseJob` per company which can be shared across api types for the same company ID. we are also Caching the result of Base Jobs with TTL. 


### API Jobs
These are the api specific jobs users have requested which depends on Base Jobs. this is Deduplicated across multiple callers and Cached with TTL.


## Implementation Details

### 1. Job Managers

The system has two layers of jobs:

#### a) BaseJobManager
- Manages **BaseJobs**, representing shared work per `companyID`.
- Each `BaseJob` contains:
  - `mu` - Mutext to preserve shared job state
  - `Result` — computed value.
  - `Err` — error, if any.
  - `Done` — a channel that signals completion.
  - `CreatedAt` and `CompletedAt` timestamps for TTL handling.
- Deduplication logic:
  1. On `GetOrCreate`, check if a job exists in the map.
  2. If it exists and TTL has not expired, **return the same job**.
  3. Otherwise, **create a new job**, store it in the map, and run it in a goroutine.
- The job goroutine simulates work, sets `Result`/`Err`, updates `CompletedAt`, and closes `Done`.

#### b) APIJobManager
- Manages **APIJobs**, specific to `(companyID, APIType)`.
- Each `APIJob` also has `Result`, `Err`, `Done`, and timestamps.
- Deduplication logic is similar to `BaseJobManager`:
  1. Lookup by key `(companyID:APIType)`.
  2. Return existing job if active; otherwise, create a new job.
- Dependency handling:
  - Each `APIJob` waits for the corresponding `BaseJob` to complete.
  - If the base job fails, the API job propagates the error.
  - Completion is signaled via the job’s `Done` channel.

---

### 2. Concurrency Handling

- **Mutexes** protect the job maps and individual job fields.
- `Done` channels serve as a **synchronization barrier**:
  - Multiple goroutines can safely wait for job completion.
- **Context propagation** allows request cancellation and timeouts to flow downstream.
- **Lazy TTL eviction** ensures completed results are evicted only when requested, preventing in-flight job deletion.

---

### 3. Deduplication and Caching

- Deduplication occurs at two levels:
  - **BaseJob**: deduplicated per `companyID`.
  - **APIJob**: deduplicated per `(companyID, APIType)`.
- TTL is measured from `CompletedAt`:
  - Jobs are only evicted after they finish and exceed TTL.
  - This avoids duplicate executions while jobs are still running.

---

### 4. HTTP Integration

- Handlers call `APIJobManager.GetOrCreate(ctx, companyID, apiType)`.
- The HTTP request’s context is passed to the job to support cancellation.
- The handler waits on `job.Done` for completion, then returns the result.
- If the request context is canceled before job completion, a **timeout response** is returned.

---


## Key Guarantees

- At-most-once execution per company ID and API type
- Safe under high concurrency
- No data races or deadlocks
- No eviction of in-flight jobs
- Context-aware (supports cancellation & timeouts)



## How to Run the Code

### 1. Clone the repository

```bash
git clone <repository-url>
cd <repository-root>

go run ./cmd/server
```

The server listens on a port (e.g., :8080) and exposes endpoints:
```bash
GET /api/financials?companyId=<companyid>
GET /api/sales?companyId=<companyid>
GET /api/employee?companyId=<companyid>
```

## How to Run Tests

From the project root (where `go.mod` exists):

```bash
go test ./internal/jobs
```

