from opentelemetry.trace import get_tracer_provider

from examples.opentelemetry_instrumentation.langfuse.client import openai
from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

# > Task
HatchetInstrumentor(
    ## Langfuse sets the global tracer provider
    tracer_provider=get_tracer_provider(),
).instrument()

hatchet = Hatchet()


@hatchet.task()
async def langfuse_task(input: EmptyModel, ctx: Context) -> dict[str, str | None]:
    ## Usage, cost, etc. of this call will be send to Langfuse
    generation = await openai.chat.completions.create(
        model="gpt-4o-mini",
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Where does Anna Karenina take place?"},
        ],
    )

    location = generation.choices[0].message.content

    return {
        "location": location,
    }




def main() -> None:
    worker = hatchet.worker("langfuse-example-worker", workflows=[langfuse_task])
    worker.start()


if __name__ == "__main__":
    main()
