# frozen_string_literal: true

require_relative "lib/hatchet/version"

Gem::Specification.new do |spec|
  spec.name = "hatchet-sdk"
  spec.version = Hatchet::VERSION
  spec.authors = ["gabriel ruttner"]
  spec.email = ["gabe@hatchet.run"]

  spec.summary = "Ruby SDK for Hatchet, a distributed, fault-tolerant task orchestration engine"
  spec.description = "The official Ruby SDK for Hatchet, a distributed, fault-tolerant task orchestration engine. Easily integrate Hatchet's task scheduling and workflow orchestration capabilities into your Ruby applications."
  spec.homepage = "https://hatchet.run"
  spec.license = "MIT"
  spec.required_ruby_version = ">= 3.1.0"

  spec.metadata["allowed_push_host"] = "https://rubygems.org"

  spec.metadata["homepage_uri"] = spec.homepage
  spec.metadata["source_code_uri"] = "https://github.com/hatchet-dev/hatchet/tree/main/sdks/ruby"
  spec.metadata["changelog_uri"] = "https://github.com/hatchet-dev/hatchet/blob/main/sdks/ruby/src/CHANGELOG.md"

  # Specify which files should be added to the gem when it is released.
  # The `git ls-files -z` loads the files in the RubyGem that have been added into git.
  gemspec = File.basename(__FILE__)
  spec.files = IO.popen(%w[git ls-files -z], chdir: __dir__, err: IO::NULL) do |ls|
    ls.readlines("\x0", chomp: true).reject do |f|
      (f == gemspec) ||
        f.start_with?(*%w[bin/ test/ spec/ features/ .git .github appveyor Gemfile]) ||
        !File.exist?(File.join(__dir__, f))
    end
  end

  # Add generated REST API client files to the gem
  rest_client_dir = "lib/hatchet/clients/rest"
  if Dir.exist?(rest_client_dir)
    rest_files = Dir.glob("#{rest_client_dir}/**/*.rb").select { |f| File.file?(f) }
    spec.files.concat(rest_files)
  end

  # Add generated protobuf/gRPC contract files to the gem
  contracts_dir = "lib/hatchet/contracts"
  if Dir.exist?(contracts_dir)
    contract_files = Dir.glob("#{contracts_dir}/**/*.rb").select { |f| File.file?(f) }
    spec.files.concat(contract_files)
  end
  spec.bindir = "exe"
  spec.executables = spec.files.grep(%r{\Aexe/}) { |f| File.basename(f) }
  spec.require_paths = ["lib"]

  # Runtime dependencies for REST API client
  spec.add_dependency "faraday", "~> 2.0"
  spec.add_dependency "faraday-multipart"
  spec.add_dependency "marcel"
  spec.add_dependency "json", "~> 2.0"

  # Runtime dependencies for gRPC
  spec.add_dependency "grpc", "~> 1.60"
  spec.add_dependency "google-protobuf", "~> 4.0"
  spec.add_dependency "concurrent-ruby", ">= 1.1"

  # Development dependencies
  spec.add_development_dependency "gem-release", "~> 2.2"
  spec.add_development_dependency "rspec", "~> 3.0"
  spec.add_development_dependency "grpc-tools", "~> 1.60"

  # For more information and examples about making a new gem, check out our
  # guide at: https://bundler.io/guides/creating_gem.html
end
