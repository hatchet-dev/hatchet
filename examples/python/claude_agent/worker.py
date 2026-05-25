from examples.claude_agent.tools import (
    hatchet,
    lookup_customer,
    check_order_status,
    create_ticket,
)


def main() -> None:
    worker = hatchet.worker(
        "support-tools-worker",
        workflows=[lookup_customer, check_order_status, create_ticket],
    )
    worker.start()


if __name__ == "__main__":
    main()
