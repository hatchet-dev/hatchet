## [0.87.14] - 2026-05-30

### Fixed
- Use leases to protect concurrent partition creation (un-revert) by @juliusgeo in [#4051](https://github.com/hatchet-dev/hatchet/pull/4051)

## [0.87.12] - 2026-05-30

### Fixed
- **engine:** Re-enable external id dupe check, add flag to enable dynamic window size by @mrkaye97 in [#4050](https://github.com/hatchet-dev/hatchet/pull/4050)

## [0.87.11] - 2026-05-30

### Fixed
- Remove external id check for now, it's too expensive by @mrkaye97 in [#4048](https://github.com/hatchet-dev/hatchet/pull/4048)

## [0.87.10] - 2026-05-30

### Fixed
- **engine:** Lengthen timeout for external id dupe check by @mrkaye97 in [#4047](https://github.com/hatchet-dev/hatchet/pull/4047)

## [0.87.9] - 2026-05-29

### Fixed
- Migration order for partition leases by @juliusgeo in [#4046](https://github.com/hatchet-dev/hatchet/pull/4046)

## [0.87.8] - 2026-05-29

### Fixed
- Use leases to protect concurrent partition creation by @juliusgeo in [#4044](https://github.com/hatchet-dev/hatchet/pull/4044)

## [0.87.7] - 2026-05-29

### Added
- **dashboard:** Support named shard deployment targets by @igor-kupczynski in [#4042](https://github.com/hatchet-dev/hatchet/pull/4042)


### Fixed
- **engine:** Some more payload offload improvements by @mrkaye97 in [#4041](https://github.com/hatchet-dev/hatchet/pull/4041)
- Sleep bug for anything over 59s by @darren-west in [#4012](https://github.com/hatchet-dev/hatchet/pull/4012)
- **cli:** Update quickstart template SDK versions by @BloggerBust in [#4025](https://github.com/hatchet-dev/hatchet/pull/4025)

## [0.87.6] - 2026-05-28

### Added
- **engine:** Index file-based payload offloads by @mrkaye97 in [#3979](https://github.com/hatchet-dev/hatchet/pull/3979)

## [0.87.5] - 2026-05-28

### Fixed
- Inactivity timeout breaks when set to greater than 24.85 days by @abelanger5 in [#4036](https://github.com/hatchet-dev/hatchet/pull/4036)
- Use `$user_` prefix, remove cross-domain hacks by @abelanger5 in [#4035](https://github.com/hatchet-dev/hatchet/pull/4035)

## [0.87.3] - 2026-05-28

### Fixed
- Go use durable listener for async results by @grutt in [#4019](https://github.com/hatchet-dev/hatchet/pull/4019)


## New Contributors
* @darren-west made their first contribution in [#4012](https://github.com/hatchet-dev/hatchet/pull/4012)
