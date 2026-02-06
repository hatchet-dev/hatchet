import logging

from hatchet_sdk import Context, EmptyModel, Hatchet

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)

hatchet = Hatchet(debug=True)
logger = logging.getLogger(__name__)


RESET = "\033[0m"
GREEN = "\033[32m"
YELLOW = "\033[33m"
RED = "\033[31m"


@hatchet.task()
def my_task(input: EmptyModel, ctx: Context) -> None:
    for i in range(500):
        jibberish = "a" * (i // 5)
        log_line = f"processing item {i} with appended jibberish {jibberish}"
        if i % 3 == 0:
            logger.info(f"{GREEN}{log_line}{RESET}")
        elif i % 3 == 1:
            logger.warning(f"{YELLOW}{log_line}{RESET}")
        else:
            logger.error(f"{RED}{log_line}{RESET}")

    return None


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[my_task])
    worker.start()

if __name__ == "__main__":
    main()