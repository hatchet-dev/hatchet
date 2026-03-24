from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet(debug=True)


class WebhookInput(BaseModel):
    type: str
    message: str


@hatchet.task(input_validator=WebhookInput, on_events=["webhook:test"])
def webhook(input: WebhookInput, ctx: Context) -> dict[str, str]:
    return input.model_dump()


# > Stripe webhook task
class StripeObject(BaseModel):
    customer: str
    amount: int


class StripeData(BaseModel):
    object: StripeObject


class StripePaymentInput(BaseModel):
    type: str
    data: StripeData


class StripePaymentOutput(BaseModel):
    customer: str
    amount: int


@hatchet.task(
    input_validator=StripePaymentInput,
    on_events=["stripe:payment_intent.succeeded"],
)
def handle_stripe_payment(
    input: StripePaymentInput, ctx: Context
) -> StripePaymentOutput:
    customer = input.data.object.customer
    amount = input.data.object.amount
    print(f"Payment of {amount} from {customer}")
    return StripePaymentOutput(customer=customer, amount=amount)


# !!


# > GitHub webhook task
class GitHubPullRequest(BaseModel):
    number: int
    title: str


class GitHubRepository(BaseModel):
    full_name: str


class GitHubPRInput(BaseModel):
    action: str
    pull_request: GitHubPullRequest
    repository: GitHubRepository


class GitHubPROutput(BaseModel):
    repo: str
    pr: int


@hatchet.task(
    input_validator=GitHubPRInput,
    on_events=["github:pull_request:opened"],
)
def handle_github_pr(input: GitHubPRInput, ctx: Context) -> GitHubPROutput:
    repo = input.repository.full_name
    pr_number = input.pull_request.number
    title = input.pull_request.title
    print(f"PR #{pr_number} opened on {repo}: {title}")
    return GitHubPROutput(repo=repo, pr=pr_number)


# !!


# > Slack event subscription task
class SlackEvent(BaseModel):
    type: str
    user: str
    text: str
    channel: str


class SlackEventInput(BaseModel):
    event: SlackEvent


class SlackEventOutput(BaseModel):
    handled: bool


@hatchet.task(
    input_validator=SlackEventInput,
    on_events=["slack:event:app_mention"],
)
def handle_slack_mention(input: SlackEventInput, ctx: Context) -> SlackEventOutput:
    print(
        f"Mentioned by {input.event.user} in {input.event.channel}: {input.event.text}"
    )
    return SlackEventOutput(handled=True)


# !!


# > Slack slash command task
class SlackCommandInput(BaseModel):
    command: str
    text: str
    user_name: str
    response_url: str


class SlackCommandOutput(BaseModel):
    command: str
    args: str


@hatchet.task(
    input_validator=SlackCommandInput,
    on_events=["slack:command:/deploy"],
)
def handle_slack_command(input: SlackCommandInput, ctx: Context) -> SlackCommandOutput:
    print(f"{input.user_name} ran {input.command} {input.text}")
    return SlackCommandOutput(command=input.command, args=input.text)


# !!


# > Slack interaction task
class SlackAction(BaseModel):
    action_id: str


class SlackUser(BaseModel):
    username: str


class SlackInteractionInput(BaseModel):
    type: str
    actions: list[SlackAction]
    user: SlackUser


class SlackInteractionOutput(BaseModel):
    action: str


@hatchet.task(
    input_validator=SlackInteractionInput,
    on_events=["slack:interaction:block_actions"],
)
def handle_slack_interaction(
    input: SlackInteractionInput, ctx: Context
) -> SlackInteractionOutput:
    action = input.actions[0]
    print(f"{input.user.username} clicked button: {action.action_id}")
    return SlackInteractionOutput(action=action.action_id)


# !!


def main() -> None:
    worker = hatchet.worker(
        "webhook-worker",
        workflows=[
            webhook,
            handle_stripe_payment,
            handle_github_pr,
            handle_slack_mention,
            handle_slack_command,
            handle_slack_interaction,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
