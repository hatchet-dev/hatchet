from uuid import uuid4

import pytest

from examples.welcome_email.worker import (
    ONBOARDING_EVENT_KEY,
    SignupInput,
    welcome_email,
)
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_welcome_email_onboarding_before_timeout(hatchet: Hatchet) -> None:
    """User completes onboarding before timeout -> follow-up skipped."""
    user_id = f"test-{uuid4().hex[:8]}"
    signup = SignupInput(
        email="alice@example.com",
        user_id=user_id,
    )

    ref = await welcome_email.aio_run(signup, wait_for_result=False)

    await hatchet.event.aio_push(
        ONBOARDING_EVENT_KEY,
        {"status": "done"},
        scope=signup.user_id,
    )

    result = await ref.aio_result()

    assert result.user_id == user_id
    assert result.welcome_sent is True
    assert result.follow_up_sent is False


@pytest.mark.asyncio(loop_scope="session")
async def test_welcome_email_timeout_follow_up(hatchet: Hatchet) -> None:
    """No onboarding event -> workflow times out and sends follow-up."""
    user_id = f"test-{uuid4().hex[:8]}"
    signup = SignupInput(
        email="bob@example.com",
        user_id=user_id,
    )

    result = await welcome_email.aio_run(signup)

    assert result.user_id == user_id
    assert result.welcome_sent is True
    assert result.follow_up_sent is True
