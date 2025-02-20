from dotenv import load_dotenv

from hatchet_sdk.hatchet import HatchetRest

load_dotenv()

hatchet = HatchetRest()


def main() -> None:
    # Look up the failed workflow runs
    run = hatchet.rest.workflow_run_create("19528945-17df-48df-88f4-72d650ce7cae", {})
    print(run)


if __name__ == "__main__":
    main()
