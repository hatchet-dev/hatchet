# > RootLogger


import logging

from hatchet_sdk import ClientConfig, Hatchet

logging.basicConfig(level=logging.INFO)

root_logger = logging.getLogger()

hatchet = Hatchet(
    debug=True,
    config=ClientConfig(
        logger=root_logger,
    ),
)


