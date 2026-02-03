from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, DefaultFilter, Hatchet

hatchet = Hatchet(debug=True)


class WebhookInputWithScope(BaseModel):
    type: str
    message: str
    scope: str | None = None


class WebhookInputWithStaticPayload(BaseModel):
    type: str
    message: str
    source: str | None = None
    environment: str | None = None
    metadata: dict[str, Any] | None = None
    tags: list[str] | None = None
    customer_id: str | None = None
    processed: bool | None = None


@hatchet.task(
    input_validator=WebhookInputWithScope,
    on_events=["webhook-scope:test"],
    default_filters=[
        DefaultFilter(
            expression="true",
            scope="test-scope-value",
            payload={},
        )
    ],
)
def webhook_with_scope(input: WebhookInputWithScope, ctx: Context) -> dict[str, Any]:
    return input.model_dump()


@hatchet.task(
    input_validator=WebhookInputWithStaticPayload,
    on_events=["webhook-static:test"],
)
def webhook_with_static_payload(input: WebhookInputWithStaticPayload, ctx: Context) -> dict[str, Any]:
    return input.model_dump()


def main() -> None:
    worker = hatchet.worker(
        "webhook-scope-worker",
        workflows=[
            webhook_with_scope,
            webhook_with_static_payload,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
