import os
import time
from pathlib import Path
from subprocess import Popen, run
import boto3  # type: ignore
from typing import Any, Callable

import pytest

from examples.aws.s3.worker import fetch_buckets_workflow

# Set variables running worker against LocalStack instance
os.environ.setdefault("AWS_ENDPOINT_URL", "http://localhost:4566")
os.environ.setdefault("AWS_ACCESS_KEY_ID", "test")
os.environ.setdefault("AWS_SECRET_ACCESS_KEY", "test")
os.environ.setdefault("AWS_REGION", "us-east-1")

TEST_BUCKETS = [f"bucket-{i}" for i in range(3)]
COMPOSE_DIR = Path(__file__).parent

s3_client = boto3.client("s3")


def wait_until(fn: Callable[[], Any], timeout: float, period: float = 0.25) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        if fn():
            return True
        time.sleep(period)
    return False


def _count_objects(bucket: str) -> int:
    paginator = s3_client.get_paginator("list_objects_v2")
    return sum(page.get("KeyCount", 0) for page in paginator.paginate(Bucket=bucket))


@pytest.fixture(scope="session", autouse=True)
def setup() -> Any:
    try:
        run(["docker", "compose", "up", "-d", "--wait"], cwd=COMPOSE_DIR, check=True)

        # TODO: This should be extracted into its own fixture to determine the
        #  bucket + object count at runtime.
        for bucket in TEST_BUCKETS:
            s3_client.create_bucket(Bucket=bucket)
            for n in range(3):
                s3_client.put_object(
                    Bucket=bucket,
                    Key=f"doc-{n}.txt",
                    Body="a" * 128,
                )

        ready = wait_until(
            lambda: all(_count_objects(b) == 3 for b in TEST_BUCKETS),
            timeout=60,
        )
        assert ready, "LocalStack seed never reached 3 objects per bucket"

        yield
    finally:
        run(["docker", "compose", "down", "--volumes"], cwd=COMPOSE_DIR)


@pytest.mark.parametrize(
    "on_demand_worker",
    [["poetry", "run", "python", "examples/aws/s3/worker.py"]],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_pipeline_processes_and_deletes_all_objects(
    on_demand_worker: Popen[Any],
) -> None:
    # NOTE: The 'setup' fixture will automatically populate S3 with some data.
    await fetch_buckets_workflow.aio_run(wait_for_result=False)
    ready = wait_until(
        lambda: all(_count_objects(b) == 0 for b in TEST_BUCKETS),
        timeout=60,
    )
    assert ready, "Expected all buckets to be drained."
