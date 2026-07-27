# hatchet-oss database model

Generate a standalone, searchable HTML view of the `hatchet-oss` database
model:

```sh
task generate-hatchet-oss-model
```

The output is written to `generated/hatchet-oss.html`. Open that file in any
browser; it has no runtime dependencies or network requests. The generated
Orbit explorer focuses on one table at a time and supports relationship-depth
controls, pan and zoom, full table details, navigable joins, focus-history
back/forward controls, search, and persisted light/dark themes.

Each invocation discovers the canonical OSS schema snapshots from the `schema`
list in `pkg/repository/sqlcv1/sqlc.yaml` and parses their current contents. A
new snapshot under `sql/schema` is included when it is added to that list.
Schema edits are therefore reflected the next time the task runs; the output is
not automatically regenerated on every edit or by another generation task.
The generator serializes the discovered model into
`hack/db/hatchet-oss-template.html`, producing one self-contained HTML file.
Relationship traversal always follows every table, including high-degree hubs,
up to the selected depth. The generator does not connect to a database.

To choose a different output path:

```sh
task generate-hatchet-oss-model -- --output /tmp/hatchet-oss.html
```
