---
applyTo: "**"
---

# Comment Style

Comments must explain **why** or **how** — not **what**. If a comment only restates
the function name or describes what the next line of code does, delete it.

## Remove

- Section banners that label blocks of code by name or language
- Doc comments that paraphrase the function or field name
- Inline narration of the next line of code

## Keep

- Non-obvious design decisions and the reason behind them
- Constraints with their cause: why a limit exists, why a fallback is needed
- "Why something is absent": why a guard is missing, why a default is chosen
- Algorithmic details that aren't self-evident from the code
