# frozen_string_literal: true

RSpec.describe Hatchet do
  it "has a version number" do
    expect(Hatchet::VERSION).not_to be nil
  end

  it "cans create a client" do
    client = Hatchet::Client.new("test")
    expect(client).not_to be nil
    expect(client.api_key).to eq("test")
  end
end
