# frozen_string_literal: true

require "hatchet-sdk"
require_relative "worker"

HATCHET_CLIENT = Hatchet::Client.new

# > Create a filter
HATCHET_CLIENT.filters.create(
  workflow_id: EVENT_WORKFLOW.id,
  expression: "input.should_skip == false",
  scope: "foobarbaz",
  payload: {
    "main_character" => "Anna",
    "supporting_character" => "Stiva",
    "location" => "Moscow"
  }
)
# !!

# > Skip a run
HATCHET_CLIENT.event.push(
  EVENT_KEY,
  { "should_skip" => true },
  scope: "foobarbaz"
)
# !!

# > Trigger a run
HATCHET_CLIENT.event.push(
  EVENT_KEY,
  { "should_skip" => false },
  scope: "foobarbaz"
)
# !!
