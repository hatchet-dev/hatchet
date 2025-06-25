# > LoggingWorkflow

import logging
import time

from examples.logger.client import hatchet
from hatchet_sdk import Context, EmptyModel

logger = logging.getLogger(__name__)

logging_workflow = hatchet.workflow(
    name="LoggingWorkflow",
)


@logging_workflow.task()
def root_logger(input: EmptyModel, ctx: Context) -> dict[str, str]:
    for i in range(12):
        logger.info(f"executed step1 - {i}")
        logger.info({"step1": "step1"})

        time.sleep(0.1)

    return {"status": "success"}


# !!

# > ContextLogger


@logging_workflow.task()
def context_logger(input: EmptyModel, ctx: Context) -> dict[str, str]:
    for i in range(12):
        ctx.log(f"executed step1 - {i}")
        ctx.log({"step1": "step1"})

        time.sleep(0.1)

    return {"status": "success"}


# !!
