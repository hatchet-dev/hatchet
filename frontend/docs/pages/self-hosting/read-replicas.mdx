# Read Replica Support

For high-throughput production deployments, Hatchet supports database read replicas to distribute database load and improve read performance. This feature allows you to direct read queries to a separate database instance while continuing to send write operations to the primary database. **This can significantly improve performance in read-heavy workloads without requiring application changes.**

You can enable read replica support by setting the following environment variables:

```bash
READ_REPLICA_ENABLED=true
READ_REPLICA_DATABASE_URL='postgresql://hatchet:hatchet@127.0.0.1:5432/hatchet'
READ_REPLICA_MAX_CONNS=200
READ_REPLICA_MIN_CONNS=50
```

## Configuration Options

- `READ_REPLICA_ENABLED`: Set to `true` to enable read replica support
- `READ_REPLICA_DATABASE_URL`: Connection string for the read replica database
- `READ_REPLICA_MAX_CONNS`: Maximum number of connections in the read replica connection pool
- `READ_REPLICA_MIN_CONNS`: Minimum number of connections to maintain in the read replica connection pool

## Limitations

- Replication lag may result in slightly stale or missing data being returned from read operations
- The read replica is only utilized by analytical queries (to load workflow runs, task runs, and metrics in the UI)
