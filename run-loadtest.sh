HATCHET_CLIENT_TOKEN="eyJhbGciOiJFUzI1NiIsICJraWQiOiJ0M0I0QXcifQ.eyJhdWQiOiJodHRwczovL3N0YWdpbmctY2hvbmt5LmhhdGNoZXQtdG9vbHMuY29tIiwgImV4cCI6NDkzOTcyNjI1OSwgImdycGNfYnJvYWRjYXN0X2FkZHJlc3MiOiJkdTI4ay5zdGFnaW5nLWNob25reS5oYXRjaGV0LXRvb2xzLmNvbTo0NDMiLCAiaWF0IjoxNzg2MTI2MjU5LCAiaXNzIjoiaHR0cHM6Ly9zdGFnaW5nLWNob25reS5oYXRjaGV0LXRvb2xzLmNvbSIsICJzZXJ2ZXJfdXJsIjoiaHR0cHM6Ly9zdGFnaW5nLWNob25reS5oYXRjaGV0LXRvb2xzLmNvbSIsICJzdWIiOiI5NzliNDUxZS03NDM5LTQ3NGQtYTlmYS1kZDY3Mzc4ZThlYjMiLCAidG9rZW5faWQiOiIzYjAxYjcxYi00Zjk3LTQ0M2MtOGM1YS05ODFjZWExYzc3ZmMifQ.cpEf6_k-8CycuUBAlDvXAFd-hCOsyCj7Xux0ic-E8Grz2FGSXY4Rljfq1vES91W7Q6JOcyF_e0RiUhCihOoxnA"
NAMESPACE="staging-chonky"
EVENTS_PER_SECOND="${10:-}"
WORKER_COUNT="${11:10}"
RANDOM_ID="loadtest-$(date +%s)-$($SRANDOM)-local"

WORKER_DEPLOYMENT_NAME="loadtest-worker"
WORKER_COUNT=50
cat <<EOF | kubectl replace -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${WORKER_DEPLOYMENT_NAME}
  namespace: ${NAMESPACE}
spec:
  replicas: ${WORKER_COUNT}
  selector:
    matchLabels:
      app: loadtest-worker
  template:
    metadata:
      labels:
        app: loadtest-worker
    spec:
      containers:
        - image: ghcr.io/hatchet-dev/hatchet/hatchet-loadtest:v0.105.20-alpha.0
          imagePullPolicy: Always
          name: loadtest-worker
          command: ["/hatchet/hatchet-load-test-worker"]
          env:
            - name: HATCHET_CLIENT_TOKEN
              value: ${HATCHET_CLIENT_TOKEN}
            - name: HATCHET_LOADTEST_DURABLE_CHILDREN
              value: "1"
            - name: HATCHET_LOADTEST_DURABLE_CHILD_DURATION_MS
              value: "1"
            - name: HATCHET_LOADTEST_DURABLE_SLOTS
              value: "500"
            - name: HATCHET_LOADTEST_SLOTS
              value: "500"
          resources:
            limits:
              memory: 256Mi
            requests:
              cpu: 250m
              memory: 256Mi
EOF

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: ${RANDOM_ID}
  namespace: ${NAMESPACE}
spec:
  restartPolicy: Never
  containers:
    - image: ghcr.io/hatchet-dev/hatchet/hatchet-loadtest:v0.105.20-alpha.0
      imagePullPolicy: Always
      name: loadtest
      command: ["/hatchet/hatchet-load-test"]
      args:
        - loadtest
        - --select
        - dag-shapes,dag-nested
        - --duration
        - "6h"
        - --events
        - "10"
        - --level
        - warn
        - --averageDurationThreshold
        - "200ms"
        - --externalWorker
        - --registrationTimeout
        - "300s"
        - --wait
        - "120s"
      env:
        - name: HATCHET_CLIENT_TOKEN
          value: ${HATCHET_CLIENT_TOKEN}
        - name: HATCHET_LOADTEST_ENVIRONMENT
          value: "${NAMESPACE}"
        - name: PPROF_ENABLED
          value: "true"
      resources:
        limits:
          memory: 512Mi
        requests:
          cpu: 500m
          memory: 512Mi
EOF



# # durable
# 2026/08/18 02:59:10 ℹ️ pushed per event key: {
#   "load-test:durable-event": 29893
# }
# 2026-08-18T03:00:47.095Z WRN timing collector: error listing workflow runs error="context canceled" service=loadtest
# 2026-08-18T03:00:47.095Z WRN timing collector: error listing workflow runs error="context canceled" service=loadtest
# 2026/08/18 03:00:47 ℹ️ pushed 29893 "load-test:durable-event" events, using 500 events/s (externalWorker: engine-observed samples, 100% sampled — queued n=29775, scheduling n=29775, execution n=29775)
# 2026/08/18 03:00:47 ℹ️ overall engine timing (n=29775):
# 2026/08/18 03:00:47 ℹ️   final average queued time per event: 76.088895ms
# 2026/08/18 03:00:47 ℹ️   final average scheduling time per event: 548.942138ms
# 2026/08/18 03:00:47 ℹ️   final average duration per executed event: 17.928824468s
# 2026/08/18 03:00:47 ℹ️ engine timing for load-test:durable-event (n=29775):
# 2026/08/18 03:00:47 ℹ️   queued=76.088895ms scheduling=548.942138ms execution=17.928824468s
# 2026/08/18 03:00:47 ℹ️   scheduling (push) latency: avg=17.664592ms n=29893
# 2026/08/18 03:00:47 ℹ️ not all environment vars for sending plots to Slack enabled...skipping
# 2026/08/18 03:00:47 ✅ success
