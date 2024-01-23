import os
import sys
from loguru import logger

# loguru config
config = {
    "handlers": [
        {"sink": sys.stdout, "format": "hatchet -- {time} - {message}"},
    ],
}

logger.configure(**config)
