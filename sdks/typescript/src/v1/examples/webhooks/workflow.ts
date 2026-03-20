import { hatchet } from '../hatchet-client';

export type WebhookInput = {
  type: string;
  message: string;
};

export const webhookWorkflow = hatchet.workflow<WebhookInput>({
  name: 'webhook-workflow',
  onEvents: ['webhook:test'],
});

webhookWorkflow.task({
  name: 'webhook-task',
  fn: async (input: WebhookInput) => {
    return input;
  },
});

// > Stripe webhook task
type StripePaymentInput = {
  type: string;
  data: {
    object: {
      customer: string;
      amount: number;
    };
  };
};

export const handleStripePayment = hatchet.task({
  name: 'handle-stripe-payment',
  on: {
    event: 'stripe:payment_intent.succeeded',
  },
  fn: async (input: StripePaymentInput) => {
    const { customer, amount } = input.data.object;
    console.log(`Payment of ${amount} from ${customer}`);
    return { customer, amount };
  },
});
// !!

// > GitHub webhook task
type GitHubPRInput = {
  action: string;
  pull_request: {
    number: number;
    title: string;
  };
  repository: {
    full_name: string;
  };
};

export const handleGitHubPR = hatchet.task({
  name: 'handle-github-pr',
  on: {
    event: 'github:pull_request:opened',
  },
  fn: async (input: GitHubPRInput) => {
    const repo = input.repository.full_name;
    const prNumber = input.pull_request.number;
    const title = input.pull_request.title;
    console.log(`PR #${prNumber} opened on ${repo}: ${title}`);
    return { repo, pr: prNumber };
  },
});
// !!

// > Slack event subscription task
type SlackEventInput = {
  event: {
    type: string;
    user: string;
    text: string;
    channel: string;
  };
};

export const handleSlackMention = hatchet.task({
  name: 'handle-slack-mention',
  on: {
    event: 'slack:event:app_mention',
  },
  fn: async (input: SlackEventInput) => {
    const { user, text, channel } = input.event;
    console.log(`Mentioned by ${user} in ${channel}: ${text}`);
    return { handled: true };
  },
});
// !!

// > Slack slash command task
type SlackCommandInput = {
  command: string;
  text: string;
  user_name: string;
  response_url: string;
};

export const handleSlackCommand = hatchet.task({
  name: 'handle-slack-command',
  on: {
    event: 'slack:command:/deploy',
  },
  fn: async (input: SlackCommandInput) => {
    console.log(`${input.user_name} ran ${input.command} ${input.text}`);
    return { command: input.command, args: input.text };
  },
});
// !!

// > Slack interaction task
type SlackInteractionInput = {
  type: string;
  actions: Array<{ action_id: string }>;
  user: { username: string };
};

export const handleSlackInteraction = hatchet.task({
  name: 'handle-slack-interaction',
  on: {
    event: 'slack:interaction:block_actions',
  },
  fn: async (input: SlackInteractionInput) => {
    const action = input.actions[0];
    console.log(`${input.user.username} clicked button: ${action.action_id}`);
    return { action: action.action_id };
  },
});
// !!
