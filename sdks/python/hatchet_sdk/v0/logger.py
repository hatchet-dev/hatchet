import logging
import sys

# Create a named logger
logger = logging.getLogger("hatchet")
logger.setLevel(logging.INFO)

handler = logging.StreamHandler(sys.stdout)
formatter = logging.Formatter("[%(levelname)s]\t🪓 -- %(asctime)s - %(message)s")
handler.setFormatter(formatter)
logger.addHandler(handler)

logger.propagate = False
