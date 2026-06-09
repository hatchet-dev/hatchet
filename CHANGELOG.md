## [0.89.0] - 2026-06-09


## [0.88.0] - 2026-05-30

### Fixed
- Use leases to protect concurrent partition creation (un-revert) by @juliusgeo in [#4051](https://github.com/hatchet-dev/hatchet/pull/4051)


## [0.87.12] - 2026-05-30

### Fixed
- Re-enable external id dupe check, add flag to enable dynamic window size by @mrkaye97 in [#4050](https://github.com/hatchet-dev/hatchet/pull/4050)


## [0.87.11] - 2026-05-30

### Fixed
- Remove external id check for now, it's too expensive by @mrkaye97 in [#4048](https://github.com/hatchet-dev/hatchet/pull/4048)


## [0.87.10] - 2026-05-30

### Fixed
- Lengthen timeout for external id dupe check by @mrkaye97 in [#4047](https://github.com/hatchet-dev/hatchet/pull/4047)


## [0.87.9] - 2026-05-29

### Fixed
- Migration order for partition leases by @juliusgeo in [#4046](https://github.com/hatchet-dev/hatchet/pull/4046)


## [0.87.8] - 2026-05-29

### Fixed
- Use leases to protect concurrent partition creation by @juliusgeo in [#4044](https://github.com/hatchet-dev/hatchet/pull/4044)


## [0.87.7] - 2026-05-29

### Added
- Support named shard deployment targets by @igor-kupczynski in [#4042](https://github.com/hatchet-dev/hatchet/pull/4042)

### Fixed
- Some more payload offload improvements by @mrkaye97 in [#4041](https://github.com/hatchet-dev/hatchet/pull/4041)
- Sleep bug for anything over 59s by @darren-west in [#4012](https://github.com/hatchet-dev/hatchet/pull/4012)
- Update quickstart template SDK versions by @BloggerBust in [#4025](https://github.com/hatchet-dev/hatchet/pull/4025)


## [0.87.6] - 2026-05-28

### Added
- Index file-based payload offloads by @mrkaye97 in [#3979](https://github.com/hatchet-dev/hatchet/pull/3979)


## [0.87.3] - 2026-05-28

### Fixed
- Go use durable listener for async results by @grutt in [#4019](https://github.com/hatchet-dev/hatchet/pull/4019)


## [0.87.1] - 2026-05-26

### Added
- Move billing to control plane by @grutt in [#3982](https://github.com/hatchet-dev/hatchet/pull/3982)


## [0.86.32] - 2026-05-26

### Added
- Show structured log fields in log viewer by @MicroYui in [#3972](https://github.com/hatchet-dev/hatchet/pull/3972)

### Changed
- Use shared Dialog for plan upgrade modal by @jishnundth in [#3887](https://github.com/hatchet-dev/hatchet/pull/3887)

### Removed
- Remove backoff overflow test by @juliusgeo in [#3990](https://github.com/hatchet-dev/hatchet/pull/3990)

### Fixed
- Hide github tab if managed workers are disabled by @mrkaye97 in [#3988](https://github.com/hatchet-dev/hatchet/pull/3988)


## [0.86.29] - 2026-05-22

### Fixed
- Migration number by @mrkaye97 in [#3987](https://github.com/hatchet-dev/hatchet/pull/3987)


## [0.86.28] - 2026-05-22

### Fixed
- A couple more migration performance issues by @mrkaye97 in [#3942](https://github.com/hatchet-dev/hatchet/pull/3942)


## [0.86.27] - 2026-05-22

### Added
- Make Hatchet Lite version configurable by @Ujjwal-Singh-20 in [#3925](https://github.com/hatchet-dev/hatchet/pull/3925)

### Removed
- Revert "chore: move requests to control plane" by @grutt

### Fixed
- Payload external id duplicates from durable child spawning by @mrkaye97 in [#3984](https://github.com/hatchet-dev/hatchet/pull/3984)


## [0.86.26] - 2026-05-21

### Fixed
- Increase limit default to 10k by @mrkaye97 in [#3978](https://github.com/hatchet-dev/hatchet/pull/3978)


## [0.86.25] - 2026-05-20

### Fixed
- Only run worker id action getter query when needed by @mrkaye97 in [#3976](https://github.com/hatchet-dev/hatchet/pull/3976)


## [0.86.24] - 2026-05-20

### Changed
- Make SSO Configuration UI more clear by @juliusgeo in [#3971](https://github.com/hatchet-dev/hatchet/pull/3971)

### Fixed
- Correctly handle child key-based caching in durable task child spawning by @mrkaye97 in [#3955](https://github.com/hatchet-dev/hatchet/pull/3955)


## [0.86.23] - 2026-05-20

### Fixed
- Attempt at improving worker actions query by @mrkaye97 in [#3774](https://github.com/hatchet-dev/hatchet/pull/3774)


## [0.86.22] - 2026-05-20

### Added
- Entitlement check by @grutt in [#3969](https://github.com/hatchet-dev/hatchet/pull/3969)
- Move workflow search bar to the top of workflows table by @abelanger5 in [#3967](https://github.com/hatchet-dev/hatchet/pull/3967)
- Add Dockerfile for all SDK e2e tests, fix flaky e2e tests by @juliusgeo in [#3846](https://github.com/hatchet-dev/hatchet/pull/3846)


## [0.86.21] - 2026-05-19

### Changed
- Pass `insertedAt` to OLAP payload reads by @mrkaye97 in [#3960](https://github.com/hatchet-dev/hatchet/pull/3960)

### Removed
- Revert "feat: route frontend billing through control plane" by @grutt in [#3963](https://github.com/hatchet-dev/hatchet/pull/3963)


## [0.86.20] - 2026-05-19

### Changed
- Use `externalId` for retrieving payloads by @mrkaye97 in [#3914](https://github.com/hatchet-dev/hatchet/pull/3914)


## [0.86.19] - 2026-05-19

### Added
- Add logging for serial operation by @juliusgeo in [#3957](https://github.com/hatchet-dev/hatchet/pull/3957)
- Added validation for seed password by @NathanDrake007 in [#3783](https://github.com/hatchet-dev/hatchet/pull/3783)
- Route frontend billing through control plane by @grutt

### Fixed
- Deadlock on trigger workflows by @mrkaye97 in [#3906](https://github.com/hatchet-dev/hatchet/pull/3906)
- Re-order auth flow to prevent side effects on handleCookieAuth by @gregfurman in [#3923](https://github.com/hatchet-dev/hatchet/pull/3923)
- Dedupe security check in hatchet-lite by @MicroYui in [#3928](https://github.com/hatchet-dev/hatchet/pull/3928)
- Run the job at midnight utc by @mrkaye97 in [#3909](https://github.com/hatchet-dev/hatchet/pull/3909)
- Persist dashboard column visibility across reloads by @sajdakabir in [#3844](https://github.com/hatchet-dev/hatchet/pull/3844)
- Poll plan by @grutt
- Dedicated state by @grutt


## [0.86.18] - 2026-05-14

### Fixed
- New user flow on control plane deployment by @grutt in [#3891](https://github.com/hatchet-dev/hatchet/pull/3891)


## [0.86.17] - 2026-05-13

### Fixed
- Separate transactions in payload migration by @mrkaye97 in [#3907](https://github.com/hatchet-dev/hatchet/pull/3907)


## [0.86.16] - 2026-05-13

### Added
- Add error if counts don't match so we can debug by @mrkaye97 in [#3905](https://github.com/hatchet-dev/hatchet/pull/3905)


## [0.86.15] - 2026-05-13

### Added
- Paginate through payloads by external id, not using the PK by @mrkaye97 in [#3883](https://github.com/hatchet-dev/hatchet/pull/3883)


## [0.86.14] - 2026-05-13

### Added
- Control plane redirect by @grutt in [#3904](https://github.com/hatchet-dev/hatchet/pull/3904)


## [0.86.12] - 2026-05-12

### Fixed
- Limit the number of re-pubs to be the max death count by @mrkaye97 in [#3899](https://github.com/hatchet-dev/hatchet/pull/3899)


## [0.86.10] - 2026-05-12

### Added
- Add external id index to payloads tables by @mrkaye97 in [#3879](https://github.com/hatchet-dev/hatchet/pull/3879)


## [0.86.9] - 2026-05-12

### Added
- Add cron schedule time to additional meta by @mrkaye97 in [#3884](https://github.com/hatchet-dev/hatchet/pull/3884)

### Changed
- New `taskNames` query param on the task stats endpoint for KEDA friendliness by @mnafees in [#3791](https://github.com/hatchet-dev/hatchet/pull/3791)

### Fixed
- Durable eviction / restore causing duplicate queue items by @mrkaye97 in [#3864](https://github.com/hatchet-dev/hatchet/pull/3864)


## [0.86.8] - 2026-05-11

### Fixed
- Avoid invalid user FK in cookie test by @igor-kupczynski in [#3882](https://github.com/hatchet-dev/hatchet/pull/3882)
- Occasional empty payloads on DAG task retries by @juliusgeo in [#3860](https://github.com/hatchet-dev/hatchet/pull/3860)


## [0.86.7] - 2026-05-11

### Fixed
- Add backoff to sleeps by @mrkaye97 in [#3880](https://github.com/hatchet-dev/hatchet/pull/3880)


## [0.86.6] - 2026-05-11

### Added
- Add migratediag package for DSN redaction and error handling by @igor-kupczynski in [#3878](https://github.com/hatchet-dev/hatchet/pull/3878)

### Fixed
- Allow partial successes on status updates by @mrkaye97 in [#3861](https://github.com/hatchet-dev/hatchet/pull/3861)


## [0.86.5] - 2026-05-08

### Fixed
- Use correct log out and invalidate stale cookies by @grutt in [#3858](https://github.com/hatchet-dev/hatchet/pull/3858)


## [0.86.4] - 2026-05-08

### Added
- Configurable OLAP MQ QoS by @mrkaye97 in [#3857](https://github.com/hatchet-dev/hatchet/pull/3857)
- Add unit tests for DurableEventsListener reconnect and event delivery during retry backoff by @igor-kupczynski

### Fixed
- Durable wait mutex issue by @mrkaye97


## [0.86.1] - 2026-05-06

### Added
- Add cols to offload table to track count diffs by @mrkaye97 in [#3841](https://github.com/hatchet-dev/hatchet/pull/3841)


## [0.85.10] - 2026-05-06

### Fixed
- Merge issue? these definitely had been removed before... by @mrkaye97 in [#3839](https://github.com/hatchet-dev/hatchet/pull/3839)


## [0.85.9] - 2026-05-06

### Fixed
- Remove dual writes into temp tables for status updates by @mrkaye97 in [#3829](https://github.com/hatchet-dev/hatchet/pull/3829)
- Slow cold start when workflows are inactive > 1 day by @juliusgeo in [#3830](https://github.com/hatchet-dev/hatchet/pull/3830)


## [0.85.8] - 2026-05-06

### Fixed
- Prune partitions in reconciliation query with min inserted at filter by @mrkaye97 in [#3838](https://github.com/hatchet-dev/hatchet/pull/3838)
- Dedupe task ids before updating to assigned by @mrkaye97 in [#3818](https://github.com/hatchet-dev/hatchet/pull/3818)


## [0.85.7] - 2026-05-05

### Fixed
- Add back task event tmp dual write by @mrkaye97 in [#3828](https://github.com/hatchet-dev/hatchet/pull/3828)
- Improve performance of status metrics query via index usage + partition pruning by @mrkaye97 in [#3800](https://github.com/hatchet-dev/hatchet/pull/3800)
- Improve transactional safety of payload offload reads and writes by @mrkaye97 in [#3814](https://github.com/hatchet-dev/hatchet/pull/3814)


## [0.85.6] - 2026-05-05

### Fixed
- Payload external id dupe by @mrkaye97 in [#3824](https://github.com/hatchet-dev/hatchet/pull/3824)


## [0.85.5] - 2026-05-05

### Added
- Direct task + dag status updates by @mrkaye97 in [#3554](https://github.com/hatchet-dev/hatchet/pull/3554)
- Swap links to control plane by @grutt in [#3816](https://github.com/hatchet-dev/hatchet/pull/3816)


## [0.85.4] - 2026-05-04

### Fixed
- Show tenant invite modal correctly by @mrkaye97 in [#3815](https://github.com/hatchet-dev/hatchet/pull/3815)


## [0.85.3] - 2026-05-02

### Fixed
- Panic on trigger by @mrkaye97 in [#3803](https://github.com/hatchet-dev/hatchet/pull/3803)
- Refresh token edge cases by @grutt in [#3802](https://github.com/hatchet-dev/hatchet/pull/3802)


## [0.85.2] - 2026-05-01

### Fixed
- Wait for user universe by @grutt in [#3801](https://github.com/hatchet-dev/hatchet/pull/3801)


## [0.85.1] - 2026-05-01

### Fixed
- Hoist org management / tenant management page by @mrkaye97 in [#3797](https://github.com/hatchet-dev/hatchet/pull/3797)


## [0.85.0] - 2026-05-01

### Added
- Per organization inactivity timeout by @juliusgeo in [#3795](https://github.com/hatchet-dev/hatchet/pull/3795)
- Shard-affinity by @grutt in [#3788](https://github.com/hatchet-dev/hatchet/pull/3788)
- Add force SSO toggle by @juliusgeo in [#3787](https://github.com/hatchet-dev/hatchet/pull/3787)

### Fixed
- Fixes for SSO UI layout by @juliusgeo in [#3793](https://github.com/hatchet-dev/hatchet/pull/3793)
- Fix--control-plane-bash by @grutt in [#3781](https://github.com/hatchet-dev/hatchet/pull/3781)


## [0.83.61] - 2026-04-29

### Added
- Feature flag sso by @grutt in [#3786](https://github.com/hatchet-dev/hatchet/pull/3786)

### Changed
- Remove unused err var by @jishnundth in [#3785](https://github.com/hatchet-dev/hatchet/pull/3785)

### Fixed
- Workflow reschedule update fail by @jishnundth in [#3782](https://github.com/hatchet-dev/hatchet/pull/3782)


## [0.83.60] - 2026-04-28

### Fixed
- Add cron job to deactivate stale step concurrency configs by @mrkaye97 in [#3775](https://github.com/hatchet-dev/hatchet/pull/3775)
- Use `SimpleTable` for span attributes by @mrkaye97 in [#3778](https://github.com/hatchet-dev/hatchet/pull/3778)


## [0.83.59] - 2026-04-28

### Added
- Durable go by @grutt in [#3696](https://github.com/hatchet-dev/hatchet/pull/3696)

### Fixed
- Attempt to fix deadlocks, part 1bn by @mrkaye97 in [#3776](https://github.com/hatchet-dev/hatchet/pull/3776)


## [0.83.57] - 2026-04-28

### Changed
- Ignore invalid step IDs when inserting or replaying tasks by @mnafees in [#3735](https://github.com/hatchet-dev/hatchet/pull/3735)
- Make all list partition calls specific to v1 tenants only by @mnafees in [#3758](https://github.com/hatchet-dev/hatchet/pull/3758)

### Fixed
- Don't drop index by @mrkaye97 in [#3765](https://github.com/hatchet-dev/hatchet/pull/3765)
- Increase max password length to 64 and improve validation error message by @NathanDrake007 in [#3713](https://github.com/hatchet-dev/hatchet/pull/3713)


## [0.83.56] - 2026-04-28

### Fixed
- Acquire locks right away by @mrkaye97 in [#3762](https://github.com/hatchet-dev/hatchet/pull/3762)


## [0.83.55] - 2026-04-28

### Fixed
- Improve performance of cron schedule polling query by @mrkaye97 in [#3754](https://github.com/hatchet-dev/hatchet/pull/3754)


## [0.83.54] - 2026-04-27

### Fixed
- Remove row count check, replace with threshold by @mrkaye97 in [#3756](https://github.com/hatchet-dev/hatchet/pull/3756)


## [0.83.53] - 2026-04-27

### Added
- Go partition by partition by @mrkaye97 in [#3755](https://github.com/hatchet-dev/hatchet/pull/3755)
- Frontend SSO support by @juliusgeo in [#3582](https://github.com/hatchet-dev/hatchet/pull/3582)

### Fixed
- Trace view improvements by @mrkaye97 in [#3702](https://github.com/hatchet-dev/hatchet/pull/3702)


## [0.83.52] - 2026-04-27

### Fixed
- Do nothing on conflict, and diff out existing rows by @mrkaye97 in [#3752](https://github.com/hatchet-dev/hatchet/pull/3752)


## [0.83.50] - 2026-04-27

### Fixed
- Add exclusive lock, run analyze by @mrkaye97 in [#3739](https://github.com/hatchet-dev/hatchet/pull/3739)


## [0.83.49] - 2026-04-27

### Changed
- Error out when marking queue items as resolved by @mnafees in [#3736](https://github.com/hatchet-dev/hatchet/pull/3736)

### Fixed
- Add row count check before swapping by @mrkaye97 in [#3709](https://github.com/hatchet-dev/hatchet/pull/3709)


## [0.83.48] - 2026-04-24

### Fixed
- Fetching state causing redirect loop on signup by @mrkaye97 in [#3708](https://github.com/hatchet-dev/hatchet/pull/3708)


## [0.83.47] - 2026-04-24

### Fixed
- Broken migration by @mrkaye97 in [#3707](https://github.com/hatchet-dev/hatchet/pull/3707)


## [0.83.46] - 2026-04-24

### Fixed
- Remove status partitioning on `v1_(runs|dags|tasks)_olap` by @mrkaye97 in [#3603](https://github.com/hatchet-dev/hatchet/pull/3603)


## [0.83.45] - 2026-04-24

### Fixed
- Regen control plane api by @abelanger5 in [#3705](https://github.com/hatchet-dev/hatchet/pull/3705)


## [0.83.44] - 2026-04-24

### Changed
- Make sure we call for durable task invocation only for durable tasks in the scheduler by @mnafees in [#3698](https://github.com/hatchet-dev/hatchet/pull/3698)


## [0.83.43] - 2026-04-24

### Added
- Separate frontend and server url, pass the server url as part of the tenant by @abelanger5 in [#3697](https://github.com/hatchet-dev/hatchet/pull/3697)


## [0.83.42] - 2026-04-24

### Fixed
- Couple more frontend + webhooks things by @mrkaye97 in [#3695](https://github.com/hatchet-dev/hatchet/pull/3695)


## [0.83.41] - 2026-04-23

### Fixed
- Tailwind hell by @mrkaye97 in [#3693](https://github.com/hatchet-dev/hatchet/pull/3693)


## [0.83.40] - 2026-04-23

### Fixed
- Couple more frontend bugs by @mrkaye97 in [#3692](https://github.com/hatchet-dev/hatchet/pull/3692)


## [0.83.39] - 2026-04-23

### Fixed
- Return from delete query by @abelanger5 in [#3690](https://github.com/hatchet-dev/hatchet/pull/3690)
- Validate tenant membership on V1DagListTasks by @abelanger5 in [#3691](https://github.com/hatchet-dev/hatchet/pull/3691)


## [0.83.38] - 2026-04-23

### Added
- Workflow filter FTS by @mrkaye97 in [#3685](https://github.com/hatchet-dev/hatchet/pull/3685)

### Changed
- Make sure we don't poll for work in case of deleted tenants by @mnafees in [#3682](https://github.com/hatchet-dev/hatchet/pull/3682)
- Also clean slot configs for workers by @mnafees in [#3666](https://github.com/hatchet-dev/hatchet/pull/3666)
- Cache CEL programs in order to avoid expensive heap allocations on each event match by @mnafees in [#3667](https://github.com/hatchet-dev/hatchet/pull/3667)

### Fixed
- Clean up a bunch of settings pages on the dashboard by @mrkaye97 in [#3669](https://github.com/hatchet-dev/hatchet/pull/3669)
- Heights of event log, etc. on the dag view by @mrkaye97 in [#3687](https://github.com/hatchet-dev/hatchet/pull/3687)
- Significantly speed up event queries by @mrkaye97 in [#3688](https://github.com/hatchet-dev/hatchet/pull/3688)


## [0.83.37] - 2026-04-22

### Fixed
- Collision on lastTenant key by @abelanger5 in [#3671](https://github.com/hatchet-dev/hatchet/pull/3671)
- Use exchange token for getting plans by @abelanger5 in [#3668](https://github.com/hatchet-dev/hatchet/pull/3668)


## [0.83.36] - 2026-04-22

### Added
- Remove "account" dropdown, add notifications dropdown by @mrkaye97 in [#3665](https://github.com/hatchet-dev/hatchet/pull/3665)

### Fixed
- Webhook responses, event info on context, internal fix for labels matches by @mrkaye97 in [#3625](https://github.com/hatchet-dev/hatchet/pull/3625)


## [0.83.35] - 2026-04-22

### Removed
- Revert "Remove "account" dropdown', add notifications dropdown " by @mrkaye97 in [#3664](https://github.com/hatchet-dev/hatchet/pull/3664)


## [0.83.34] - 2026-04-21

### Changed
- Cleanup old workers via daily gocron by @mnafees in [#3663](https://github.com/hatchet-dev/hatchet/pull/3663)

### Removed
- Remove "account" dropdown', add notifications dropdown by @TehShrike in [#3365](https://github.com/hatchet-dev/hatchet/pull/3365)


## [0.83.33] - 2026-04-21

### Fixed
- Disable version info for control plane, add sync for tenant alerting settings by @abelanger5 in [#3659](https://github.com/hatchet-dev/hatchet/pull/3659)


## [0.83.32] - 2026-04-21

### Added
- Durable execution frontend work and API improvements by @mrkaye97 in [#3639](https://github.com/hatchet-dev/hatchet/pull/3639)


## [0.83.31] - 2026-04-21

### Fixed
- Use exchange token interceptor for cloud endpoints by @abelanger5 in [#3658](https://github.com/hatchet-dev/hatchet/pull/3658)


## [0.83.30] - 2026-04-21

### Fixed
- Set cloud enabled when control plane is enabled by @abelanger5 in [#3657](https://github.com/hatchet-dev/hatchet/pull/3657)


## [0.83.29] - 2026-04-21

### Fixed
- Fix control-plane logout bug, fix oauth redirect urls by @juliusgeo in [#3656](https://github.com/hatchet-dev/hatchet/pull/3656)


## [0.83.28] - 2026-04-21

### Changed
- Conditionally use control-plane metadata endpoint by @juliusgeo in [#3654](https://github.com/hatchet-dev/hatchet/pull/3654)


## [0.83.27] - 2026-04-20

### Added
- Rate limit deletion by @juliusgeo in [#3638](https://github.com/hatchet-dev/hatchet/pull/3638)

### Fixed
- Infinite rerender bug and handle 403 better by @abelanger5 in [#3652](https://github.com/hatchet-dev/hatchet/pull/3652)


## [0.83.26] - 2026-04-20

### Added
- Sync repository for syncing data into a tenant by @abelanger5 in [#3614](https://github.com/hatchet-dev/hatchet/pull/3614)

### Changed
- Generate insecure keysets locally by @juliusgeo in [#3622](https://github.com/hatchet-dev/hatchet/pull/3622)

### Fixed
- Exchange token authz, remove n+1 tenant lookup for organizations, slack oauth casing by @abelanger5 in [#3631](https://github.com/hatchet-dev/hatchet/pull/3631)


## [0.83.25] - 2026-04-15

### Fixed
- Configurable schedulerCheckActive Interval by @grutt in [#3624](https://github.com/hatchet-dev/hatchet/pull/3624)


## [0.83.24] - 2026-04-15

### Changed
- Rename PgBouncer env vars by @mnafees in [#3319](https://github.com/hatchet-dev/hatchet/pull/3319)

### Fixed
- Empty parent task external id on olap dags and runs by @mrkaye97 in [#3605](https://github.com/hatchet-dev/hatchet/pull/3605)
- Broken doc links by @mrkaye97 in [#3616](https://github.com/hatchet-dev/hatchet/pull/3616)


## [0.83.23] - 2026-04-13

### Added
- No limit modal with no limits by @grutt in [#3572](https://github.com/hatchet-dev/hatchet/pull/3572)

### Changed
- Make rbac package generic and reusable by @abelanger5 in [#3581](https://github.com/hatchet-dev/hatchet/pull/3581)

### Fixed
- Add healthcheck to harness engine startup by @gregfurman in [#3601](https://github.com/hatchet-dev/hatchet/pull/3601)


## [0.83.22] - 2026-04-10

### Removed
- Run actions on gha runners again by @mrkaye97 in [#3588](https://github.com/hatchet-dev/hatchet/pull/3588)

### Fixed
- Worker labels not respected on retry by @mrkaye97 in [#3591](https://github.com/hatchet-dev/hatchet/pull/3591)
- Run payloads job at midnight by @mrkaye97 in [#3578](https://github.com/hatchet-dev/hatchet/pull/3578)


## [0.83.18] - 2026-04-07

### Added
- Wait for event with lookback window by @mrkaye97 in [#3442](https://github.com/hatchet-dev/hatchet/pull/3442)


## [0.83.17] - 2026-04-07

### Added
- Feat--offers by @grutt in [#3511](https://github.com/hatchet-dev/hatchet/pull/3511)


## [0.83.15] - 2026-04-03

### Added
- Exchange token mechanism and CORs headers by @abelanger5 in [#3405](https://github.com/hatchet-dev/hatchet/pull/3405)
- Control plane phase 2, frontend changes by @abelanger5 in [#3536](https://github.com/hatchet-dev/hatchet/pull/3536)

### Fixed
- OTel trace lookup insert deadlock by @mrkaye97 in [#3542](https://github.com/hatchet-dev/hatchet/pull/3542)


## [0.83.14] - 2026-04-03

### Fixed
- Properly list workflows in dropdown by @mrkaye97 in [#3534](https://github.com/hatchet-dev/hatchet/pull/3534)


## [0.83.13] - 2026-04-03

### Added
- Enable event log on core db by @mrkaye97 in [#3537](https://github.com/hatchet-dev/hatchet/pull/3537)
- Add output tab to workflow run details by @grutt in [#3393](https://github.com/hatchet-dev/hatchet/pull/3393)

### Fixed
- Fix--dag-nesting by @grutt in [#3447](https://github.com/hatchet-dev/hatchet/pull/3447)


## [0.83.10] - 2026-04-01

### Fixed
- No self-referencing spans by @grutt in [#3445](https://github.com/hatchet-dev/hatchet/pull/3445)


## [0.83.9] - 2026-04-01

### Fixed
- Otel-bug-bash by @grutt in [#3389](https://github.com/hatchet-dev/hatchet/pull/3389)


## [0.83.8] - 2026-04-01

### Changed
- Update management API swagger with audit logs endpoint by @mnafees in [#3414](https://github.com/hatchet-dev/hatchet/pull/3414)


## [0.83.7] - 2026-04-01

### Changed
- Backend proxy for getting feature flags by @mrkaye97 in [#3437](https://github.com/hatchet-dev/hatchet/pull/3437)


## [0.83.5] - 2026-04-01

### Added
- Log filtering by workflow by @mrkaye97 in [#3435](https://github.com/hatchet-dev/hatchet/pull/3435)

### Changed
- Do not error out for historical run data for workflows and workflow versions that may have been deleted by @mnafees in [#3403](https://github.com/hatchet-dev/hatchet/pull/3403)

### Fixed
- Flaky Cypress test by @mrkaye97 in [#3436](https://github.com/hatchet-dev/hatchet/pull/3436)


## [0.83.4] - 2026-03-26

### Added
- Add more detailed logging to in memory advisory lock, configurable timeout by @juliusgeo in [#3408](https://github.com/hatchet-dev/hatchet/pull/3408)


## [0.83.3] - 2026-03-25

### Changed
- (fix): Make sure sqlc generates TotalReads as int64 by @juliusgeo in [#3404](https://github.com/hatchet-dev/hatchet/pull/3404)


## [0.83.2] - 2026-03-25

### Added
- Add in memory, keyed, locking queue to protect RunConcurrencyStrategy by @juliusgeo in [#3384](https://github.com/hatchet-dev/hatchet/pull/3384)


## [0.83.1] - 2026-03-25

### Added
- Add timeout context to OTel shutdown by @juliusgeo in [#3366](https://github.com/hatchet-dev/hatchet/pull/3366)

### Changed
- Use inserted at for proper indexing in DAG replay queries by @mnafees in [#3399](https://github.com/hatchet-dev/hatchet/pull/3399)

### Fixed
- Panic on lookup table create by @grutt in [#3385](https://github.com/hatchet-dev/hatchet/pull/3385)


## [0.83.0] - 2026-03-24

### Added
- Enable tenant-scoped logs view by @abelanger5 in [#3381](https://github.com/hatchet-dev/hatchet/pull/3381)

### Changed
- Get rid of jsonschema for security fix by @mnafees in [#3379](https://github.com/hatchet-dev/hatchet/pull/3379)


## [0.82.3] - 2026-03-23

### Changed
- Hide log-level view by @abelanger5 in [#3374](https://github.com/hatchet-dev/hatchet/pull/3374)

### Fixed
- Bump engine ver by @mrkaye97 in [#3373](https://github.com/hatchet-dev/hatchet/pull/3373)


## [0.82.2] - 2026-03-23

### Fixed
- Write engine spans to lookup table by @grutt in [#3371](https://github.com/hatchet-dev/hatchet/pull/3371)


## [0.82.1] - 2026-03-23

### Changed
- Move OTel tables to OLAP repo by @mnafees in [#3369](https://github.com/hatchet-dev/hatchet/pull/3369)
- Don't flicker the runs table when refetching by @TehShrike in [#3367](https://github.com/hatchet-dev/hatchet/pull/3367)


## [0.82.0] - 2026-03-23

### Added
- Pay as you go pricing by @grutt in [#3353](https://github.com/hatchet-dev/hatchet/pull/3353)

### Changed
- Observability overhaul + traces support by @mnafees in [#3213](https://github.com/hatchet-dev/hatchet/pull/3213)
- Improve error message when failing to send task to worker by @juliusgeo in [#3350](https://github.com/hatchet-dev/hatchet/pull/3350)
- Some fixes for load test plot generation by @juliusgeo in [#3338](https://github.com/hatchet-dev/hatchet/pull/3338)

### Removed
- Remove server ShutdownWait by @juliusgeo in [#3351](https://github.com/hatchet-dev/hatchet/pull/3351)

### Fixed
- Fallback when Tenant ID is zero UUID by @gregfurman in [#3349](https://github.com/hatchet-dev/hatchet/pull/3349)


## [0.81.2] - 2026-03-19

### Added
- Add min height to logs chart by @abelanger5 in [#3337](https://github.com/hatchet-dev/hatchet/pull/3337)
- Add latency plots to load test by @juliusgeo in [#3259](https://github.com/hatchet-dev/hatchet/pull/3259)

### Changed
- Only return non-archived tenants in TenantMember queries by @TehShrike in [#3317](https://github.com/hatchet-dev/hatchet/pull/3317)


## [0.81.0] - 2026-03-18

### Added
- Tenant-scoped logs view by @abelanger5 in [#3307](https://github.com/hatchet-dev/hatchet/pull/3307)


## [0.80.10] - 2026-03-18

### Changed
- Separate out partition cleanup helpers for visibility by @juliusgeo in [#3321](https://github.com/hatchet-dev/hatchet/pull/3321)


## [0.80.9] - 2026-03-18

### Fixed
- Honor active docker context when DOCKER_HOST is unset by @BloggerBust in [#3251](https://github.com/hatchet-dev/hatchet/pull/3251)


## [0.80.8] - 2026-03-18

### Changed
- Guarantee that organization tenants will always be an array by @TehShrike in [#3316](https://github.com/hatchet-dev/hatchet/pull/3316)
- New organizations+tenants screen by @TehShrike in [#3198](https://github.com/hatchet-dev/hatchet/pull/3198)
- Improved invitation accept screen by @TehShrike in [#3151](https://github.com/hatchet-dev/hatchet/pull/3151)

### Removed
- Remove dispatch backlog, replace with timeout lock acquisition by @juliusgeo in [#3290](https://github.com/hatchet-dev/hatchet/pull/3290)

### Fixed
- Cleanup orphaned metrics by @grutt in [#3300](https://github.com/hatchet-dev/hatchet/pull/3300)


## [0.80.4] - 2026-03-17

### Added
- Add `workflow_run_external_id` to trigger run ack proto by @mrkaye97 in [#3299](https://github.com/hatchet-dev/hatchet/pull/3299)

### Fixed
- Enhance SQL query name extraction in otel tracer and fallback by @grutt in [#3277](https://github.com/hatchet-dev/hatchet/pull/3277)
- Silence tenant alert error by @grutt in [#3298](https://github.com/hatchet-dev/hatchet/pull/3298)


## [0.80.3] - 2026-03-16

### Added
- Add cleanup module to handle graceful shutdown, improve logging experience by @juliusgeo in [#3260](https://github.com/hatchet-dev/hatchet/pull/3260)
- Durable Execution Revamp by @mrkaye97 in [#2954](https://github.com/hatchet-dev/hatchet/pull/2954)

### Fixed
- Remove join in currency queries by @grutt in [#3294](https://github.com/hatchet-dev/hatchet/pull/3294)


## [0.79.44] - 2026-03-15

### Added
- Add support for additional RBAC configruation via YAML configuration by @grutt in [#3285](https://github.com/hatchet-dev/hatchet/pull/3285)

### Changed
- Gracefully handle empty bulk scheduled deletes by @avirajkhare00 in [#3279](https://github.com/hatchet-dev/hatchet/pull/3279)

### Fixed
- Sort registered workflows on the worker page by @mrkaye97 in [#3284](https://github.com/hatchet-dev/hatchet/pull/3284)
- Validate the presence of null unicode in output by @gregfurman in [#3164](https://github.com/hatchet-dev/hatchet/pull/3164)
- Configure logger for Hatchet client by @gregfurman in [#3046](https://github.com/hatchet-dev/hatchet/pull/3046)


## [0.79.43] - 2026-03-13

### Fixed
- Delete missing workers by @mrkaye97 in [#3273](https://github.com/hatchet-dev/hatchet/pull/3273)
- Dont double count wf by @grutt in [#3270](https://github.com/hatchet-dev/hatchet/pull/3270)


## [0.79.41] - 2026-03-13

### Changed
- Attempt to fix deadlock in `PollScheduledWorkflows` by scoping `FOR UPDATE` lock by @mnafees in [#3261](https://github.com/hatchet-dev/hatchet/pull/3261)
- Janky Fix: Extract input payloads for standalone tasks for dashboard by @mrkaye97 in [#3128](https://github.com/hatchet-dev/hatchet/pull/3128)


## [0.79.39] - 2026-03-13

### Fixed
- Error on max aggregate keys by @grutt in [#3267](https://github.com/hatchet-dev/hatchet/pull/3267)
- Otel config loader and trunc query name by @grutt in [#3266](https://github.com/hatchet-dev/hatchet/pull/3266)


## [0.79.38] - 2026-03-13

### Fixed
- Analytics no set by @grutt in [#3264](https://github.com/hatchet-dev/hatchet/pull/3264)
- Add user id to context on login by @grutt in [#3256](https://github.com/hatchet-dev/hatchet/pull/3256)


## [0.79.37] - 2026-03-12

### Fixed
- Dont burry analytics properties by @grutt in [#3254](https://github.com/hatchet-dev/hatchet/pull/3254)


## [0.79.36] - 2026-03-12

### Added
- Feat--consistent-analytics-events by @grutt in [#3239](https://github.com/hatchet-dev/hatchet/pull/3239)


## [0.79.35] - 2026-03-12

### Changed
- Pool gzip writers to reduce RabbitMQ message compression allocations by @mnafees in [#3103](https://github.com/hatchet-dev/hatchet/pull/3103)


## [0.79.34] - 2026-03-12

### Added
- Env var for stream event buffer timeout by @mrkaye97 in [#3223](https://github.com/hatchet-dev/hatchet/pull/3223)

### Changed
- Don't panic in AuthZ, bubble up instead by @juliusgeo in [#3238](https://github.com/hatchet-dev/hatchet/pull/3238)

### Fixed
- Failure after cancellation by @grutt in [#3243](https://github.com/hatchet-dev/hatchet/pull/3243)
- Fix owner invitation bug by @juliusgeo in [#3230](https://github.com/hatchet-dev/hatchet/pull/3230)


## [0.79.33] - 2026-03-10

### Changed
- RBAC v0 by @juliusgeo in [#3185](https://github.com/hatchet-dev/hatchet/pull/3185)

### Fixed
- Fix frontend build issues with latest cloud API by @mnafees in [#3216](https://github.com/hatchet-dev/hatchet/pull/3216)


## [0.79.29] - 2026-03-07

### Added
- Add callback support for tenant and tenant member updates by @abelanger5 in [#3201](https://github.com/hatchet-dev/hatchet/pull/3201)


## [0.79.23] - 2026-03-06

### Changed
- Pause workflow query is deprecated so remove the option from the frontend by @mnafees in [#3183](https://github.com/hatchet-dev/hatchet/pull/3183)


## [0.79.17] - 2026-03-04

### Fixed
- Go unexported type by @mrkaye97 in [#3160](https://github.com/hatchet-dev/hatchet/pull/3160)


## [0.79.16] - 2026-03-04

### Added
- Add queue to update scheduled cron triggers on-demand by @juliusgeo in [#3149](https://github.com/hatchet-dev/hatchet/pull/3149)


## [0.79.15] - 2026-03-04

### Added
- Dynamic worker label assign by @mrkaye97 in [#3137](https://github.com/hatchet-dev/hatchet/pull/3137)


## [0.79.14] - 2026-03-03

### Added
- Add seconds granularity to cron jobs by @juliusgeo in [#3136](https://github.com/hatchet-dev/hatchet/pull/3136)

### Changed
- Enable loadtest with PgBouncer by @mnafees in [#3143](https://github.com/hatchet-dev/hatchet/pull/3143)


## [0.79.13] - 2026-03-02

### Added
- User callback additional methods by @abelanger5 in [#3057](https://github.com/hatchet-dev/hatchet/pull/3057)


## [0.79.12] - 2026-02-28

### Fixed
- More small tenant switching + z index issues by @mrkaye97 in [#3124](https://github.com/hatchet-dev/hatchet/pull/3124)
- Rm z index for action dialog by @mrkaye97 in [#3120](https://github.com/hatchet-dev/hatchet/pull/3120)


## [0.79.11] - 2026-02-27

### Added
- New "create organization" and "create tenant" interfaces by @TehShrike in [#3068](https://github.com/hatchet-dev/hatchet/pull/3068)

### Fixed
- Modals should appear above the mobile sidebar by @TehShrike in [#3114](https://github.com/hatchet-dev/hatchet/pull/3114)


## [0.79.10] - 2026-02-26

### Fixed
- External ids by @mrkaye97 in [#3111](https://github.com/hatchet-dev/hatchet/pull/3111)


## [0.79.9] - 2026-02-26

### Added
- Add `ctx.WasSkipped` helper to the Go SDK by @mnafees in [#3094](https://github.com/hatchet-dev/hatchet/pull/3094)
- Add credit balance query and display in subscription component by @grutt in [#3107](https://github.com/hatchet-dev/hatchet/pull/3107)


## [0.79.7] - 2026-02-25

### Changed
- Non blocking ctx.Log with meaningful retries by @mnafees in [#3106](https://github.com/hatchet-dev/hatchet/pull/3106)


## [0.79.6] - 2026-02-25

### Changed
- [Go] Feat: Details Getter by @mrkaye97 in [#3105](https://github.com/hatchet-dev/hatchet/pull/3105)


## [0.79.5] - 2026-02-24

### Added
- Add env vars for max conn lifetime and idle time for pgx by @mnafees in [#3096](https://github.com/hatchet-dev/hatchet/pull/3096)


## [0.79.4] - 2026-02-24

### Added
- Add missing primary key to `"WorkflowTriggerCronRef"` by @mnafees in [#3086](https://github.com/hatchet-dev/hatchet/pull/3086)

### Fixed
- Fix cross-strategy slot contamination in chained concurrency gates by @mnafees in [#3089](https://github.com/hatchet-dev/hatchet/pull/3089)


## [0.79.3] - 2026-02-23

### Changed
- Make sure to use 60 seconds timeout for PutWorkflowVersion by @mnafees in [#3085](https://github.com/hatchet-dev/hatchet/pull/3085)


## [0.79.2] - 2026-02-23

### Fixed
- Move event log to a tab on the task run detail by @mrkaye97 in [#3067](https://github.com/hatchet-dev/hatchet/pull/3067)


## [0.79.1] - 2026-02-21

### Added
- Add new agent instruction tools by @grutt in [#3059](https://github.com/hatchet-dev/hatchet/pull/3059)

### Fixed
- Expand long logs on click instead of on hover by @mrkaye97 in [#3069](https://github.com/hatchet-dev/hatchet/pull/3069)
- Input in transformer by @mrkaye97 in [#3070](https://github.com/hatchet-dev/hatchet/pull/3070)


## [0.79.0] - 2026-02-20

### Added
- Runs commands by @abelanger5 in [#3058](https://github.com/hatchet-dev/hatchet/pull/3058)


## [0.78.30] - 2026-02-19

### Added
- Add migration for worker slot config index by @grutt in [#3062](https://github.com/hatchet-dev/hatchet/pull/3062)


## [0.78.27] - 2026-02-18

### Added
- New search bar component by @abelanger5 in [#2909](https://github.com/hatchet-dev/hatchet/pull/2909)
- Multiple slot types by @grutt in [#2927](https://github.com/hatchet-dev/hatchet/pull/2927)

### Fixed
- Fix--durable-slot-acquisition by @grutt in [#3048](https://github.com/hatchet-dev/hatchet/pull/3048)


## [0.78.26] - 2026-02-16

### Added
- Feat--llm-readable-docs by @grutt in [#3030](https://github.com/hatchet-dev/hatchet/pull/3030)
- Official Ruby SDK by @grutt in [#3004](https://github.com/hatchet-dev/hatchet/pull/3004)

### Changed
- Return event ID after successful webhook trigger by @mnafees in [#3039](https://github.com/hatchet-dev/hatchet/pull/3039)


## [0.78.25] - 2026-02-13

### Added
- Add python and typescript webhook client by @jishnundth in [#2959](https://github.com/hatchet-dev/hatchet/pull/2959)

### Changed
- [hotfix] Corrected custom value meter for resource limit by @mnafees in [#3021](https://github.com/hatchet-dev/hatchet/pull/3021)
- New `UpdateLimits` method for `TenantResourceLimit` table by @mnafees in [#2895](https://github.com/hatchet-dev/hatchet/pull/2895)


## [0.78.23] - 2026-02-13

### Changed
- More deprecation messages for older Go SDKs by @mnafees in [#3006](https://github.com/hatchet-dev/hatchet/pull/3006)

### Fixed
- DAG height on TUI by @abelanger5 in [#3019](https://github.com/hatchet-dev/hatchet/pull/3019)
- Remove null bytes from error message to prevent db crash by @mrkaye97 in [#3010](https://github.com/hatchet-dev/hatchet/pull/3010)


## [0.78.22] - 2026-02-11

### Changed
- Truncate to first 10k characters of log line in Go SDK by @mnafees in [#2998](https://github.com/hatchet-dev/hatchet/pull/2998)


## [0.78.21] - 2026-02-11

### Added
- Reduced cold starts for new workers and queues by @abelanger5 in [#2969](https://github.com/hatchet-dev/hatchet/pull/2969)


## [0.78.20] - 2026-02-11

### Added
- Add support for Svix webhooks by @mnafees in [#2996](https://github.com/hatchet-dev/hatchet/pull/2996)


## [0.78.19] - 2026-02-11

### Added
- Log on delayed heartbeat by @grutt in [#2994](https://github.com/hatchet-dev/hatchet/pull/2994)


## [0.78.18] - 2026-02-10

### Changed
- Do not replay invalid tasks by @mnafees in [#2976](https://github.com/hatchet-dev/hatchet/pull/2976)


## [0.78.16] - 2026-02-10

### Changed
- [hotfix] Fix Docker frontend build issue by @mnafees in [#2983](https://github.com/hatchet-dev/hatchet/pull/2983)
- [hotfix] Fix `BillingRequired` component by @mnafees in [#2982](https://github.com/hatchet-dev/hatchet/pull/2982)
- Tasks marked as skipped in the UI by @mnafees in [#2978](https://github.com/hatchet-dev/hatchet/pull/2978)
- Mark old v0 and generics-based v1 Go SDK methods as deprecated by @mnafees in [#2962](https://github.com/hatchet-dev/hatchet/pull/2962)


## [0.78.12] - 2026-02-05

### Fixed
- Check `uuid.Nil` when creating or groups too by @mrkaye97 in [#2958](https://github.com/hatchet-dev/hatchet/pull/2958)
- Wrapped types in the Go SDK by @abelanger5 in [#2957](https://github.com/hatchet-dev/hatchet/pull/2957)


## [0.78.11] - 2026-02-05

### Fixed
- Always generate UUID for OrGroup by @mrkaye97 in [#2955](https://github.com/hatchet-dev/hatchet/pull/2955)


## [0.78.10] - 2026-02-05

### Fixed
- More panics by @mrkaye97 in [#2945](https://github.com/hatchet-dev/hatchet/pull/2945)


## [0.78.9] - 2026-02-04

### Fixed
- Make uuid optional for desired worker by @mrkaye97 in [#2946](https://github.com/hatchet-dev/hatchet/pull/2946)


## [0.78.8] - 2026-02-04

### Changed
- Go SDK gRPC client reconnection improvements by @mnafees in [#2934](https://github.com/hatchet-dev/hatchet/pull/2934)

### Fixed
- UUID Panics by @mrkaye97 in [#2944](https://github.com/hatchet-dev/hatchet/pull/2944)


## [0.78.7] - 2026-02-04

### Added
- Extend webhook support for scope_expression and payload by @jishnundth in [#2874](https://github.com/hatchet-dev/hatchet/pull/2874)


## [0.78.6] - 2026-02-04

### Fixed
- Don't cast user id in session.Values by @mrkaye97 in [#2937](https://github.com/hatchet-dev/hatchet/pull/2937)


## [0.78.5] - 2026-02-04

### Fixed
- Resource ID type by @mrkaye97 in [#2929](https://github.com/hatchet-dev/hatchet/pull/2929)


## [0.78.4] - 2026-02-04

### Changed
- Use typed maps by @abelanger5 in [#2928](https://github.com/hatchet-dev/hatchet/pull/2928)


## [0.78.3] - 2026-02-03

### Fixed
- Explicit use of tx in olap readPayloads by @abelanger5 in [#2925](https://github.com/hatchet-dev/hatchet/pull/2925)


## [0.78.2] - 2026-02-03

### Fixed
- Startup for optimistic scheduler by @abelanger5 in [#2924](https://github.com/hatchet-dev/hatchet/pull/2924)


## [0.78.0] - 2026-02-03

### Added
- Durable user event log by @abelanger5 in [#2861](https://github.com/hatchet-dev/hatchet/pull/2861)
- Email alert support via SMTP by @gregfurman in [#2868](https://github.com/hatchet-dev/hatchet/pull/2868)

### Changed
- Attempt II at removing `pgtype.UUID` everywhere + convert string UUIDs into `uuid.UUID` by @mrkaye97 in [#2894](https://github.com/hatchet-dev/hatchet/pull/2894)
- Management tokens that never expire by @mnafees in [#2889](https://github.com/hatchet-dev/hatchet/pull/2889)

### Fixed
- Add back SendTemplateEmail methods, fix postmark emails by @abelanger5 in [#2910](https://github.com/hatchet-dev/hatchet/pull/2910)
- Dag update distinct locks by @grutt in [#2903](https://github.com/hatchet-dev/hatchet/pull/2903)
- Only fetch finalized workflow runs by @grutt in [#2896](https://github.com/hatchet-dev/hatchet/pull/2896)
- Compute payload size correctly for pg_notify by @abelanger5 in [#2873](https://github.com/hatchet-dev/hatchet/pull/2873)
- Orphaned inactive queues by @grutt in [#2893](https://github.com/hatchet-dev/hatchet/pull/2893)


## [0.77.37] - 2026-01-30

### Changed
- Log Search Frontend, Part II by @mrkaye97 in [#2886](https://github.com/hatchet-dev/hatchet/pull/2886)


## [0.77.36] - 2026-01-29

### Changed
- Make sure we query for pending invites in the create tenant page by @mnafees in [#2883](https://github.com/hatchet-dev/hatchet/pull/2883)


## [0.77.35] - 2026-01-29

### Added
- Log Search Frontend, Part I by @mrkaye97 in [#2830](https://github.com/hatchet-dev/hatchet/pull/2830)


## [0.77.34] - 2026-01-29

### Fixed
- Tenant resource limit resource col migration by @grutt in [#2885](https://github.com/hatchet-dev/hatchet/pull/2885)


## [0.77.33] - 2026-01-29

### Added
- Workflow input JSON schema in trigger preview by @mrkaye97 in [#2851](https://github.com/hatchet-dev/hatchet/pull/2851)

### Fixed
- Validate json at edges and dont retry on invalid by @grutt in [#2882](https://github.com/hatchet-dev/hatchet/pull/2882)
- Typo by @grutt in [#2875](https://github.com/hatchet-dev/hatchet/pull/2875)
- Big int alignment for cleanup function by @grutt in [#2877](https://github.com/hatchet-dev/hatchet/pull/2877)


## [0.77.32] - 2026-01-28

### Added
- OTel Collector by @mrkaye97 in [#2863](https://github.com/hatchet-dev/hatchet/pull/2863)


## [0.77.30] - 2026-01-27

### Fixed
- Make metrics in graph align with badge metrics by @mrkaye97 in [#2858](https://github.com/hatchet-dev/hatchet/pull/2858)


## [0.77.25] - 2026-01-26

### Removed
- Revert "fix: make point metrics line up with badges " by @mrkaye97 in [#2857](https://github.com/hatchet-dev/hatchet/pull/2857)


## [0.77.24] - 2026-01-26

### Fixed
- Fix with coalesce by @mnafees in [#2856](https://github.com/hatchet-dev/hatchet/pull/2856)
- Make point metrics line up with badges by @mrkaye97 in [#2739](https://github.com/hatchet-dev/hatchet/pull/2739)


## [0.77.23] - 2026-01-26

### Added
- Add order by direction param to v1LogLineList by @mrkaye97 in [#2849](https://github.com/hatchet-dev/hatchet/pull/2849)

### Changed
- More tenant related repo methods by @mnafees in [#2854](https://github.com/hatchet-dev/hatchet/pull/2854)


## [0.77.21] - 2026-01-23

### Changed
- [Go] Feat: Webhooks feature client for the Go SDK by @mrkaye97 in [#2792](https://github.com/hatchet-dev/hatchet/pull/2792)


## [0.77.16] - 2026-01-21

### Fixed
- List concurrency strategies queries by @grutt in [#2838](https://github.com/hatchet-dev/hatchet/pull/2838)


## [0.77.13] - 2026-01-21

### Added
- Add search and levels to logs API by @mrkaye97 in [#2835](https://github.com/hatchet-dev/hatchet/pull/2835)

### Changed
- Take exclusive lock by @mnafees in [#2837](https://github.com/hatchet-dev/hatchet/pull/2837)
- Updates by @grutt in [#2827](https://github.com/hatchet-dev/hatchet/pull/2827)

### Fixed
- Flaky integration test by @grutt in [#2834](https://github.com/hatchet-dev/hatchet/pull/2834)


## [0.77.9] - 2026-01-21

### Changed
- [hotfix] Indicate unauthorized access to billing page for non-owners by @mnafees in [#2833](https://github.com/hatchet-dev/hatchet/pull/2833)


## [0.77.5] - 2026-01-20

### Changed
- Have log line lookups use external id by @abelanger5 in [#2822](https://github.com/hatchet-dev/hatchet/pull/2822)


## [0.77.1] - 2026-01-19

### Fixed
- Fix naming of migration by @mnafees in [#2819](https://github.com/hatchet-dev/hatchet/pull/2819)


## [0.77.0] - 2026-01-19

### Changed
- Billing changes by @mnafees in [#2643](https://github.com/hatchet-dev/hatchet/pull/2643)


## [0.75.4] - 2026-01-18

### Changed
- Shrink task metrics chart a bunch by @mrkaye97 in [#2738](https://github.com/hatchet-dev/hatchet/pull/2738)

### Fixed
- Typescript post-quickstart by @abelanger5 in [#2809](https://github.com/hatchet-dev/hatchet/pull/2809)


## [0.75.3] - 2026-01-16

### Fixed
- Minor quickstart issues by @abelanger5 in [#2807](https://github.com/hatchet-dev/hatchet/pull/2807)


## [0.75.2] - 2026-01-16

### Added
- Update quickstarts with package manager support, e2e tests for quickstarts by @abelanger5 in [#2801](https://github.com/hatchet-dev/hatchet/pull/2801)

### Changed
- New onboarding flow by @mnafees in [#2757](https://github.com/hatchet-dev/hatchet/pull/2757)
- Auth front-end changes by @sebastiangraz in [#2802](https://github.com/hatchet-dev/hatchet/pull/2802)


## [0.75.1] - 2026-01-15

### Changed
- Small fixes/improvements to CLI logic by @abelanger5 in [#2793](https://github.com/hatchet-dev/hatchet/pull/2793)


## [0.75.0] - 2026-01-13

### Added
- Hatchet cli by @abelanger5 in [#2701](https://github.com/hatchet-dev/hatchet/pull/2701)


## [0.74.14] - 2026-01-12

### Fixed
- Concurrency display on workflow page by @mrkaye97 in [#2780](https://github.com/hatchet-dev/hatchet/pull/2780)


## [0.74.13] - 2026-01-12

### Added
- Add additional meta to the run detail getter by @mrkaye97 in [#2770](https://github.com/hatchet-dev/hatchet/pull/2770)

### Fixed
- Regression on v0 PutWorkflow for scheduling timeout by @abelanger5 in [#2779](https://github.com/hatchet-dev/hatchet/pull/2779)


## [0.74.12] - 2026-01-11

### Fixed
- Statement timeout by @mrkaye97 in [#2774](https://github.com/hatchet-dev/hatchet/pull/2774)
- Actually reconnect to postgres if conn fails. by @m-kostrzewa in [#2772](https://github.com/hatchet-dev/hatchet/pull/2772)


## [0.74.11] - 2026-01-10

### Added
- Run detail getter on the engine by @mrkaye97 in [#2725](https://github.com/hatchet-dev/hatchet/pull/2725)

### Fixed
- Worker id by @mrkaye97 in [#2773](https://github.com/hatchet-dev/hatchet/pull/2773)


## [0.74.8] - 2026-01-08

### Fixed
- Better error on deprecated endpoints by @abelanger5 in [#2763](https://github.com/hatchet-dev/hatchet/pull/2763)
- Chunk and recursively retry too-large message sends by @mrkaye97 in [#2761](https://github.com/hatchet-dev/hatchet/pull/2761)


## [0.74.7] - 2026-01-07

### Fixed
- Un-hard-code location by @mrkaye97 in [#2760](https://github.com/hatchet-dev/hatchet/pull/2760)


## [0.74.6] - 2026-01-07

### Fixed
- Payload location issue by @mrkaye97 in [#2759](https://github.com/hatchet-dev/hatchet/pull/2759)
- Child runs counts missing filter by @mrkaye97 in [#2744](https://github.com/hatchet-dev/hatchet/pull/2744)


## [0.74.5] - 2026-01-06

### Changed
- Send `create:user` Event from OAuth Flow by @undrash in [#2683](https://github.com/hatchet-dev/hatchet/pull/2683)

### Removed
- UI version removal by @abelanger5 in [#2756](https://github.com/hatchet-dev/hatchet/pull/2756)


## [0.74.4] - 2026-01-06

### Changed
- Use try advisory for replat tasks by @mnafees in [#2755](https://github.com/hatchet-dev/hatchet/pull/2755)


## [0.74.3] - 2026-01-06

### Changed
- Set a connection-level statement timeout by @abelanger5 in [#2750](https://github.com/hatchet-dev/hatchet/pull/2750)
- Move v1 packages, remove webhook worker references by @abelanger5 in [#2749](https://github.com/hatchet-dev/hatchet/pull/2749)

### Fixed
- More frontend work - table cleanup, worker detail page improvements, etc. by @mrkaye97 in [#2746](https://github.com/hatchet-dev/hatchet/pull/2746)


## [0.74.2] - 2025-12-31

### Changed
- Consolidate repository methods by @abelanger5 in [#2730](https://github.com/hatchet-dev/hatchet/pull/2730)


## [0.74.1] - 2025-12-31

### Changed
- Remove v0-exclusive database queries by @abelanger5 in [#2729](https://github.com/hatchet-dev/hatchet/pull/2729)
- Remove v0 paths from codebase by @abelanger5 in [#2728](https://github.com/hatchet-dev/hatchet/pull/2728)

### Fixed
- Handle panic by @mrkaye97 in [#2732](https://github.com/hatchet-dev/hatchet/pull/2732)
- Frozen refetch state gets stuck by @mrkaye97 in [#2736](https://github.com/hatchet-dev/hatchet/pull/2736)
- Remove action dropdown, fix a couple broken tooltips by @mrkaye97 in [#2737](https://github.com/hatchet-dev/hatchet/pull/2737)


## [0.73.110] - 2025-12-26

### Fixed
- Layout by @mrkaye97 in [#2724](https://github.com/hatchet-dev/hatchet/pull/2724)


## [0.73.109] - 2025-12-26

### Fixed
- Tenant invite accept flow by @mrkaye97 in [#2723](https://github.com/hatchet-dev/hatchet/pull/2723)
- Goroutine to periodically extend lease during reconciliation by @mrkaye97 in [#2722](https://github.com/hatchet-dev/hatchet/pull/2722)


## [0.73.108] - 2025-12-26

### Fixed
- Rename migration by @mrkaye97 in [#2721](https://github.com/hatchet-dev/hatchet/pull/2721)


## [0.73.107] - 2025-12-26

### Changed
- Publish `COULD_NOT_SEND_TO_WORKER` OLAP event due to worker backlog by @mnafees in [#2710](https://github.com/hatchet-dev/hatchet/pull/2710)
- Reuse timers for delayed semaphore release in MQ buffers by @mnafees in [#2691](https://github.com/hatchet-dev/hatchet/pull/2691)
- Msgqueue msg IDs as constants for ease of navigation and readability by @mnafees in [#2692](https://github.com/hatchet-dev/hatchet/pull/2692)

### Removed
- Revert "Revert "chore: run list query optimizations " " by @mrkaye97 in [#2720](https://github.com/hatchet-dev/hatchet/pull/2720)

### Fixed
- Static rate limits resetting to zero by @mrkaye97 in [#2714](https://github.com/hatchet-dev/hatchet/pull/2714)
- Minor fe bugs and nits by @grutt in [#2711](https://github.com/hatchet-dev/hatchet/pull/2711)


## [0.73.106] - 2025-12-23

### Added
- Improved navigation by @grutt in [#2704](https://github.com/hatchet-dev/hatchet/pull/2704)
- Hatchet Metrics Monitoring, I by @mrkaye97 in [#2699](https://github.com/hatchet-dev/hatchet/pull/2699)

### Removed
- Revert "chore: run list query optimizations " by @mrkaye97 in [#2708](https://github.com/hatchet-dev/hatchet/pull/2708)

### Fixed
- Rm cleanup logic for now by @mrkaye97 in [#2707](https://github.com/hatchet-dev/hatchet/pull/2707)
- WithToken should override environment variable by @abelanger5 in [#2706](https://github.com/hatchet-dev/hatchet/pull/2706)
- Fix move fixture by @grutt in [#2705](https://github.com/hatchet-dev/hatchet/pull/2705)


## [0.73.104] - 2025-12-23

### Added
- Improved error boundaries by @grutt in [#2689](https://github.com/hatchet-dev/hatchet/pull/2689)
- Bulk management schedules by @grutt in [#2687](https://github.com/hatchet-dev/hatchet/pull/2687)

### Fixed
- Fix  layout bugs by @grutt in [#2703](https://github.com/hatchet-dev/hatchet/pull/2703)
- Dead alert links by @mrkaye97 in [#2688](https://github.com/hatchet-dev/hatchet/pull/2688)


## [0.73.103] - 2025-12-22

### Fixed
- Dynamically-sized chunks on payload read by @mrkaye97 in [#2700](https://github.com/hatchet-dev/hatchet/pull/2700)


## [0.73.102] - 2025-12-22

### Removed
- Revert "Feat: Hatchet Metrics Monitoring, I " by @mrkaye97 in [#2698](https://github.com/hatchet-dev/hatchet/pull/2698)

### Fixed
- Table name by @mrkaye97 in [#2697](https://github.com/hatchet-dev/hatchet/pull/2697)


## [0.73.101] - 2025-12-22

### Added
- Hatchet Metrics Monitoring, I by @mrkaye97 in [#2480](https://github.com/hatchet-dev/hatchet/pull/2480)


## [0.73.99] - 2025-12-22

### Fixed
- Last bits of payload job cleanup by @mrkaye97 in [#2690](https://github.com/hatchet-dev/hatchet/pull/2690)
- Rare cases of duplicate writes causing stuck updates by @abelanger5 in [#2681](https://github.com/hatchet-dev/hatchet/pull/2681)
- Filter + pagination state handling hack by @mrkaye97 in [#2682](https://github.com/hatchet-dev/hatchet/pull/2682)


## [0.73.98] - 2025-12-17

### Added
- New event getter + janky v0 fix by @mrkaye97 in [#2667](https://github.com/hatchet-dev/hatchet/pull/2667)

### Changed
- Remove inline button styles where possible by @mrkaye97 in [#2671](https://github.com/hatchet-dev/hatchet/pull/2671)


## [0.73.97] - 2025-12-16

### Fixed
- Payload List Index Performance by @mrkaye97 in [#2669](https://github.com/hatchet-dev/hatchet/pull/2669)
- Root redirect with last tenant atom by @mrkaye97 in [#2664](https://github.com/hatchet-dev/hatchet/pull/2664)


## [0.73.96] - 2025-12-15

### Fixed
- Pagination by bounds by @mrkaye97 in [#2654](https://github.com/hatchet-dev/hatchet/pull/2654)


## [0.73.94] - 2025-12-11

### Changed
- Write to S3 outside of goroutine by @mrkaye97 in [#2646](https://github.com/hatchet-dev/hatchet/pull/2646)


## [0.73.93] - 2025-12-11

### Fixed
- OLAP Immediate Offloads by @mrkaye97 in [#2644](https://github.com/hatchet-dev/hatchet/pull/2644)


## [0.73.92] - 2025-12-11

### Added
- Add oldest queued + running jobs to task stats by @mrkaye97 in [#2638](https://github.com/hatchet-dev/hatchet/pull/2638)


## [0.73.91] - 2025-12-10

### Added
- Parallelize replication from PG -> External by @mrkaye97 in [#2637](https://github.com/hatchet-dev/hatchet/pull/2637)


## [0.73.90] - 2025-12-10

### Fixed
- Global Lease for OLAP by @mrkaye97 in [#2635](https://github.com/hatchet-dev/hatchet/pull/2635)


## [0.73.89] - 2025-12-10

### Fixed
- Query logic bug by @mrkaye97 in [#2631](https://github.com/hatchet-dev/hatchet/pull/2631)


## [0.73.88] - 2025-12-10

### Added
- Add support for slack slash commands in webhook by @sidpremkumar in [#2630](https://github.com/hatchet-dev/hatchet/pull/2630)

### Fixed
- Don't reset offset if a new process acquires lease by @mrkaye97 in [#2628](https://github.com/hatchet-dev/hatchet/pull/2628)


## [0.73.87] - 2025-12-09

### Added
- OLAP Payload Cutover Job by @mrkaye97 in [#2618](https://github.com/hatchet-dev/hatchet/pull/2618)


## [0.73.86] - 2025-12-08

### Changed
- Simplify external store signature by @mrkaye97 in [#2616](https://github.com/hatchet-dev/hatchet/pull/2616)


## [0.73.85] - 2025-12-08

### Changed
- Update Expression Page + Slack Webhook Onboarding by @sidpremkumar in [#2614](https://github.com/hatchet-dev/hatchet/pull/2614)

### Fixed
- Fix slack challenge + interactive webhook by @sidpremkumar in [#2612](https://github.com/hatchet-dev/hatchet/pull/2612)


## [0.73.84] - 2025-12-08

### Added
- Process all old partitions in a loop by @mrkaye97 in [#2613](https://github.com/hatchet-dev/hatchet/pull/2613)


## [0.73.83] - 2025-12-08

### Added
- Add guide for downgrading versions by @mnafees in [#2588](https://github.com/hatchet-dev/hatchet/pull/2588)


## [0.73.82] - 2025-12-05

### Fixed
- Add validation by @mrkaye97 in [#2610](https://github.com/hatchet-dev/hatchet/pull/2610)


## [0.73.81] - 2025-12-05

### Fixed
- Leasing for payload job by @mrkaye97 in [#2609](https://github.com/hatchet-dev/hatchet/pull/2609)


## [0.73.80] - 2025-12-05

### Added
- Job for payload cutovers to external by @mrkaye97 in [#2586](https://github.com/hatchet-dev/hatchet/pull/2586)
- Dlq for dispatcher queues by @abelanger5 in [#2600](https://github.com/hatchet-dev/hatchet/pull/2600)

### Changed
- Initialize concurrency keys slice for replayed tasks by @mnafees in [#2549](https://github.com/hatchet-dev/hatchet/pull/2549)

### Fixed
- Fix double toast on sidebar by @sidpremkumar in [#2607](https://github.com/hatchet-dev/hatchet/pull/2607)


## [0.73.78] - 2025-12-03

### Changed
- Cross-Domain Tracking and Analytics Refactoring by @undrash in [#2587](https://github.com/hatchet-dev/hatchet/pull/2587)

### Fixed
- Prevent large worker gRPC stream backlogs by @abelanger5 in [#2597](https://github.com/hatchet-dev/hatchet/pull/2597)
- Don't trigger posthog Pageview on query param changes by @undrash in [#2598](https://github.com/hatchet-dev/hatchet/pull/2598)
- Load shed on slow worker backlogs by @abelanger5 in [#2595](https://github.com/hatchet-dev/hatchet/pull/2595)


## [0.73.76] - 2025-12-02

### Fixed
- Move check for large payloads to after json.Marshal by @abelanger5 in [#2594](https://github.com/hatchet-dev/hatchet/pull/2594)


## [0.73.75] - 2025-12-02

### Fixed
- Ensure that slow worker doesn't interrupt dispatcher, guard large RabbitMQ pubs by @abelanger5 in [#2591](https://github.com/hatchet-dev/hatchet/pull/2591)
- GetLatestWorkflowVersionForWorkflows by @grutt in [#2590](https://github.com/hatchet-dev/hatchet/pull/2590)


## [0.73.74] - 2025-11-28

### Added
- Add sent to worker event in the dispatcher by @mrkaye97 in [#2584](https://github.com/hatchet-dev/hatchet/pull/2584)


## [0.73.73] - 2025-11-26

### Added
- Add gzip compression by @sidpremkumar in [#2539](https://github.com/hatchet-dev/hatchet/pull/2539)

### Fixed
- Noisy Payload Error by @mrkaye97 in [#2561](https://github.com/hatchet-dev/hatchet/pull/2561)


## [0.73.72] - 2025-11-26

### Added
- Add tooltip showing full step name on hover by @mrkaye97 in [#2563](https://github.com/hatchet-dev/hatchet/pull/2563)

### Changed
- Analyze v1 lookup table by @grutt in [#2568](https://github.com/hatchet-dev/hatchet/pull/2568)
- Optimize UUID sqlchelpers by @mnafees in [#2532](https://github.com/hatchet-dev/hatchet/pull/2532)
- Add spans to worker list handler by @grutt in [#2554](https://github.com/hatchet-dev/hatchet/pull/2554)

### Removed
- Revert "optimize UUID sqlchelpers " by @mrkaye97 in [#2571](https://github.com/hatchet-dev/hatchet/pull/2571)

### Fixed
- Query optimization get latest workflow version by @grutt in [#2576](https://github.com/hatchet-dev/hatchet/pull/2576)
- OLAP Task Event Dual Write Bug by @mrkaye97 in [#2572](https://github.com/hatchet-dev/hatchet/pull/2572)
- Add whitespace-pre to log by @H01001000 in [#2555](https://github.com/hatchet-dev/hatchet/pull/2555)


## [0.73.71] - 2025-11-21

### Changed
- Common pgxpool afterconnect method by @mnafees in [#2553](https://github.com/hatchet-dev/hatchet/pull/2553)


## [0.73.70] - 2025-11-20

### Changed
- [Go SDK] Resubscribe and get a new listener stream when gRPC connections fail by @mnafees in [#2544](https://github.com/hatchet-dev/hatchet/pull/2544)


## [0.73.68] - 2025-11-18

### Fixed
- Use sessionStorage instead of localStorage by @mrkaye97 in [#2541](https://github.com/hatchet-dev/hatchet/pull/2541)


## [0.73.67] - 2025-11-18

### Added
- Initial cross-domain identify setup by @mrkaye97 in [#2533](https://github.com/hatchet-dev/hatchet/pull/2533)

### Fixed
- Small scheduler optimizations by @abelanger5 in [#2426](https://github.com/hatchet-dev/hatchet/pull/2426)


## [0.73.66] - 2025-11-17

### Added
- REST API Instrumentation by @mrkaye97 in [#2529](https://github.com/hatchet-dev/hatchet/pull/2529)

### Fixed
- Revert n+1 queries on the list API by @mrkaye97 in [#2531](https://github.com/hatchet-dev/hatchet/pull/2531)


## [0.73.65] - 2025-11-14

### Changed
- Case on conflict for v1_statuses_olap entry by @mnafees in [#2528](https://github.com/hatchet-dev/hatchet/pull/2528)
- Attempt to fix pgx multi dimensional slice reflection error #1 by @mnafees in [#2523](https://github.com/hatchet-dev/hatchet/pull/2523)
- Archive tenant modal flow by @mnafees in [#2509](https://github.com/hatchet-dev/hatchet/pull/2509)

### Fixed
- Fix seq scan in `PollCronSchedules` query by @mnafees in [#2524](https://github.com/hatchet-dev/hatchet/pull/2524)
- Log dupes by @mrkaye97 in [#2526](https://github.com/hatchet-dev/hatchet/pull/2526)
- Fix nil error in `handleTaskBulkAssignedTask` by @mnafees in [#2427](https://github.com/hatchet-dev/hatchet/pull/2427)


## [0.73.64] - 2025-11-12

### Changed
- [Go SDK] Case on worker labels for durable tasks by @mnafees in [#2511](https://github.com/hatchet-dev/hatchet/pull/2511)


## [0.73.63] - 2025-11-07

### Added
- Add pagination support for V1LogLineList by @jishnundth in [#2354](https://github.com/hatchet-dev/hatchet/pull/2354)

### Changed
- Immediate Payload Offloads OLAP Wiring by @mrkaye97 in [#2492](https://github.com/hatchet-dev/hatchet/pull/2492)


## [0.73.62] - 2025-11-07

### Changed
- Pass labels to durable worker by @mnafees in [#2504](https://github.com/hatchet-dev/hatchet/pull/2504)


## [0.73.61] - 2025-11-06

### Added
- Configurable OLAP status update size limits by @mrkaye97 in [#2499](https://github.com/hatchet-dev/hatchet/pull/2499)

### Fixed
- Propagate parent id through to `V1TaskSummary` properly by @mrkaye97 in [#2496](https://github.com/hatchet-dev/hatchet/pull/2496)


## [0.73.60] - 2025-11-04

### Fixed
- Deadlocks on trigger, olap prometheus background worker, otel improvements by @mnafees in [#2475](https://github.com/hatchet-dev/hatchet/pull/2475)


## [0.73.58] - 2025-11-02

### Added
- Add grpc otel spans, better tx debugging by @abelanger5 in [#2474](https://github.com/hatchet-dev/hatchet/pull/2474)

### Changed
- Update frontend onboarding steps by @sidpremkumar in [#2478](https://github.com/hatchet-dev/hatchet/pull/2478)
- [hotfix] Temporarily increase `TestLoadCLI` average threshold by @mnafees in [#2461](https://github.com/hatchet-dev/hatchet/pull/2461)

### Fixed
- Fix Go SDK cron inputs by @mnafees in [#2481](https://github.com/hatchet-dev/hatchet/pull/2481)
- Include payload partitions in olap partitions to drop by @mrkaye97 in [#2472](https://github.com/hatchet-dev/hatchet/pull/2472)


## [0.73.56] - 2025-10-30

### Changed
- Update managed compute regions by @mnafees in [#2470](https://github.com/hatchet-dev/hatchet/pull/2470)

### Fixed
- Read payloads from payload store for event API by @mrkaye97 in [#2471](https://github.com/hatchet-dev/hatchet/pull/2471)


## [0.73.55] - 2025-10-30

### Fixed
- Re-enable writes by @mrkaye97 in [#2469](https://github.com/hatchet-dev/hatchet/pull/2469)


## [0.73.54] - 2025-10-30

### Changed
- Run cleanup on more tables by @mnafees in [#2467](https://github.com/hatchet-dev/hatchet/pull/2467)

### Fixed
- Don't send expiry alert on internal proxy tokens by @abelanger5 in [#2468](https://github.com/hatchet-dev/hatchet/pull/2468)


## [0.73.53] - 2025-10-30

### Changed
- No need to check for partitions when updating them by @mnafees in [#2466](https://github.com/hatchet-dev/hatchet/pull/2466)


## [0.73.52] - 2025-10-30

### Changed
- [hotfix] Meaningful casing for engine liveness and readiness probes by @mnafees in [#2465](https://github.com/hatchet-dev/hatchet/pull/2465)


## [0.73.51] - 2025-10-30

### Changed
- Increase timeout and log more by @mnafees in [#2464](https://github.com/hatchet-dev/hatchet/pull/2464)


## [0.73.50] - 2025-10-30

### Changed
- Do not run cleanup on `v1_workflow_concurrency_slot` by @mnafees in [#2463](https://github.com/hatchet-dev/hatchet/pull/2463)


## [0.73.49] - 2025-10-30

### Added
- Add support for non-wal payload store logic to skip main db by @sidpremkumar in [#2445](https://github.com/hatchet-dev/hatchet/pull/2445)

### Changed
- Logs for liveness and readiness endpoints + PG conn stats by @mnafees in [#2460](https://github.com/hatchet-dev/hatchet/pull/2460)

### Fixed
- Reduce status update limits from 10k -> 1k by @abelanger5 in [#2462](https://github.com/hatchet-dev/hatchet/pull/2462)


## [0.73.47] - 2025-10-28

### Changed
- [hotfix] Fix running task stats without concurrency keys by @mnafees in [#2452](https://github.com/hatchet-dev/hatchet/pull/2452)


## [0.73.46] - 2025-10-28

### Changed
- New tenant task stats endpoint by @mnafees in [#2433](https://github.com/hatchet-dev/hatchet/pull/2433)
- Retry RMQ messages indefinitely with aggressive logging after 5 retries by @mnafees in [#2448](https://github.com/hatchet-dev/hatchet/pull/2448)
- Increase timeout to 30 seconds by @mnafees in [#2449](https://github.com/hatchet-dev/hatchet/pull/2449)

### Fixed
- Fix confusing error by @mnafees in [#2447](https://github.com/hatchet-dev/hatchet/pull/2447)


## [0.73.45] - 2025-10-23

### Added
- Add vars to tune concurrency poller by @mnafees in [#2428](https://github.com/hatchet-dev/hatchet/pull/2428)

### Changed
- Run cleanup job every minute by @mnafees in [#2440](https://github.com/hatchet-dev/hatchet/pull/2440)

### Fixed
- Payload performance by @abelanger5 in [#2441](https://github.com/hatchet-dev/hatchet/pull/2441)


## [0.73.44] - 2025-10-21

### Fixed
- Move err check to before len check by @abelanger5 in [#2437](https://github.com/hatchet-dev/hatchet/pull/2437)


## [0.73.43] - 2025-10-20

### Added
- OLAP Payloads by @mrkaye97 in [#2410](https://github.com/hatchet-dev/hatchet/pull/2410)


## [0.73.42] - 2025-10-17

### Fixed
- Fix race condition in child spawn by @mnafees in [#2429](https://github.com/hatchet-dev/hatchet/pull/2429)


## [0.73.41] - 2025-10-16

### Fixed
- Fix for Hatchet Lite to still properly fallback to using the `postgres` message queue kind by @mnafees in [#2392](https://github.com/hatchet-dev/hatchet/pull/2392)


## [0.73.40] - 2025-10-16

### Changed
- Cleanup job for old and invalid entries by @mnafees in [#2378](https://github.com/hatchet-dev/hatchet/pull/2378)
- [hotfix] Avoid throwing error logs from ratelimit MW for invalid API routes by @mnafees in [#2420](https://github.com/hatchet-dev/hatchet/pull/2420)

### Fixed
- Fix OTel span attribute naming convention by @mnafees in [#2409](https://github.com/hatchet-dev/hatchet/pull/2409)
- Swallow idempotency key error for scheduled runs by @mrkaye97 in [#2425](https://github.com/hatchet-dev/hatchet/pull/2425)


## [0.73.40-alpha.0] - 2025-10-15

### Fixed
- Payload fallback for child runs by @mrkaye97 in [#2421](https://github.com/hatchet-dev/hatchet/pull/2421)


## [0.73.38] - 2025-10-15

### Added
- Stateful polling intervals by @abelanger5 in [#2417](https://github.com/hatchet-dev/hatchet/pull/2417)
- Scheduled run detail view, bulk cancel / replay with pagination helper by @mrkaye97 in [#2416](https://github.com/hatchet-dev/hatchet/pull/2416)

### Changed
- Introduce vars to tune `ANALYZE` job gocron run intervals by @mnafees in [#2407](https://github.com/hatchet-dev/hatchet/pull/2407)
- Use `UTC` for all pgx connections and check for database TZ by @mnafees in [#2398](https://github.com/hatchet-dev/hatchet/pull/2398)


## [0.73.35] - 2025-10-08

### Added
- Gzip compression for large payloads, persistent OLAP writes by @mrkaye97 in [#2368](https://github.com/hatchet-dev/hatchet/pull/2368)
- Immediate Payload Offloads by @mrkaye97 in [#2375](https://github.com/hatchet-dev/hatchet/pull/2375)
- Pausable Crons by @mrkaye97 in [#2395](https://github.com/hatchet-dev/hatchet/pull/2395)

### Changed
- Properly case on output byte length by @mnafees in [#2394](https://github.com/hatchet-dev/hatchet/pull/2394)

### Fixed
- Optimize concurrency slot trigger method by @abelanger5 in [#2391](https://github.com/hatchet-dev/hatchet/pull/2391)


## [0.73.34] - 2025-10-03

### Changed
- Include `tenant_id` in OTel spans wherever possible by @mnafees in [#2382](https://github.com/hatchet-dev/hatchet/pull/2382)

### Fixed
- Payload fallbacks, WAL conflict handling, WAL eviction by @mrkaye97 in [#2372](https://github.com/hatchet-dev/hatchet/pull/2372)
- Run analyze every 3 hours by @mrkaye97 in [#2380](https://github.com/hatchet-dev/hatchet/pull/2380)


## [0.73.33] - 2025-10-02

### Added
- Add ApplyNamespace for BulkRunWorkflow by @icbd in [#2374](https://github.com/hatchet-dev/hatchet/pull/2374)


## [0.73.32] - 2025-09-30

### Changed
- Candidate Fix: WAL Write Dupes by @mrkaye97 in [#2369](https://github.com/hatchet-dev/hatchet/pull/2369)
- Event payload max height, popover positioning by @mrkaye97 in [#2367](https://github.com/hatchet-dev/hatchet/pull/2367)


## [0.73.31] - 2025-09-30

### Fixed
- Relax check constraint to allow null payloads by @mrkaye97 in [#2366](https://github.com/hatchet-dev/hatchet/pull/2366)


## [0.73.30] - 2025-09-30

### Added
- Max channels for rabbitmq by @abelanger5 in [#2365](https://github.com/hatchet-dev/hatchet/pull/2365)

### Changed
- Ignore tenants with deletedAt non null by @mnafees in [#2364](https://github.com/hatchet-dev/hatchet/pull/2364)


## [0.73.29] - 2025-09-29

### Changed
- Use member populator for tenant member API ops by @mnafees in [#2363](https://github.com/hatchet-dev/hatchet/pull/2363)

### Fixed
- Disable inner scroll by @mrkaye97 in [#2362](https://github.com/hatchet-dev/hatchet/pull/2362)


## [0.73.28] - 2025-09-29

### Changed
- Worker detail fixes by @mrkaye97 in [#2353](https://github.com/hatchet-dev/hatchet/pull/2353)

### Fixed
- Use separate connections for pub and sub by @abelanger5 in [#2358](https://github.com/hatchet-dev/hatchet/pull/2358)


## [0.73.26] - 2025-09-26

### Fixed
- Async start step run action event by @abelanger5 in [#2351](https://github.com/hatchet-dev/hatchet/pull/2351)


## [0.73.25] - 2025-09-26

### Changed
- FE Polish, VI: Make badges dynamically sized, use slate instead of fuchsia for queued, display ms on dates by @mrkaye97 in [#2352](https://github.com/hatchet-dev/hatchet/pull/2352)


## [0.73.24] - 2025-09-26

### Fixed
- Scope override by @mrkaye97 in [#2349](https://github.com/hatchet-dev/hatchet/pull/2349)


## [0.73.23] - 2025-09-26

### Fixed
- Rename metrics queries, always refetch queue metrics, change default refetch interval, configurable WAL poll limit by @mrkaye97 in [#2346](https://github.com/hatchet-dev/hatchet/pull/2346)


## [0.73.22] - 2025-09-25

### Changed
- Error log if we send >10mb message over the internal queue by @mrkaye97 in [#2345](https://github.com/hatchet-dev/hatchet/pull/2345)


## [0.73.21] - 2025-09-25

### Changed
- FE Polish V: Searchable workflows, additional metadata tab by @mrkaye97 in [#2342](https://github.com/hatchet-dev/hatchet/pull/2342)

### Fixed
- Improve DAG status updates by @abelanger5 in [#2343](https://github.com/hatchet-dev/hatchet/pull/2343)
- Show legacy data by @mrkaye97 in [#2344](https://github.com/hatchet-dev/hatchet/pull/2344)


## [0.73.20] - 2025-09-24

### Changed
- FE Polish, IV: Tooltips, task event scrolling by @mrkaye97 in [#2335](https://github.com/hatchet-dev/hatchet/pull/2335)
- FE Polish, III: More state management fixes, worker pages by @mrkaye97 in [#2332](https://github.com/hatchet-dev/hatchet/pull/2332)

### Fixed
- Event getter backwards compat by @mrkaye97 in [#2337](https://github.com/hatchet-dev/hatchet/pull/2337)
- Use `SplitN` instead of `Split` by @mrkaye97 in [#2336](https://github.com/hatchet-dev/hatchet/pull/2336)
- Stable ordering for flattened tasks + child tasks by @mrkaye97 in [#2334](https://github.com/hatchet-dev/hatchet/pull/2334)


## [0.73.18] - 2025-09-23

### Fixed
- Rogue effect hook, some more cleanup by @mrkaye97 in [#2329](https://github.com/hatchet-dev/hatchet/pull/2329)


## [0.73.17] - 2025-09-23

### Added
- Add error level logs if we fall back to the task input for monitoring by @mrkaye97 in [#2328](https://github.com/hatchet-dev/hatchet/pull/2328)


## [0.73.16] - 2025-09-23

### Fixed
- Frontend polish, Part II by @mrkaye97 in [#2327](https://github.com/hatchet-dev/hatchet/pull/2327)


## [0.73.15] - 2025-09-23

### Changed
- Update docs to use Go SDK v1 by @mnafees in [#2313](https://github.com/hatchet-dev/hatchet/pull/2313)


## [0.73.14] - 2025-09-22

### Added
- Show statuses of run filters with colors by @mrkaye97 in [#2325](https://github.com/hatchet-dev/hatchet/pull/2325)


## [0.73.13] - 2025-09-21

### Added
- Support dynamic rate limit durations by @abelanger5 in [#2320](https://github.com/hatchet-dev/hatchet/pull/2320)

### Changed
- Compact toolbar, refetch improvements, table improvements by @mrkaye97 in [#2292](https://github.com/hatchet-dev/hatchet/pull/2292)

### Fixed
- Skip locked on queue updates by @abelanger5 in [#2321](https://github.com/hatchet-dev/hatchet/pull/2321)


## [0.73.12] - 2025-09-19

### Fixed
- Payload WAL dupes by @mrkaye97 in [#2319](https://github.com/hatchet-dev/hatchet/pull/2319)


## [0.73.11] - 2025-09-19

### Fixed
- Update payload properly on replay by @mrkaye97 in [#2317](https://github.com/hatchet-dev/hatchet/pull/2317)


## [0.73.10] - 2025-09-18

### Fixed
- Payloads OLAP backwards compat by @mrkaye97 in [#2316](https://github.com/hatchet-dev/hatchet/pull/2316)


## [0.73.9] - 2025-09-18

### Fixed
- Event filtering edge case by @mrkaye97 in [#2311](https://github.com/hatchet-dev/hatchet/pull/2311)


## [0.73.8] - 2025-09-18

### Fixed
- Fix seed default tenant slug by @mnafees in [#2315](https://github.com/hatchet-dev/hatchet/pull/2315)


## [0.73.7] - 2025-09-17

### Fixed
- Empty state by @grutt in [#2310](https://github.com/hatchet-dev/hatchet/pull/2310)


## [0.73.6] - 2025-09-17

### Changed
- Properly fall back to tenant switcher by @mnafees in [#2307](https://github.com/hatchet-dev/hatchet/pull/2307)

### Fixed
- DAG details rendering in side panel, backwards compatible event list API by @mrkaye97 in [#2309](https://github.com/hatchet-dev/hatchet/pull/2309)


## [0.73.4] - 2025-09-16

### Changed
- Allow RabbitMQ to be used with Hatchet Lite by @mnafees in [#2128](https://github.com/hatchet-dev/hatchet/pull/2128)


## [0.73.3] - 2025-09-16

### Changed
- [hotfix] CLI arg to specify average duration per event threshold for loadtest to succeed by @mnafees in [#2300](https://github.com/hatchet-dev/hatchet/pull/2300)

### Fixed
- Fix `GetDetails` in `Runs` feature client of Go SDK v1 by @mnafees in [#2297](https://github.com/hatchet-dev/hatchet/pull/2297)
- WAL partition poll function type by @mrkaye97 in [#2301](https://github.com/hatchet-dev/hatchet/pull/2301)


## [0.73.2] - 2025-09-15

### Added
- Add organization by @mnafees in [#2299](https://github.com/hatchet-dev/hatchet/pull/2299)


## [0.73.1] - 2025-09-12

### Fixed
- Revert partition pruning by @mrkaye97 in [#2295](https://github.com/hatchet-dev/hatchet/pull/2295)


## [0.73.0] - 2025-09-12

### Added
- Add panic handler to Go SDK by @mnafees in [#2293](https://github.com/hatchet-dev/hatchet/pull/2293)
- Partition pruning for `ListTaskParentOutputs`, lookup index for `v1_payload_wal` by @mrkaye97 in [#2294](https://github.com/hatchet-dev/hatchet/pull/2294)
- Payload Store Repository by @mrkaye97 in [#2047](https://github.com/hatchet-dev/hatchet/pull/2047)

### Removed
- Remove `nginx` and use custom static fileservers by @mnafees in [#1928](https://github.com/hatchet-dev/hatchet/pull/1928)

### Fixed
- Scheduled runs race w/ idempotency key check by @mrkaye97 in [#2077](https://github.com/hatchet-dev/hatchet/pull/2077)


## [0.72.8] - 2025-09-11

### Added
- Filters UI, Events page refactor, Misc. other fixes by @mrkaye97 in [#2276](https://github.com/hatchet-dev/hatchet/pull/2276)
- Feat  improve auth error handling by @grutt in [#1893](https://github.com/hatchet-dev/hatchet/pull/1893)


## [0.72.4] - 2025-09-10

### Changed
- Do not show archived tenants anywhere by @mnafees in [#2280](https://github.com/hatchet-dev/hatchet/pull/2280)
- Docs-in-app, Part I by @mrkaye97 in [#2183](https://github.com/hatchet-dev/hatchet/pull/2183)


## [0.72.3] - 2025-09-09

### Changed
- Org UI feedback improvements by @mnafees in [#2275](https://github.com/hatchet-dev/hatchet/pull/2275)
- Error out instead of panic by @mnafees in [#2274](https://github.com/hatchet-dev/hatchet/pull/2274)


## [0.72.2] - 2025-09-09

### Added
- Worker slot Prom metrics by @mrkaye97 in [#2195](https://github.com/hatchet-dev/hatchet/pull/2195)

### Changed
- Go SDK v1 feature client changes by @mnafees in [#2160](https://github.com/hatchet-dev/hatchet/pull/2160)

### Fixed
- Fix custom auth casing by @mnafees in [#2268](https://github.com/hatchet-dev/hatchet/pull/2268)


## [0.72.1] - 2025-09-08

### Changed
- Expired invite status by @mnafees in [#2266](https://github.com/hatchet-dev/hatchet/pull/2266)

### Fixed
- Fix type names by @mnafees in [#2264](https://github.com/hatchet-dev/hatchet/pull/2264)


## [0.72.0] - 2025-09-05

### Changed
- Introduce UI for Organizations by @mnafees in [#2247](https://github.com/hatchet-dev/hatchet/pull/2247)
- Make sure to case on err properly by @mnafees in [#2248](https://github.com/hatchet-dev/hatchet/pull/2248)
- Properly handle 404s from populator middleware to avoid panics by @mnafees in [#2238](https://github.com/hatchet-dev/hatchet/pull/2238)

### Fixed
- Fixes for organization selector by @mnafees in [#2257](https://github.com/hatchet-dev/hatchet/pull/2257)


## [0.71.13] - 2025-09-02

### Changed
- Periodically run `ANALYZE` on `v1_task` and `v1_task_event` by @mnafees in [#2236](https://github.com/hatchet-dev/hatchet/pull/2236)
- Guard cleanAdditionalMetadata against JSON null; client: avoid null AdditionalMetadata in BulkPush; add regression test by @xcono in [#2191](https://github.com/hatchet-dev/hatchet/pull/2191)
- Inject into context by @mnafees in [#2213](https://github.com/hatchet-dev/hatchet/pull/2213)
- Onboarding key by @grutt in [#2212](https://github.com/hatchet-dev/hatchet/pull/2212)

### Fixed
- Rm annoying loki logs by @mrkaye97 in [#2224](https://github.com/hatchet-dev/hatchet/pull/2224)
- Remove rate limited items from in memory buffer by @abelanger5 in [#2207](https://github.com/hatchet-dev/hatchet/pull/2207)


## [0.71.10] - 2025-08-26

### Fixed
- Remove `custom auth` by @mrkaye97 in [#2203](https://github.com/hatchet-dev/hatchet/pull/2203)


## [0.71.9] - 2025-08-26

### Fixed
- Explicit ordering in ReleaseTasks and lock parent slots by @abelanger5 in [#2201](https://github.com/hatchet-dev/hatchet/pull/2201)
- Don't query database when flush is called concurrently by @abelanger5 in [#2202](https://github.com/hatchet-dev/hatchet/pull/2202)
- Confusing error message by @abelanger5 in [#2199](https://github.com/hatchet-dev/hatchet/pull/2199)


## [0.71.8] - 2025-08-25

### Fixed
- Child runs not rendering after one day, empty worker ids, additional meta filters not being applied to counts by @mrkaye97 in [#2196](https://github.com/hatchet-dev/hatchet/pull/2196)


## [0.71.8-alpha.0] - 2025-08-25

### Added
- Improved onboarding part 1 by @grutt in [#2186](https://github.com/hatchet-dev/hatchet/pull/2186)

### Fixed
- Match and cancel newest/in progress deadlocks by @abelanger5 in [#2190](https://github.com/hatchet-dev/hatchet/pull/2190)


## [0.71.7] - 2025-08-22

### Added
- Add visibility to stream send event by @abelanger5 in [#2174](https://github.com/hatchet-dev/hatchet/pull/2174)
- Analytics events by @grutt in [#2171](https://github.com/hatchet-dev/hatchet/pull/2171)

### Changed
- Re-enable refetch queue metrics, fix action button / dropdown state by @mrkaye97 in [#2182](https://github.com/hatchet-dev/hatchet/pull/2182)
- Limit frequency of updates to rate limits by @abelanger5 in [#2173](https://github.com/hatchet-dev/hatchet/pull/2173)


## [0.71.4] - 2025-08-20

### Added
- Run `ANALYZE` on a few tables once a day by @mrkaye97 in [#2163](https://github.com/hatchet-dev/hatchet/pull/2163)
- Add Linear to preconfigured webhooks by @mrkaye97 in [#2157](https://github.com/hatchet-dev/hatchet/pull/2157)

### Changed
- Introduce `customAuth` to the OpenAPI spec by @mnafees in [#2168](https://github.com/hatchet-dev/hatchet/pull/2168)

### Fixed
- Move rate limited queue items off the main queue by @abelanger5 in [#2155](https://github.com/hatchet-dev/hatchet/pull/2155)
- Populate DAG Metadata Sequentially by @mrkaye97 in [#2156](https://github.com/hatchet-dev/hatchet/pull/2156)
- Auto-generate docs snippets and examples by @mrkaye97 in [#2139](https://github.com/hatchet-dev/hatchet/pull/2139)
- Deadlocking on DAG concurrency by @mrkaye97 in [#2111](https://github.com/hatchet-dev/hatchet/pull/2111)


## [0.70.7] - 2025-08-14

### Added
- Webhook fixes / improvements by @mrkaye97 in [#2131](https://github.com/hatchet-dev/hatchet/pull/2131)

### Changed
- Runs list state management + bug fixes part I by @mrkaye97 in [#2114](https://github.com/hatchet-dev/hatchet/pull/2114)
- [hotfix] Better messaging around tenant prometheus metrics empty state by @mnafees in [#2124](https://github.com/hatchet-dev/hatchet/pull/2124)
- Workflow combobox search functionality by @mnafees in [#2118](https://github.com/hatchet-dev/hatchet/pull/2118)

### Fixed
- Add back sync docs script by @mrkaye97 in [#2123](https://github.com/hatchet-dev/hatchet/pull/2123)


## [0.70.6] - 2025-08-12

### Added
- Add k8s pod info to traces by @mnafees in [#2109](https://github.com/hatchet-dev/hatchet/pull/2109)

### Changed
- [Python] Feat: Dependency Injection, Improved error handling by @mrkaye97 in [#2067](https://github.com/hatchet-dev/hatchet/pull/2067)
- [HAT-432] Enforce task priorities to be between 1 and 3 by @mnafees in [#2110](https://github.com/hatchet-dev/hatchet/pull/2110)

### Fixed
- Optimize DAG timing query for Prom by @mrkaye97 in [#2102](https://github.com/hatchet-dev/hatchet/pull/2102)
- Waterfall panic + query simplification by @mrkaye97 in [#2116](https://github.com/hatchet-dev/hatchet/pull/2116)
- Don't wait for grpc stream send on rabbitmq loop by @abelanger5 in [#2115](https://github.com/hatchet-dev/hatchet/pull/2115)


## [0.70.5] - 2025-08-07

### Changed
- Fail task tracing by @mrkaye97 in [#2101](https://github.com/hatchet-dev/hatchet/pull/2101)


## [0.70.4] - 2025-08-06

### Added
- Add contextual data for trigger via events by @mnafees in [#2092](https://github.com/hatchet-dev/hatchet/pull/2092)

### Fixed
- Call `PopulateTaskRunData` sequentially by @mrkaye97 in [#2097](https://github.com/hatchet-dev/hatchet/pull/2097)


## [0.70.3] - 2025-08-06

### Added
- Add telemetry to task status repo methods by @mnafees in [#2091](https://github.com/hatchet-dev/hatchet/pull/2091)

### Fixed
- Improve performance of `UpdateTasksToAssigned` by @mrkaye97 in [#2094](https://github.com/hatchet-dev/hatchet/pull/2094)


## [0.70.2] - 2025-08-06

### Added
- Add telemetry around task statuses in controller by @mnafees in [#2090](https://github.com/hatchet-dev/hatchet/pull/2090)

### Fixed
- ProcessTaskTimeouts limit and timeout by @grutt in [#2087](https://github.com/hatchet-dev/hatchet/pull/2087)
- Webhook copy improvements by @mrkaye97 in [#2081](https://github.com/hatchet-dev/hatchet/pull/2081)
