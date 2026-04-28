from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import (
    DurableContext,
    Hatchet,
    SleepCondition,
    UserEventCondition,
    or_,
)

hatchet = Hatchet()

ONBOARDING_EVENT_KEY = "user:onboarding-completed"
TIMEOUT_SECONDS = 5
LOOKBACK_MINUTES = 5


# > Models
class SignupInput(BaseModel):
    email: str
    user_id: str


class WelcomeEmailResult(BaseModel):
    user_id: str
    welcome_sent: bool
    follow_up_sent: bool




# > Welcome email task
@hatchet.durable_task(
    name="welcome-email",
    on_events=["user:signup"],
    input_validator=SignupInput,
    execution_timeout=timedelta(minutes=5),
)
async def welcome_email(input: SignupInput, ctx: DurableContext) -> WelcomeEmailResult:
    # Step 1: Send the welcome email
    print(f"Sending welcome email to {input.email}: finish your first onboarding step")

    # Step 2: Wait for the user to complete onboarding, or time out
    # (use a longer duration for a more realistic workflow)
    now = await ctx.aio_now()
    consider_events_since = now - timedelta(minutes=LOOKBACK_MINUTES)

    wait_result = await ctx.aio_wait_for(
        "onboarding-or-timeout",
        or_(
            SleepCondition(timedelta(seconds=TIMEOUT_SECONDS)),
            # Scope the event condition to this user so that another user's
            # onboarding-completed event does not resolve this wait.
            UserEventCondition(
                event_key=ONBOARDING_EVENT_KEY,
                scope=input.user_id,
                consider_events_since=consider_events_since,
            ),
        ),
    )

    # The or-group result is {"CREATE": {"<condition_key>": ...}}.
    # Check whether the onboarding event was the one that resolved.
    resolved_key = list(wait_result["CREATE"].keys())[0]
    onboarding_completed = resolved_key == ONBOARDING_EVENT_KEY

    if onboarding_completed:
        # Step 3a: User completed onboarding -> skip follow-up
        print(f"User {input.user_id} completed onboarding, skipping follow-up")
        return WelcomeEmailResult(
            user_id=input.user_id,
            welcome_sent=True,
            follow_up_sent=False,
        )

    # Step 3b: Timeout -> send follow-up email
    print(f"Sending follow-up email to {input.email}: need help finishing onboarding?")
    return WelcomeEmailResult(
        user_id=input.user_id,
        welcome_sent=True,
        follow_up_sent=True,
    )




# > Worker registration
def main() -> None:
    worker = hatchet.worker("welcome-email-worker", workflows=[welcome_email])
    worker.start()


if __name__ == "__main__":
    main()


