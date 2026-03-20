# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new unless defined?(HATCHET)

# > Stripe webhook task
HANDLE_STRIPE_PAYMENT = HATCHET.task(
  name: "handle-stripe-payment",
  on_events: ["stripe:payment_intent.succeeded"]
) do |input, ctx|
  customer = input["data"]["object"]["customer"]
  amount = input["data"]["object"]["amount"]
  puts "Payment of #{amount} from #{customer}"
  { "customer" => customer, "amount" => amount }
end
# !!

# > GitHub webhook task
HANDLE_GITHUB_PR = HATCHET.task(
  name: "handle-github-pr",
  on_events: ["github:pull_request:opened"]
) do |input, ctx|
  repo = input["repository"]["full_name"]
  pr_number = input["pull_request"]["number"]
  title = input["pull_request"]["title"]
  puts "PR ##{pr_number} opened on #{repo}: #{title}"
  { "repo" => repo, "pr" => pr_number }
end
# !!

# > Slack event subscription task
HANDLE_SLACK_MENTION = HATCHET.task(
  name: "handle-slack-mention",
  on_events: ["slack:event:app_mention"]
) do |input, ctx|
  event = input["event"]
  puts "Mentioned by #{event["user"]} in #{event["channel"]}: #{event["text"]}"
  { "handled" => true }
end
# !!

# > Slack slash command task
HANDLE_SLACK_COMMAND = HATCHET.task(
  name: "handle-slack-command",
  on_events: ["slack:command:/deploy"]
) do |input, ctx|
  puts "#{input["user_name"]} ran #{input["command"]} #{input["text"]}"
  { "command" => input["command"], "args" => input["text"] }
end
# !!

# > Slack interaction task
HANDLE_SLACK_INTERACTION = HATCHET.task(
  name: "handle-slack-interaction",
  on_events: ["slack:interaction:block_actions"]
) do |input, ctx|
  action = input["actions"][0]
  puts "#{input["user"]["username"]} clicked button: #{action["action_id"]}"
  { "action" => action["action_id"] }
end
# !!

def main
  worker = HATCHET.worker(
    name: "webhook-worker",
    workflows: [
      HANDLE_STRIPE_PAYMENT,
      HANDLE_GITHUB_PR,
      HANDLE_SLACK_MENTION,
      HANDLE_SLACK_COMMAND,
      HANDLE_SLACK_INTERACTION
    ]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
