# frozen_string_literal: true

require_relative "hatchet/version"

module Hatchet
  class Error < StandardError; end

  class Client
    attr_reader :api_key

    def initialize(api_key)
      @api_key = api_key
    end
  end
end
