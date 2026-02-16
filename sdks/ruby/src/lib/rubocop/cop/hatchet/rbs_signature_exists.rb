# frozen_string_literal: true

module RuboCop
  module Cop
    module Hatchet
      # Ensures every hand-written `.rb` file under `lib/` has a corresponding
      # `.rbs` type signature file under `sig/`.
      #
      # The mapping replaces `lib/` with `sig/` and `.rb` with `.rbs`.
      # For example:
      #   lib/hatchet/config.rb  ->  sig/hatchet/config.rbs
      #   lib/hatchet-sdk.rb     ->  sig/hatchet-sdk.rbs
      #
      # Generated directories (clients/rest, contracts) should be excluded
      # via the RuboCop configuration rather than hard-coded here.
      #
      # @example
      #   # bad – missing sig/hatchet/my_class.rbs
      #   # lib/hatchet/my_class.rb exists without a corresponding .rbs
      #
      #   # good – sig/hatchet/my_class.rbs exists
      #   # lib/hatchet/my_class.rb has a matching .rbs file
      class RbsSignatureExists < Base
        MSG = "Missing RBS signature file: `%<expected_path>s`"

        def on_new_investigation
          # Only inspect files under lib/
          source_path = processed_source.file_path
          return unless source_path&.include?("/lib/")

          # Extract the relative path from lib/ onward
          lib_index = source_path.index("/lib/")
          return unless lib_index

          relative = source_path[(lib_index + 1)..]

          # Compute expected .rbs path
          rbs_relative = relative.sub(%r{^lib/}, "sig/").sub(/\.rb\z/, ".rbs")

          # Resolve to absolute path based on project root (parent of lib/)
          project_root = source_path[0...lib_index]
          expected_rbs = File.join(project_root, rbs_relative)

          return if File.exist?(expected_rbs)

          add_offense(
            processed_source.ast || processed_source.buffer,
            message: format(MSG, expected_path: rbs_relative),
          )
        end
      end
    end
  end
end
