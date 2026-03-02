# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "ReturnExceptionsTask" do
  it "returns exceptions for failed tasks and results for successful ones" do
    results = RETURN_EXCEPTIONS_TASK.run_many(
      10.times.map do |i|
        RETURN_EXCEPTIONS_TASK.create_bulk_run_item(
          input: { "index" => i }
        )
      end,
      return_exceptions: true
    )

    results.each_with_index do |result, i|
      if i.even?
        expect(result).to be_a(Exception)
        expect(result.message).to include("error in task with index #{i}")
      else
        expect(result).to eq({ "message" => "this is a successful task." })
      end
    end
  end
end
