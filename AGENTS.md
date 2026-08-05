# AGENTS

## Testing features locally

- To test a feature against a real running Hatchet instance without docker
  compose, follow `.claude/skills/embedded-hatchet/SKILL.md`: it runs the
  full engine + API in-process from this checkout, backed by a throwaway
  embedded Postgres, and can register workers and run workflows.

## CI

- Any CI surface that boots a Hatchet server instance (engine, API, `hatchet-lite`, docker-compose, or helm) must set `SERVER_SECURITY_CHECK_ENABLED=false`. The check defaults to enabled and phones home to `security.hatchet.run`; CI must never do that. `go test`-based boots are already covered by the test harness; every other boot site sets the var explicitly.

## Code Comments

- Comments should explain non-obvious intent, invariants, or constraints in the current code. Do not mention the old implementation/state (for example, "preserve the behavior of the original query"); state the current rule directly.

## Docs MDX

- In MDX JSX component bodies, such as `<Callout>`, avoid Markdown link syntax (`[text](href)`). Prettier can wrap the label across lines and break MDX parsing. Use an explicit JSX link instead:

```mdx
<Callout type="info">
  See the{" "}
  <a href="/v1/retry-policies#go-sdk-client-retry-behavior">
    Go SDK client retry behavior section
  </a>
</Callout>
```
