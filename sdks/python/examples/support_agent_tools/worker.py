from examples.support_agent_tools.tools import (
    check_order_status,
    create_ticket,
    hatchet,
    lookup_customer,
)


def main() -> None:
    worker = hatchet.worker(
        "support-tools-worker",
        workflows=[lookup_customer, check_order_status, create_ticket],
    )
    worker.start()


if __name__ == "__main__":
    main()
