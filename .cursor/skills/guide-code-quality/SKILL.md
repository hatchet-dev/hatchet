---
name: guide-code-quality
description: Code quality standards for Hatchet SDK guide/cookbook examples across Python, TypeScript, Go, and Ruby. Use when writing, editing, or reviewing files in sdks/guides/, or when creating new cookbook examples.
---

# Guide Code Quality Standards

Guide code in `sdks/guides/{lang}/` is the source of truth for documentation snippets. It must be **runnable, type-safe, and free of suppressions**.

## Snippet Markers

Wrap every extractable section with markers. Everything between markers becomes a doc snippet.

| Language | Open | Close |
|----------|------|-------|
| Python / Ruby | `# > Step Title` | `# !!` |
| TypeScript / Go | `// > Step Title` | `// !!` |

Titles become snake_case keys: `# > Step 04 Rate Limited Scrape` → `step_04_rate_limited_scrape`. **All helper functions, models, and types used by a snippet must be inside the markers** so the generator captures them.

After editing any guide file, regenerate: `cd frontend/snippets && python3 generate.py`

## Zero Suppressions Policy

No `type: ignore`, `@ts-ignore`, `nolint`, `disable_error_code`, or equivalent. Fix the code instead.

- If types mismatch at a third-party SDK boundary, convert explicitly (e.g. role-narrowing helpers, `.model_dump()`)
- If an import pattern triggers errors, restructure the code (e.g. `__init__.py` + relative imports instead of `try/except ImportError`)

## Python

### Type Checking

mypy config in `pyproject.toml` — no `disable_error_code`:

```toml
[tool.mypy]
disallow_untyped_defs = true
disallow_incomplete_defs = true
warn_return_any = true
no_implicit_optional = true
strict_equality = true
warn_unused_ignores = true
```

### Input/Output Models

Use Pydantic `BaseModel` with `input_validator` for all task inputs. Never use bare `dict` as a task parameter type.

```python
class UrlInput(BaseModel):
    url: str

scrape_wf = hatchet.workflow(name="ScrapeUrl", input_validator=UrlInput)

@scrape_wf.task()
async def scrape_url(input: UrlInput, ctx: Context) -> dict:
    return mock_scrape(input.url)
```

Standalone tasks use `input_validator` on the decorator:

```python
@hatchet.durable_task(name="ClassifyMessage", input_validator=MessageInput)
async def classify_message(input: MessageInput, ctx: DurableContext) -> dict:
    ...
```

### Service Interfaces

Use Pydantic models for service method signatures — not bare `dict` or `Any`:

```python
class ChatMessage(BaseModel):
    role: Literal["user", "assistant", "system", "tool"]
    content: str

class CompletionResult(BaseModel):
    content: str
    tool_calls: list[ToolCallResult]
    done: bool

class LLMService(ABC):
    @abstractmethod
    def complete(self, messages: list[ChatMessage]) -> CompletionResult: ...
```

### Package Structure

Every guide subdirectory must have `__init__.py`. Use relative imports only — no `try/except ImportError` fallback pattern:

```python
from .mock_scraper import mock_scrape  # correct
```

### Third-Party SDK Boundaries

When our Pydantic models don't match SDK TypedDicts, convert with role narrowing:

```python
from openai.types.chat import ChatCompletionMessageParam, ChatCompletionUserMessageParam, ...

def _to_openai_message(m: ChatMessage) -> ChatCompletionMessageParam:
    if m.role == "user":
        return ChatCompletionUserMessageParam(role="user", content=m.content)
    if m.role == "assistant":
        return ChatCompletionAssistantMessageParam(role="assistant", content=m.content)
    ...
```

Keep these converters **inside** snippet markers.

## TypeScript

- `tsconfig.json` must use `"strict": true`
- Use Zod schemas for input validation where the SDK supports it
- All functions must have explicit return types
- Use the SDK's typed interfaces, not `any`

## Go

- All exported functions must have doc comments
- Use typed structs for task input/output — not `map[string]interface{}`  where avoidable
- Handle all errors explicitly (`if err != nil`)
- Use `context.Context` properly through the call chain

## General (All Languages)

### Guide Directory Layout

```
guide-name/
├── __init__.py          # Python only
├── worker.{ext}         # Main worker with task definitions
├── trigger.{ext}        # Optional: client-side trigger example
├── mock_{service}.{ext} # Mock services for local dev
└── {service}.{ext}      # Service interfaces (ABC/interface/trait)
```

### Code Style

- No narrating comments (`// create the client`, `# import the module`)
- Docstrings only on public functions that need them
- Service pattern: abstract interface + mock implementation + factory function (`get_*_service()`)
- Mock services return deterministic data — no randomness
- All `main()` / entrypoint functions must have return type annotations
