import pytest
from dotenv import load_dotenv

from hatchet_sdk import new_client
from hatchet_sdk.hatchet import Hatchet

load_dotenv()


@pytest.mark.asyncio(scope="session")
async def test_direct_client_event() -> None:
    client = new_client()
    e = client.event.push("user:create", {"test": "test"})

    assert e.eventId is not None


@pytest.mark.filterwarnings(
    "ignore:Direct access to client is deprecated:DeprecationWarning"
)
@pytest.mark.asyncio(scope="session")
async def test_hatchet_client_event() -> None:
    hatchet = Hatchet()
    e = hatchet.client.event.push("user:create", {"test": "test"})

    assert e.eventId is not None
