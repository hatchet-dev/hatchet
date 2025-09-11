# > Webhooks

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet(debug=True)


class WebhookInput(BaseModel):
    type: str
    message: str


@hatchet.task(input_validator=WebhookInput, on_events=["webhook:test"])
def webhook(input: WebhookInput, ctx: Context) -> dict[str, str]:
    return input.model_dump()


def main() -> None:
    worker = hatchet.worker("webhook-worker", workflows=[webhook])
    worker.start()



if __name__ == "__main__":
    main()
