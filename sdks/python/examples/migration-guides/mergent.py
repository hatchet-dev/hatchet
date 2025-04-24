from typing import List, Dict
from pydantic import BaseModel
import requests
from datetime import datetime, timedelta
from .hatchet_client import hatchet

async def process_image(image_url: str, filters: List[str]) -> Dict:
    # Do some image processing
    return {"url": image_url, "size": 100, "format": "png"}

# ❓ Before (Mergent)
async def process_image_task(request):
    image_url = request.json["image_url"]
    filters = request.json["filters"]
    try:
        result = await process_image(image_url, filters)
        return {"success": True, "processed_url": result["url"]}
    except Exception as e:
        print(f"Image processing failed: {e}")
        raise
# !!

# ❓ After (Hatchet)
class ImageProcessInput(BaseModel):
    image_url: str
    filters: List[str]

class ImageProcessOutput(BaseModel):
    processed_url: str
    metadata: Dict[str, any]

@hatchet.task(
    name="image-processor",
    retries=3,
    execution_timeout="10m"
)
async def image_processor(input: ImageProcessInput) -> ImageProcessOutput:
    # Do some image processing
    result = await process_image(input.image_url, input.filters)

    if not result["url"]:
        raise ValueError("Processing failed to generate URL")

    return ImageProcessOutput(
        processed_url=result["url"],
        metadata={
            "size": result["size"],
            "format": result["format"],
            "applied_filters": input.filters
        }
    )
# !!

async def run():
    # ❓ Running a task (Mergent)
    options = {
        "headers": {
            "Authorization": "Bearer <token>",
            "Content-Type": "application/json"
        },
        "json": {
            "name": "4cf95241-fa19-47ef-8a67-71e483747649",
            "queue": "default",
            "request": {
                "url": "https://example.com",
                "headers": {
                    "Authorization": "fake-secret-token",
                    "Content-Type": "application/json"
                },
                "body": "Hello, world!"
            }
        }
    }

    try:
        response = requests.post("https://api.mergent.co/v2/tasks", **options)
        print(response.json())
    except Exception as e:
        print(f"Error: {e}")
    # !!

    # ❓ Running a task (Hatchet)
    result = await image_processor.run(ImageProcessInput(
        image_url="https://example.com/image.png",
        filters=["blur"]
    ))

    # you can await fully typed results
    print(result)
    # !!

async def schedule():
    # ❓ Scheduling tasks (Mergent)
    options = {
        # same options as before
        "json": {
            # same body as before
            "delay": "5m"
        }
    }
    # !!

    print(options)

    # ❓ Scheduling tasks (Hatchet)
    # Schedule the task to run at a specific time
    run_at = datetime.now() + timedelta(days=1)
    await image_processor.schedule(
        run_at,
        ImageProcessInput(
            image_url="https://example.com/image.png",
            filters=["blur"]
        )
    )

    # Schedule the task to run every hour
    await image_processor.cron(
        "run-hourly",
        "0 * * * *",
        ImageProcessInput(
            image_url="https://example.com/image.png",
            filters=["blur"]
        )
    )
    # !! 