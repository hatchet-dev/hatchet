import logging

from hatchet_sdk import ClientConfig, Hatchet

logging.basicConfig(level=logging.INFO)

hatchet = Hatchet(
    debug=True,
    config=ClientConfig(
        logger=logging.getLogger(),
    ),
)
