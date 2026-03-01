from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet(debug=True)


# > Step 01 Define Webhook Task
class WebhookPayload(BaseModel):
    event_id: str
    type: str
    data: dict


@hatchet.task(
    input_validator=WebhookPayload,
    on_events=["webhook:stripe", "webhook:github"],
)
def process_webhook(input: WebhookPayload, ctx: Context) -> dict:
    """Process webhook payload. Hatchet acknowledges immediately, processes async."""
    return {"processed": input.event_id, "type": input.type}




# > Step 02 Register Webhook
def forward_webhook_to_hatchet(event_key: str, payload: dict) -> None:
    """Call this from your webhook endpoint to trigger the task."""
    hatchet.event.push(event_key, payload)
# forward_webhook_to_hatchet("webhook:stripe", {"event_id": "evt_123", "type": "payment", "data": {...}})


# > Step 03 Process Payload
def _validate_and_process(input: WebhookPayload) -> dict:
    if not input.event_id:
        raise ValueError("event_id required for deduplication")
    return {"processed": input.event_id, "type": input.type}


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "webhook-worker",
        workflows=[process_webhook],
    )
    worker.start()


if __name__ == "__main__":
    main()
