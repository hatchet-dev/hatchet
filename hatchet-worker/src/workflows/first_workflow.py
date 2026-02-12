import logging
import random
import time

from hatchet_sdk import Context, EmptyModel, Hatchet

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)

hatchet = Hatchet(debug=True)
logger = logging.getLogger(__name__)


RESET = "\033[0m"
GREEN = "\033[32m"
YELLOW = "\033[33m"
RED = "\033[31m"

REALISTIC_INFO_LINES = [
    "Initializing workflow run context for task_id={task_id}",
    "Loading configuration from environment: region=us-east-1, pool_size=10",
    "Established database connection to postgres://hatchet-db:5432/production",
    "Fetching batch records from queue: offset={offset}, limit=50",
    "Successfully deserialized {count} records from input payload",
    "Dispatching task to worker pool: concurrency_group=default, priority=normal",
    "Acquired distributed lock on resource key=user:{user_id}:billing",
    "Cache hit for key=workflow_config:v3, ttl_remaining=245s",
    "Starting data transformation pipeline: stage=1/3 (validation)",
    "Validated schema for {count} records, 0 violations found",
    "Enrichment step complete: resolved {count} external references via API",
    "Wrote {count} records to staging table tmp_batch_{batch_id}",
    "Committed transaction txn_{txn_id}: {count} rows affected",
    "Published event workflow.step.completed to channel=run-{run_id}",
    "Uploaded result artifact to s3://hatchet-artifacts/runs/{run_id}/output.json ({size}KB)",
    "Flushing metrics buffer: processed={count}, latency_p99=42ms",
    "Task completed successfully in {duration}ms, releasing lock",
    "Notifying downstream subscribers: webhook_count=2, email_count=1",
    "Checkpointing workflow state: step=3/5, status=IN_PROGRESS",
    "Heartbeat sent to orchestrator: worker_id=w-{worker_id}, load=0.{load}",
]

REALISTIC_WARNING_LINES = [
    "Slow query detected: SELECT * FROM events WHERE tenant_id=$1 took {duration}ms (threshold: 500ms)",
    "Rate limit approaching for external API (Stripe): 85/100 requests in current window",
    "Connection pool utilization at 80%: active=8, idle=2, max=10",
    "Retry scheduled for upstream dependency call to payments-service: attempt {attempt}/3",
    "Stale cache entry detected for key=tenant_config:{tenant_id}, forcing refresh",
    "Memory usage elevated: heap_alloc=487MB, gc_pause=12ms, threshold=512MB",
    "Request latency degraded: p95=320ms (SLA target: 200ms) for endpoint /api/v1/tasks",
    "Deprecated field 'callback_url' used in workflow input — migrate to 'webhook_config'",
    "Task execution time nearing timeout: elapsed=25s, limit=30s",
    "DNS resolution slow for payments-service.internal: 340ms (expected <50ms)",
    "Disk I/O wait elevated on worker node: await=45ms, queue_depth=12",
    "Certificate for upstream TLS connection expires in 7 days: cn=*.hatchet.internal",
]

REALISTIC_ERROR_LINES = [
    "Failed to connect to Redis cache at redis://cache:6379 — ECONNREFUSED, will retry with backoff",
    "Unhandled exception in task handler: KeyError: 'billing_account_id' missing from input payload",
    "Database query failed: deadlock detected on table 'workflow_runs' — transaction rolled back",
    "HTTP 503 from upstream service payments-service: 'Service Unavailable', circuit breaker OPEN",
    "Serialization error: Cannot encode NoneType for field 'completed_at' in RunResult schema",
    "Permission denied: service account lacks 'storage.objects.create' on bucket hatchet-artifacts",
    "Timeout exceeded waiting for distributed lock: key=user:9281:billing, waited=15000ms, limit=10000ms",
    "Message publish failed: Kafka broker at kafka:9092 not available — buffering 23 pending events",
    "Task input validation failed: field 'amount' expected Decimal, got str ('not_a_number')",
    "Out of memory during batch processing: attempted to allocate 1.2GB for record set, limit=1GB",
    "SSL handshake failed with upstream api.stripe.com: certificate verify failed (CERT_HAS_EXPIRED)",
    "Webhook delivery failed after 3 attempts: POST https://customer.app/hooks/hatchet returned HTTP 502",
]


@hatchet.task(retries=3)
def my_task(input: EmptyModel, ctx: Context) -> None:
    run_id = f"{random.randint(1000, 9999):04d}"
    worker_id = f"{random.randint(100, 999)}"

    logger.info(
        f"{GREEN}Starting task execution: "
        f"retry_count={ctx.retry_count}, "
        f"attempt={ctx.attempt_number}, "
        f"run_id=run-{run_id}{RESET}"
    )

    # On first attempt (retry_count == 0), simulate a failure with a realistic stack trace
    if ctx.retry_count == 0:
        logger.info(f"{GREEN}Fetching batch records from queue: offset=0, limit=50{RESET}")
        logger.info(f"{GREEN}Successfully deserialized 50 records from input payload{RESET}")
        logger.warning(
            f"{YELLOW}Slow query detected: SELECT * FROM events WHERE tenant_id=$1 "
            f"took 1243ms (threshold: 500ms){RESET}"
        )
        logger.error(
            f"{RED}Database query failed: deadlock detected on table 'workflow_runs' "
            f"— transaction rolled back{RESET}"
        )
        raise Exception(
            "DatabaseError: deadlock detected on table 'workflow_runs' "
            "while processing batch insert for run-"
            + run_id
            + ". "
            "DETAIL: Process 1234 waits for ShareLock on transaction 5678; "
            "blocked by process 9012. "
            "Process 9012 waits for ShareLock on transaction 3456; "
            "blocked by process 1234. "
            "HINT: See server log for query details."
        )

    # On subsequent retries, simulate successful processing with realistic logs
    for i in range(50):
        template = random.choice(REALISTIC_INFO_LINES)
        log_line = template.format(
            task_id=f"t-{random.randint(1000,9999)}",
            offset=i * 50,
            count=random.randint(10, 200),
            user_id=random.randint(1000, 9999),
            batch_id=random.randint(100, 999),
            txn_id=f"{random.randint(10000,99999):05d}",
            run_id=run_id,
            size=random.randint(1, 500),
            duration=random.randint(50, 3000),
            worker_id=worker_id,
            load=random.randint(10, 85),
            tenant_id=f"tn-{random.randint(100,999)}",
            attempt=ctx.attempt_number,
        )

        # Mix in warnings and errors at realistic intervals
        if random.random() < 0.08:
            warn_template = random.choice(REALISTIC_WARNING_LINES)
            warn_line = warn_template.format(
                duration=random.randint(500, 2000),
                attempt=ctx.attempt_number,
                tenant_id=f"tn-{random.randint(100,999)}",
            )
            logger.warning(f"{YELLOW}{warn_line}{RESET}")
        elif random.random() < 0.03:
            err_line = random.choice(REALISTIC_ERROR_LINES)
            logger.error(f"{RED}{err_line}{RESET}")
        else:
            logger.info(f"{GREEN}{log_line}{RESET}")

        # Simulate some work
        time.sleep(random.uniform(0.01, 0.05))

    logger.info(
        f"{GREEN}Task completed successfully: "
        f"run_id=run-{run_id}, "
        f"attempt={ctx.attempt_number}, "
        f"records_processed=50, "
        f"duration={random.randint(800, 3500)}ms{RESET}"
    )

    return None


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[my_task])
    worker.start()

if __name__ == "__main__":
    main()
