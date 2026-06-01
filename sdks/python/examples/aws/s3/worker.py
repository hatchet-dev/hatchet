"""
Example: concurrently process S3 data.

Prerequisites:
    cd sdks/python/examples/aws/s3
    docker compose up -d

Then run this worker:
    poetry run python -m examples.aws.s3.worker
"""

import os
from io import BytesIO
from typing import Any

import boto3  # type: ignore
from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    EmptyModel,
    Hatchet,
)

# > Client Setup

s3 = boto3.client("s3")

# !!

BUCKET_PREFIX = os.getenv("S3_WORKER_BUCKET_PREFIX", "bucket-")
MAX_CONCURRENT_BUCKET_POLLERS = int(
    os.getenv("S3_WORKER_MAX_CONCURRENT_BUCKET_POLLERS", "10")
)
MAX_RUNS_PER_BUCKET = int(os.getenv("S3_WORKER_MAX_RUNS_PER_BUCKET", "20"))
SLOTS = int(os.getenv("S3_WORKER_SLOTS", "40"))

hatchet = Hatchet()


# > Models
class ListObjectsInput(BaseModel):
    bucket: str


class ProcessObjectInput(BaseModel):
    bucket: str
    key: str


# !!


# > Fetch S3 Buckets

fetch_buckets_workflow = hatchet.workflow(
    name="fetch_s3_buckets",
    on_crons=["* * * * *"],
    concurrency=ConcurrencyExpression(
        expression="'singleton'",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_NEWEST,
    ),
)

# !!

# > Fetch S3 Objects

fetch_objects_workflow = hatchet.workflow(
    name="fetch_s3_objects",
    input_validator=ListObjectsInput,
    concurrency=[
        ConcurrencyExpression(
            expression="input.bucket",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.CANCEL_NEWEST,
        ),
        ConcurrencyExpression.from_int(MAX_CONCURRENT_BUCKET_POLLERS),
    ],
)

# !!

# > Process S3 Objects

process_object_workflow = hatchet.workflow(
    name="process_object",
    input_validator=ProcessObjectInput,
    concurrency=ConcurrencyExpression(
        expression="input.bucket",
        max_runs=MAX_RUNS_PER_BUCKET,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
)

# !!


# > Fetch S3 Buckets Task


@fetch_buckets_workflow.task()
async def fetch_buckets(input: EmptyModel, ctx: Context) -> dict[str, Any]:
    paginator = s3.get_paginator("list_buckets")
    pages = paginator.paginate(
        Prefix=BUCKET_PREFIX,
        PaginationConfig={"PageSize": 10},
    )

    for page in pages:
        items = [
            fetch_objects_workflow.create_bulk_run_item(
                input=ListObjectsInput(bucket=b["Name"]),
                child_key=b["Name"],
                additional_metadata={"bucket-name": b["Name"]},
            )
            for b in page.get("Buckets", [])
        ]
        if items:
            await fetch_objects_workflow.aio_run_many(items, wait_for_result=False)

    return {}


# !!


# > Fetch S3 Objects Task


@fetch_objects_workflow.task()
async def fetch_objects(input: ListObjectsInput, ctx: Context) -> dict[str, Any]:
    paginator = s3.get_paginator("list_objects_v2")
    pages = paginator.paginate(
        Bucket=input.bucket,
        PaginationConfig={"PageSize": 100},
    )

    for page in pages:
        items = [
            process_object_workflow.create_bulk_run_item(
                input=ProcessObjectInput(bucket=input.bucket, key=obj["Key"]),
                child_key=f"{input.bucket}/{obj['Key']}",
            )
            for obj in page.get("Contents", [])
        ]
        if items:
            await process_object_workflow.aio_run_many(items, wait_for_result=False)

    return {}


# !!

# > Download and Process S3 Objects Task


@process_object_workflow.task()
def process_object(input: ProcessObjectInput, ctx: Context) -> dict[str, Any]:
    buf = BytesIO()
    try:
        s3.download_fileobj(input.bucket, input.key, buf)
    except s3.exceptions.ClientError as e:
        if (code := e.response["Error"]["Code"]) in {
            "NoSuchKey",
            "NoSuchBucket",
            "404",
        }:
            ctx.log(f"skipping {input.bucket}/{input.key}: not found ({code})")
            return {}
        raise

    buf.seek(0)
    # TODO: actual image processing here

    s3.delete_object(Bucket=input.bucket, Key=input.key)
    return {}


# !!


def main() -> None:
    worker = hatchet.worker(
        f"s3-worker-{os.getpid()}",
        slots=SLOTS,
        workflows=[
            fetch_buckets_workflow,
            fetch_objects_workflow,
            process_object_workflow,
        ],
    )

    worker.start()


if __name__ == "__main__":
    main()
