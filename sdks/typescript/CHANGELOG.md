# Changelog

All notable changes to Hatchet's TypeScript SDK will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.10.6] - 2026-01-27

### Changed

- Improves handling of cancellations for tasks to limit how often tasks receive a cancellation but then are marked as succeeded anyways.
