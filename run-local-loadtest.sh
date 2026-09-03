export HATCHET_CLIENT_TLS_STRATEGY=none
export HATCHET_CLIENT_TOKEN="eyJhbGciOiJFUzI1NiIsICJraWQiOiI3N0xwMkEifQ.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCAiZXhwIjo0OTQwNjcyNjE2LCAiZ3JwY19icm9hZGNhc3RfYWRkcmVzcyI6IjEyNy4wLjAuMTo3MDcwIiwgImlhdCI6MTc4NzA3MjYxNiwgImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3Q6ODA4MCIsICJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwgInN1YiI6Ijk4OTA5YmRhLWQ4MzYtNDZjMC05ZmZjLWRiNDEwODg1NzdmMSIsICJ0b2tlbl9pZCI6ImJkNzhjZTMzLWYzOTUtNDhkZS1iZjIwLTZhN2U5YzhmNDNjMyJ9.LJYeq2_2Zp-UZSgbq0ga5sgLXycv-WajEQlRXoafdaCRmsnySI8eajFBc8v59fk9buqmh6M6s-nBDoc8QmYxFg"
export HATCHET_LOADTEST_DURABLE_CHILDREN=1
export HATCHET_LOADTEST_DURABLE_CHILD_DURATION_MS=1
export HATCHET_LOADTEST_DURABLE_SLOTS=1000
export HATCHET_LOADTEST_SLOTS=1000
go run ./cmd/hatchet-loadtest/go


export HATCHET_CLIENT_TLS_STRATEGY=none
export HATCHET_CLIENT_TOKEN="eyJhbGciOiJFUzI1NiIsICJraWQiOiI3N0xwMkEifQ.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCAiZXhwIjo0OTQwNjcyNjE2LCAiZ3JwY19icm9hZGNhc3RfYWRkcmVzcyI6IjEyNy4wLjAuMTo3MDcwIiwgImlhdCI6MTc4NzA3MjYxNiwgImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3Q6ODA4MCIsICJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwgInN1YiI6Ijk4OTA5YmRhLWQ4MzYtNDZjMC05ZmZjLWRiNDEwODg1NzdmMSIsICJ0b2tlbl9pZCI6ImJkNzhjZTMzLWYzOTUtNDhkZS1iZjIwLTZhN2U5YzhmNDNjMyJ9.LJYeq2_2Zp-UZSgbq0ga5sgLXycv-WajEQlRXoafdaCRmsnySI8eajFBc8v59fk9buqmh6M6s-nBDoc8QmYxFg"

go run ./cmd/hatchet-loadtest loadtest \
  --select durable \
  --externalWorker \
  --duration 60s \
  --events 250 \
  --averageDurationThreshold 200ms \
  --registrationTimeout 300s \
  --emitWorkers 100


#                        queued n=10041, scheduling n=10041, execution n=10041)
# 2026/08/10 16:40:37 ℹ️ overall engine timing (n=10041):
# 2026/08/10 16:40:37 ℹ️   final average queued time per event: 23.844986ms
# 2026/08/10 16:40:37 ℹ️   final average scheduling time per event: 35.355648ms
# 2026/08/10 16:40:37 ℹ️   final average duration per executed event: 145.807019ms
# 2026/08/10 16:40:37 ℹ️ engine timing for load-test:durable-event (n=10041):
# 2026/08/10 16:40:37 ℹ️   queued=23.844986ms scheduling=35.355648ms execution=145.807019ms
# 2026/08/10 16:40:37 ℹ️   scheduling (push) latency: avg=6.220082347s n=10041
# 2026/08/10 16:40:37 ℹ️ not all environment vars for sending plots to Slack enabled...skipping
# 2026/08/10 16:40:37 ✅ success
# matt@Matts-MacBook-Pro-4 hatchet % go run ./cmd/hatchet-loadtest loadtest \

# 2026/08/10 16:54:48 ℹ️ pushed 29793 "load-test:durable-event" events, using 500 events/s (externalWorker: engine-observed samples, 100% sampled — queued n=29793, scheduling n=29793, execution n=29793)
# 2026/08/10 16:54:48 ℹ️ overall engine timing (n=29793):
# 2026/08/10 16:54:48 ℹ️   final average queued time per event: 38.828811ms
# 2026/08/10 16:54:48 ℹ️   final average scheduling time per event: 45.925263352s
# 2026/08/10 16:54:48 ℹ️   final average duration per executed event: 1.829902859s
# 2026/08/10 16:54:48 ℹ️ engine timing for load-test:durable-event (n=29793):
# 2026/08/10 16:54:48 ℹ️   queued=38.828811ms scheduling=45.925263352s execution=1.829902859s
# 2026/08/10 16:54:48 ℹ️   scheduling (push) latency: avg=10.738925ms n=29793
# 2026/08/10 16:54:48 ℹ️ not all environment vars for sending plots to Slack enabled...skipping
# 2026/08/10 16:54:48 ✅ success
