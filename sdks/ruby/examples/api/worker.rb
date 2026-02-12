# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

def main
  workflow_list = HATCHET.workflows.list
  rows = workflow_list.rows || []

  rows.each do |workflow|
    puts workflow.name
    puts workflow.metadata.id
    puts workflow.metadata.created_at
    puts workflow.metadata.updated_at
  end
end

main if __FILE__ == $PROGRAM_NAME
