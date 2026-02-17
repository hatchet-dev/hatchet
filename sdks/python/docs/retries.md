1️⃣ REST Retry Behavior

Retries:

5xx → ServiceException

404 → NotFoundException (current behavior)

Does NOT retry:

Transport errors (unless TenacityConfig enabled)

Mutating verbs by default

2️⃣ gRPC Retry Behavior

Retries most transient codes

Excludes clearly permanent status codes

Driven by Tenacity retry predicate

3️⃣ Transport Errors

Timeout

Connection errors

TLS/protocol errors

Configurable via TenacityConfig

4️⃣ Idempotency Note

Clear warning:

Retrying mutating operations may duplicate side effects. Ensure idempotency guarantees on backend before enabling aggressive retries.
