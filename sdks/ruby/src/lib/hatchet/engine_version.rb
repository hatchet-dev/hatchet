# frozen_string_literal: true

module Hatchet
  # Minimum engine versions required for specific SDK features.
  #
  # Mirrors :class:`hatchet_sdk.engine_version.MinEngineVersion` in the
  # Python SDK.
  module MinEngineVersion
    SLOT_CONFIG = "v0.78.23"
    DURABLE_EVICTION = "v0.80.0"
    OBSERVABILITY = "v0.82.0"
  end

  # Semver parsing + comparison helpers used to gate features by engine version.
  module EngineVersion
    module_function

    # Parse a semver string like ``"v0.78.23"`` into ``[major, minor, patch]``.
    #
    # Returns ``[0, 0, 0]`` if parsing fails, matching the Python helper.
    #
    # @param version [String, nil] The version string (with or without a ``v`` prefix, optional ``-pre`` suffix)
    # @return [Array(Integer, Integer, Integer)]
    def parse_semver(version)
      return [0, 0, 0] if version.nil?

      v = version.to_s
      v = v.sub(/\Av/, "")
      v = v.split("-", 2).first || ""

      parts = v.split(".")
      return [0, 0, 0] if parts.length != 3

      [Integer(parts[0]), Integer(parts[1]), Integer(parts[2])]
    rescue ArgumentError, TypeError
      [0, 0, 0]
    end

    # @param left_version [String, nil]
    # @param right_version [String, nil]
    # @return [Boolean] true if semver ``left_version`` is strictly less than ``right_version``
    def semver_less_than?(left_version, right_version)
      (parse_semver(left_version) <=> parse_semver(right_version)).negative?
    end

    class << self
      alias semver_less_than semver_less_than?
    end
  end
end
