import os
import threading
import time

from hatchet_sdk import (
    ClientConfig,
    Context,
    EmbeddedHatchetConfig,
    EmptyModel,
    Hatchet,
)

# > Create an embedded client
hatchet = Hatchet.from_embedded()
# !!


class GreetInput(EmptyModel):
    name: str = "embed"


@hatchet.task(name="embedded-greet", input_validator=GreetInput)
def greet(input: GreetInput, ctx: Context) -> dict[str, str]:
    return {"greeting": f"Hello, {input.name}!"}


def configured_client() -> Hatchet:
    # > Configure the embedded engine
    hatchet = Hatchet.from_embedded(
        ClientConfig(
            embedded=EmbeddedHatchetConfig(
                # use your own Postgres instead of the bundled one
                database_url="postgres://...",
                # store the bundled Postgres runtime and data under this directory
                postgres_data_dir="~/my-project/.hatchet-pg",
                # use RabbitMQ instead of the Postgres message queue
                rabbitmq_url="amqp://...",
                # bind the API / gRPC servers to specific ports
                api_port=28243,
                grpc_port=7070,
                # start only the engine + gRPC, no REST API
                start_api=False,
                # skip running migrations on startup
                run_migrations=False,
                # engine log level (default "warn")
                log_level="info",
                # hatchet-embedded release tag to download
                version="v0.105.0",
                # use an existing sidecar binary, skips the download
                binary_path="/path/to/hatchet-embedded-sidecar",
                # pinned sha256 of the sidecar binary, replaces
                # checksums.txt as the trust anchor
                checksum="4f2a...",
            )
        )
    )
    # !!
    return hatchet


def fleet_client() -> Hatchet:
    # > Fleet with a shared database
    hatchet = Hatchet.from_embedded(
        ClientConfig(
            embedded=EmbeddedHatchetConfig(
                database_url="postgres://user:pass@db.internal:5432/hatchet"
            )
        )
    )
    # !!
    return hatchet


def main() -> None:
    worker = hatchet.worker("embedded-worker", workflows=[greet])
    threading.Thread(target=worker.start, daemon=True).start()
    time.sleep(2)

    result = greet.run(GreetInput(name="embed"))
    print(result["greeting"], flush=True)

    # > Stop the embedded engine
    hatchet.stop_embedded()
    # !!

    # the worker's subprocesses would otherwise keep the interpreter alive at exit
    os._exit(0)


if __name__ == "__main__":
    main()
