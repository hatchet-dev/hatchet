import json
import logging
import sys
import time

from dotenv import load_dotenv

from hatchet_sdk import ClientConfig, Hatchet

load_dotenv()

logging.basicConfig(level=logging.INFO)

hatchet = Hatchet(
    debug=True,
    config=ClientConfig(
        logger=logging.getLogger(),
    ),
)
