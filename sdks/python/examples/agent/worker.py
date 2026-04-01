from examples.agent.workflows import hatchet, get_temperature_workflow, get_temperature_standalone

def main() -> None:
    worker = hatchet.worker(
        "test-worker", workflows=[get_temperature_workflow, get_temperature_standalone]
    )
    worker.start()


if __name__ == "__main__":
    main()
