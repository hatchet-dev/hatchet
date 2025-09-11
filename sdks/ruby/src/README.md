# Hatchet Ruby SDK

<div align="center">

[![Gem Version](https://badge.fury.io/rb/hatchet.svg)](https://badge.fury.io/rb/hatchet)
[![Documentation](https://img.shields.io/badge/docs-hatchet.run-blue)](https://docs.hatchet.run)
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)

</div>

This is the official Ruby SDK for [Hatchet](https://hatchet.run), a distributed, fault-tolerant task queue. The SDK allows you to easily integrate Hatchet's task scheduling and workflow orchestration capabilities into your Ruby applications.

## Installation

Add this line to your application's Gemfile:

```ruby
gem 'hatchet'
```

And then execute:

```bash
bundle install
```

Or install it yourself as:

```bash
gem install hatchet
```

## Quick Start

For examples of how to use the Hatchet Ruby SDK, including worker setup and task execution, please see our [official documentation](https://docs.hatchet.run/home/setup).

## Features

- 🔄 **Workflow Orchestration**: Define complex workflows with dependencies and parallel execution
- 🔁 **Automatic Retries**: Configure retry policies for handling transient failures
- 📊 **Observability**: Track workflow progress and monitor execution metrics
- ⏰ **Scheduling**: Schedule workflows to run at specific times or on a recurring basis
- 🔄 **Event-Driven**: Trigger workflows based on events in your system

## Documentation

For detailed documentation, examples, and best practices, visit:
- [Hatchet Documentation](https://docs.hatchet.run)
- [Examples](https://github.com/hatchet-dev/hatchet/tree/main/sdks/ruby/examples)

## Development

After checking out the repo, run `bin/setup` to install dependencies. Then, run `rake spec` to run the tests. You can also run `bin/console` for an interactive prompt that will allow you to experiment.

To install this gem onto your local machine, run `bundle exec rake install`. To release a new version, update the version number in `version.rb`, and then run `bundle exec rake release`, which will create a git tag for the version, push git commits and the created tag, and push the `.gem` file to [rubygems.org](https://rubygems.org).

Run tests with `bundle exec rspec spec`.

This project uses `gem-release` to help with versioning.

```
$ gem bum # bumps to the next patch version
$ gem bump --version minor # bumps to the next minor version
$ gem bump --version major # bumps to the next major version
$ gem bump --version 1.1.1 # bumps to the specified version
```

## Contributing

We welcome contributions! Please check out our [contributing guidelines](https://docs.hatchet.run/contributing) and join our [Discord community](https://hatchet.run/discord) for discussions and support.

## License

This SDK is released under the MIT License. See [LICENSE](https://github.com/hatchet-dev/hatchet/blob/main/LICENSE) for details.
