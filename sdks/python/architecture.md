# CogniSim Test Execution – Hatchet-based Architecture

## 1. High-level Goal
Run an **interactive, long-lived test session** composed of:
1. One mandatory _device-setup_ step.
2. A stream of **N arbitrary test steps** issued by the browser in real-time.

Every step must execute on Hatchet workers and **stream its result back to the browser with sub-second latency**, while the backend enforces a **per-user concurrency cap**.

---

## 2. Logical Components
1. **Frontend (React/Vue/…​)** – renders the IDE & opens a WebSocket to the API.
2. **API Server (FastAPI)** – validates WS messages and bridges them to Hatchet:
   • pushes events with `hatchet.event.push()`  
   • listens on the run-event stream and forwards messages to the browser.
3. **Hatchet Workflow** – orchestrates the session on the worker side.
4. **Worker Process** – hosts CogniSim and executes tasks.

A simplified call-graph is shown below:
```
Browser ──WS──► API ──event.push──► Hatchet Dispatcher ─► Worker
   ▲                                        │  ▲
   └───────── listener.stream ◄─────────────┘  │
       JSON                                   │
       result ◄────────── task return ◄──────┘
```

---

## 3. Workflow Design
### 3.1 Tasks
| Name | Decorator | Purpose |
|------|-----------|---------|
| `setup_device_task` | `@hatchet.durable_task()` | Power-on emulator / physical device and build a **sticky execution key** (e.g., `user_id`).  Returns `device_session_id`, `iframe_url`, … |
| `step_router` | `@hatchet.durable_task()` | _Long-lived loop_ waiting for user events. Uses `ctx.aio_wait_for()` on **`step:<idx>` signals**. Spawns child tasks so that each user step is isolated. |
| `execute_step_task` | `@hatchet.task()` | Executes one CogniSim step (`cognisim_instance.run_step`). |

### 3.2 Workflow graph
```text
┌────────────────┐        spawn_child (≤cap)
│ setup_device   │ ─────► step_router ──┐
└────────────────┘                     │
     Sticky key = user_id              │
                                       ▼
                            execute_step_task (N)
```
* `setup_device_task` **must succeed** before anything else – it is the single parent of the router.
* `step_router` holds the CogniSim instance in memory and keeps the worker alive (durable).
* For each incoming WS message `step_router` **spawns** an `execute_step_task` allowing:
  • sequential execution (`await child_ref.aio_result()`), _or_
  • parallel execution if business rules permit.

### 3.3 Concurrency & rate-limit
Hatchet provides decorators for this:
```python
from hatchet_sdk import concurrency_limit

@device_workflow.concurrency_limit(
    key=lambda input, ctx: input.user_id,  # 1 key per user
    limit=settings.MAX_SESSIONS_PER_USER, # e.g. 2 parallel sessions
)
class device_workflow(HatchetWorkflow):
    ...
```
*Per-user* cap ⇒ you satisfy the business model.   
For per-step throttling use `@step_router.concurrency_limit(key=user_id, limit=1)` to force strict FIFO.

---

## 4. Backend Integration Pattern
1. **Start session** – REST `POST /sessions` triggers `setup_device_task.run()` and returns `workflow_run_id`.
2. **Open WS** – the browser connects with that `workflow_run_id`.
3. **Send step** – browser sends `{ action:"EXECUTE_SINGLE", step_index, … }`.
4. **API** calls:
```python
hatchet.event.push(
    event_key=f"step:{step_index}:{workflow_run_id}",
    payload=payload_dict,
)
```
5. **Router task** receives the durable event, spawns `execute_step_task`, awaits its result, then streams back a user-friendly JSON via `ctx.put_stream()`.
6. **API** listens on `hatchet.listener.stream(workflow_run_id)` and relays the message to the browser.

---

## 5. Reliability & Observability
• **Retries:** use Hatchet retry policies on `execute_step_task` (e.g., max 3).  
• **Timeouts:** set explicit timeouts per step.  
• **Structured logs & trace-ids:** include `workflow_run_id` & `step_id` via Hatchet's OpenTelemetry hooks.

---

## 6. Checklist – end-to-end implementation
### 6.1 Worker code
- [ ] Implement `setup_device_task` (returns `iframe_url`, stores CogniSim instance).
- [ ] Implement `step_router` durable loop with `ctx.aio_wait_for()`.
- [ ] Implement `execute_step_task` using `CogniSim.run_step()`.
- [ ] Apply `@concurrency_limit` on workflow/router as described.
- [ ] Add sticky assignment (`group_key`) so all tasks stay on same worker.
- [ ] Register worker with `hatchet = Hatchet()` and `hatchet.run()`.

### 6.2 API layer
- [ ] Endpoint `/sessions` – kicks off workflow, persists `workflow_run_id`.
- [ ] WebSocket handler – validate payload via Pydantic, then call `hatchet.event.push()`.
- [ ] Stream listener coroutine – multiplex `hatchet.listener.stream()` to the correct WS connection.
- [ ] Graceful shutdown handling with `listener.abort()`.

### 6.3 Frontend
- [ ] Manage WebSocket lifecycle & reconnect.
- [ ] Visual feedback for `STARTED`, `COMPLETED`, `ERROR` statuses.
- [ ] Allow "stop stream" command to close session.

### 6.4 Ops / Scaling
- [ ] Decide container size & machine type for workers (CPU/GPU).
- [ ] Auto-scale Hatchet managed compute or K8s deployment.
- [ ] Centralised log sink.

### 6.5 Security / Limits
- [ ] Validate `user_id` on every message.
- [ ] Enforce per-user run cap via Hatchet limits + DB checks.
- [ ] Sanitize step payloads (no arbitrary code).

---

## 7. References
* Hatchet docs – "Durable Execution", "Event Trigger", "Concurrency Limits"  
* Example repo – `examples/durable/worker.py`, `examples/waits/worker.py`. 