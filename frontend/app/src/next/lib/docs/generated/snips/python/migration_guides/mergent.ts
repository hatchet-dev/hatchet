import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from collections.abc import Mapping\nfrom datetime import datetime, timedelta, timezone\nfrom typing import Any\n\nimport requests\nfrom pydantic import BaseModel\nfrom requests import Response\n\nfrom hatchet_sdk.context.context import Context\n\nfrom .hatchet_client import hatchet\n\n\nasync def process_image(image_url: str, filters: list[str]) -> dict[str, Any]:\n    # Do some image processing\n    return {"url": image_url, "size": 100, "format": "png"}\n\n\n# > Before (Mergent)\nasync def process_image_task(request: Any) -> dict[str, Any]:\n    image_url = request.json["image_url"]\n    filters = request.json["filters"]\n    try:\n        result = await process_image(image_url, filters)\n        return {"success": True, "processed_url": result["url"]}\n    except Exception as e:\n        print(f"Image processing failed: {e}")\n        raise\n\n\n\n\n# > After (Hatchet)\nclass ImageProcessInput(BaseModel):\n    image_url: str\n    filters: list[str]\n\n\nclass ImageProcessOutput(BaseModel):\n    processed_url: str\n    metadata: dict[str, Any]\n\n\n@hatchet.task(\n    name="image-processor",\n    retries=3,\n    execution_timeout="10m",\n    input_validator=ImageProcessInput,\n)\nasync def image_processor(input: ImageProcessInput, ctx: Context) -> ImageProcessOutput:\n    # Do some image processing\n    result = await process_image(input.image_url, input.filters)\n\n    if not result["url"]:\n        raise ValueError("Processing failed to generate URL")\n\n    return ImageProcessOutput(\n        processed_url=result["url"],\n        metadata={\n            "size": result["size"],\n            "format": result["format"],\n            "applied_filters": input.filters,\n        },\n    )\n\n\n\n\nasync def run() -> None:\n    # > Running a task (Mergent)\n    headers: Mapping[str, str] = {\n        "Authorization": "Bearer <token>",\n        "Content-Type": "application/json",\n    }\n\n    task_data = {\n        "name": "4cf95241-fa19-47ef-8a67-71e483747649",\n        "queue": "default",\n        "request": {\n            "url": "https://example.com",\n            "headers": {\n                "Authorization": "fake-secret-token",\n                "Content-Type": "application/json",\n            },\n            "body": "Hello, world!",\n        },\n    }\n\n    try:\n        response: Response = requests.post(\n            "https://api.mergent.co/v2/tasks",\n            headers=headers,\n            json=task_data,\n        )\n        print(response.json())\n    except Exception as e:\n        print(f"Error: {e}")\n\n    # > Running a task (Hatchet)\n    result = await image_processor.aio_run(\n        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"])\n    )\n\n    # you can await fully typed results\n    print(result)\n\n\nasync def schedule() -> None:\n    # > Scheduling tasks (Mergent)\n    options = {\n        # same options as before\n        "json": {\n            # same body as before\n            "delay": "5m"\n        }\n    }\n\n    print(options)\n\n    # > Scheduling tasks (Hatchet)\n    # Schedule the task to run at a specific time\n    run_at = datetime.now(tz=timezone.utc) + timedelta(days=1)\n    await image_processor.aio_schedule(\n        run_at,\n        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"]),\n    )\n\n    # Schedule the task to run every hour\n    await image_processor.aio_create_cron(\n        "run-hourly",\n        "0 * * * *",\n        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"]),\n    )\n',
  source: 'out/python/migration_guides/mergent.py',
  blocks: {
    before_mergent: {
      start: 20,
      stop: 30,
    },
    after_hatchet: {
      start: 34,
      stop: 66,
    },
    running_a_task_mergent: {
      start: 71,
      stop: 97,
    },
    running_a_task_hatchet: {
      start: 100,
      stop: 105,
    },
    scheduling_tasks_mergent: {
      start: 110,
      stop: 116,
    },
    scheduling_tasks_hatchet: {
      start: 121,
      stop: 133,
    },
  },
  highlights: {},
};

export default snippet;
