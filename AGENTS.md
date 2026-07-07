# AGENTS

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
