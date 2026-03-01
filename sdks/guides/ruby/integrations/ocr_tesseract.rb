# frozen_string_literal: true

# Third-party integration - requires: bundle add rtesseract; install Tesseract binary
# See: /guides/document-processing

require 'rtesseract'

# > Tesseract usage
def parse_document(content)
  RTesseract.new(nil, data: content).to_s
end
# !!
