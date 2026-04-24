# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::EngineVersion do
  describe ".parse_semver" do
    it "parses a standard version with v prefix" do
      expect(described_class.parse_semver("v0.78.23")).to eq([0, 78, 23])
    end

    it "parses a version without v prefix" do
      expect(described_class.parse_semver("1.2.3")).to eq([1, 2, 3])
    end

    it "strips prerelease suffix" do
      expect(described_class.parse_semver("v0.1.0-alpha.0")).to eq([0, 1, 0])
      expect(described_class.parse_semver("v10.20.30-rc.1")).to eq([10, 20, 30])
    end

    it "returns zero tuple for empty string" do
      expect(described_class.parse_semver("")).to eq([0, 0, 0])
    end

    it "returns zero tuple for nil" do
      expect(described_class.parse_semver(nil)).to eq([0, 0, 0])
    end

    it "returns zero tuple for malformed input" do
      expect(described_class.parse_semver("v1.2")).to eq([0, 0, 0])
      expect(described_class.parse_semver("not-a-version")).to eq([0, 0, 0])
    end
  end

  describe ".semver_less_than" do
    it "returns true when a is less than b on the patch component" do
      expect(described_class.semver_less_than("v0.78.22", "v0.78.23")).to be(true)
    end

    it "returns false when a equals b" do
      expect(described_class.semver_less_than("v0.78.23", "v0.78.23")).to be(false)
    end

    it "returns false when a is greater than b on the patch component" do
      expect(described_class.semver_less_than("v0.78.24", "v0.78.23")).to be(false)
    end

    it "compares by minor component when major matches" do
      expect(described_class.semver_less_than("v0.77.99", "v0.78.0")).to be(true)
      expect(described_class.semver_less_than("v0.79.0", "v0.78.99")).to be(false)
    end

    it "compares by major component" do
      expect(described_class.semver_less_than("v0.78.23", "v1.0.0")).to be(true)
      expect(described_class.semver_less_than("v1.0.0", "v0.99.99")).to be(false)
    end

    it "treats a prerelease suffix as the base version" do
      expect(described_class.semver_less_than("v0.1.0-alpha.0", "v0.78.23")).to be(true)
    end

    it "treats empty/malformed strings as zero" do
      expect(described_class.semver_less_than("", "v0.78.23")).to be(true)
      expect(described_class.semver_less_than("v0.78.23", "")).to be(false)
    end
  end
end
