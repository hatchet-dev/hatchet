# > Trigger the workflow
from examples.welcome_email.worker import (
    ONBOARDING_EVENT_KEY,
    SignupInput,
    hatchet,
    welcome_email,
)

signup = SignupInput(
    email="alice@example.com",
    user_id="user-123",
)

# Start the welcome-email workflow
ref = welcome_email.run(signup, wait_for_result=False)
print(f"Started workflow run: {ref.workflow_run_id}")

# Push onboarding-completed event (scoped to this user)
print("Pushing onboarding-completed event...")
hatchet.event.push(
    ONBOARDING_EVENT_KEY,
    {"status": "done"},
    scope=signup.user_id,
)

# Wait for the workflow to complete
result = ref.result()
print(f"Workflow completed: {result}")


# !!
