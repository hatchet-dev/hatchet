# frozen_string_literal: true

RSpec.describe Hatchet::Connection do
  describe ".filtered_default_roots" do
    subject(:roots) { described_class.filtered_default_roots }

    it "drops the compromised roots and keeps the rest" do
      fingerprints = roots.scan(described_class::CERTIFICATE_PEM_PATTERN).map do |pem|
        Digest::SHA256.hexdigest(OpenSSL::X509::Certificate.new(pem).to_der).upcase
      end
      subjects = roots.scan(described_class::CERTIFICATE_PEM_PATTERN).map do |pem|
        OpenSSL::X509::Certificate.new(pem).subject.to_s
      end

      expect(fingerprints).not_to be_empty
      expect(fingerprints & described_class::COMPROMISED_ROOT_FINGERPRINTS).to be_empty
      expect(subjects).to include(a_string_matching(/GTS Root R4/))
    end
  end
end
