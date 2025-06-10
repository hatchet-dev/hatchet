from datetime import datetime, timedelta, timezone
from typing import Any, Dict, List, Mapping

import requests
from pydantic import BaseModel
from requests import Response

from hatchet_sdk.context.context import Context

from .hatchet_client import hatchet


async def process_image(image_url: str, filters: List[str]) -> Dict[str, Any]:
    # Do some image processing
    return {"url": image_url, "size": 100, "format": "png"}


# > Before (Mergent)
async def process_image_task(request: Any) -> Dict[str, Any]:
    image_url = request.json["image_url"]
    filters = request.json["filters"]
    try:
        result = await process_image(image_url, filters)
        return {"success": True, "processed_url": result["url"]}
    except Exception as e:
        print(f"Image processing failed: {e}")
        raise




# > After (Hatchet)
class ImageProcessInput(BaseModel):
    image_url: str
    filters: List[str]


class ImageProcessOutput(BaseModel):
    processed_url: str
    metadata: Dict[str, Any]


@hatchet.task(
    name="image-processor",
    retries=3,
    execution_timeout="10m",
    input_validator=ImageProcessInput,
)
async def image_processor(input: ImageProcessInput, ctx: Context) -> ImageProcessOutput:
    # Do some image processing
    result = await process_image(input.image_url, input.filters)

    if not result["url"]:
        raise ValueError("Processing failed to generate URL")

    return ImageProcessOutput(
        processed_url=result["url"],
        metadata={
            "size": result["size"],
            "format": result["format"],
            "applied_filters": input.filters,
        },
    )




async def run() -> None:
    # > Running a task (Mergent)
    headers: Mapping[str, str] = {
        "Authorization": "Bearer <token>",
        "Content-Type": "application/json",
    }

    task_data = {
        "name": "4cf95241-fa19-47ef-8a67-71e483747649",
        "queue": "default",
        "request": {
            "url": "https://example.com",
            "headers": {
                "Authorization": "fake-secret-token",
                "Content-Type": "application/json",
            },
            "body": "Hello, world!",
        },
    }

    try:
        response: Response = requests.post(
            "https://api.mergent.co/v2/tasks",
            headers=headers,
            json=task_data,
        )
        print(response.json())
    except Exception as e:
        print(f"Error: {e}")

    # > Running a task (Hatchet)
    result = await image_processor.aio_run(
        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"])
    )

    # you can await fully typed results
    print(result)


async def schedule() -> None:
    # > Scheduling tasks (Mergent)
    options = {
        # same options as before
        "json": {
            # same body as before
            "delay": "5m"
        }
    }

    print(options)

    # > Scheduling tasks (Hatchet)
    # Schedule the task to run at a specific time
    run_at = datetime.now(tz=timezone.utc) + timedelta(days=1)
    await image_processor.aio_schedule(
        run_at,
        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"]),
    )

    # Schedule the task to run every hour
    await image_processor.aio_create_cron(
        "run-hourly",
        "0 * * * *",
        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"]),
    )
