# Trace Sampling

For a very high-volume setup, it may be desirable to sample results for the dashboard for the purpose of limiting the amount of data stored in the Hatchet database. **This does not impact the behavior of the Hatchet engine and all tasks will still be processed.** This can be done by setting the following environment variables:

```bash
SERVER_SAMPLING_ENABLED=t
SERVER_SAMPLING_RATE=0.1 # only 10% of results will be sampled
```

Sampling is done at the workflow run level, so all tasks within the same workflow will be sampled, along with all of their events. Sampling has the following limitations:

- Parent tasks which spawn child tasks are not guaranteed to be sampled, even if their children are. This means that the child task may be shown in the dashboard without a corresponding parent task, and vice versa.
- There is no way to configure sampling to ensure that failure events are sampled.
- Only tasks which are sampled can be cancelled or replayed via the REST APIs: do not use this feature if dependent on programmatic cancellations and replays.
