import logging
import time

from examples.logger.client import hatchet
from hatchet_sdk import Context

logger = logging.getLogger(__name__)


@hatchet.workflow()
class LoggingWorkflow:
    @hatchet.step()
    def step1(self, context: Context) -> dict[str, str]:
        for i in range(12):
            logger.info("executed step1 - {}".format(i))
            logger.info({"step1": "step1"})
            time.sleep(0.1)
        return {"status": "success"}
