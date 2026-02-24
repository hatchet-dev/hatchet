import re
from collections.abc import Callable
from copy import deepcopy
from pathlib import Path


def prepend_import(content: str, import_statement: str) -> str:
    if import_statement in content:
        return content

    future_import_pattern = r"^from __future__ import [^\n]+\n"
    future_imports = re.findall(future_import_pattern, content, re.MULTILINE)
    content = re.sub(future_import_pattern, "", content, flags=re.MULTILINE)

    match = re.search(r"^import\s+|^from\s+", content, re.MULTILINE)
    insert_position = match.start() if match else 0

    future_block = "".join(future_imports)
    return (
        content[:insert_position]
        + future_block
        + import_statement
        + "\n"
        + content[insert_position:]
    )


def apply_patch(content: str, pattern: str, replacement: str) -> str:
    return re.sub(pattern, replacement, content)


def atomically_patch_file(
    file_path: str, patch_funcs: list[Callable[[str], str]]
) -> None:
    path = Path(file_path)
    original = path.read_text()

    modified = deepcopy(original)

    try:
        for func in patch_funcs:
            modified = func(modified)
    except Exception as e:
        print(f"Error patching {file_path}: {e}")
        return

    if modified != original:
        path.write_text(modified)
        print(f"Patched {file_path}")
    else:
        print(f"No changes made to {file_path}")


def patch_contract_import_paths(content: str) -> str:
    return apply_patch(content, r"\bfrom v1\b", "from hatchet_sdk.contracts.v1")


def patch_grpc_dispatcher_import(content: str) -> str:
    return apply_patch(
        content,
        r"\bimport dispatcher_pb2 as dispatcher__pb2\b",
        "from hatchet_sdk.contracts import dispatcher_pb2 as dispatcher__pb2",
    )


def patch_grpc_events_import(content: str) -> str:
    return apply_patch(
        content,
        r"\bimport events_pb2 as events__pb2\b",
        "from hatchet_sdk.contracts import events_pb2 as events__pb2",
    )


def patch_grpc_workflows_import(content: str) -> str:
    return apply_patch(
        content,
        r"\bimport workflows_pb2 as workflows__pb2\b",
        "from hatchet_sdk.contracts import workflows_pb2 as workflows__pb2",
    )


def patch_grpc_init_signature(content: str) -> str:
    return apply_patch(
        content,
        r"def __init__\(self, channel\):",
        "def __init__(self, channel: grpc.Channel | grpc.aio.Channel) -> None:",
    )


def patch_rest_429_exception(content: str) -> str:
    """Add TooManyRequestsException with 429 mapping to generated exceptions.py."""
    # Insert class definition once, before RestTransportError.
    if "class TooManyRequestsException" not in content:
        new_class = (
            "class TooManyRequestsException(ApiException):\n"
            '    """Exception for HTTP 429 Too Many Requests."""\n'
            "    pass\n"
        )

        pattern = r"(?m)^class RestTransportError\b"
        content, n = re.subn(
            pattern,
            new_class + "\n\nclass RestTransportError",
            content,
            count=1,
        )
        if n != 1:
            raise ValueError(
                "patch_rest_429_exception: expected 'class RestTransportError' anchor not found"
            )

    # insert mapping once, before the 5xx check.
    if "http_resp.status == 429" not in content:
        pattern = r"(?m)^(?P<indent>[ \t]*)if 500 <= http_resp\.status <= 599:"

        def _insert_429(m: re.Match[str]) -> str:
            indent = m.group("indent")
            return (
                f"{indent}if http_resp.status == 429:\n"
                f"{indent}    raise TooManyRequestsException(http_resp=http_resp, body=body, data=data)\n"
                f"\n"
                f"{indent}if 500 <= http_resp.status <= 599:"
            )

        content, n = re.subn(pattern, _insert_429, content, count=1)
        if n != 1:
            raise ValueError(
                "patch_rest_429_exception: expected 5xx mapping anchor not found"
            )

    return content


def patch_rest_transport_exceptions(content: str) -> str:
    """Insert typed REST transport exception classes into exceptions.py.

    Adds exception classes above render_path function, idempotently.
    """
    # Check if already patched
    if "class RestTransportError" in content:
        return content

    new_exceptions = '''\

class RestTransportError(ApiException):
    """Base exception for REST transport-level errors (network, timeout, TLS)."""

    pass


class RestTimeoutError(RestTransportError):
    """Raised when a REST request times out (connect or read timeout)."""

    pass


class RestConnectionError(RestTransportError):
    """Raised when a REST request fails to establish a connection."""

    pass


class RestTLSError(RestTransportError):
    """Raised when a REST request fails due to SSL/TLS errors."""

    pass


class RestProtocolError(RestTransportError):
    """Raised when a REST request fails due to protocol-level errors."""

    pass


'''

    # Insert before render_path function (match any arguments)
    pattern = r"(\ndef render_path\([^)]*\):)"
    replacement = new_exceptions + r"\1"

    return re.sub(pattern, replacement, content)


def patch_rest_imports(content: str) -> str:
    """Update rest.py imports to include typed transport exceptions.

    Handles both single-line and parenthesized import formats. Idempotent.
    """
    # The exceptions we need to ensure are imported
    required_exceptions = [
        "RestConnectionError",
        "RestProtocolError",
        "RestTimeoutError",
        "RestTLSError",
    ]

    # Idempotency check: if RestTLSError is already imported from this module, do nothing.
    if re.search(
        r"(?m)^from\s+hatchet_sdk\.clients\.rest\.exceptions\s+import[^\n]*\bRestTLSError\b",
        content,
    ):
        return content

    # Parenthesized import block includes RestTLSError
    if re.search(
        r"^from\s+hatchet_sdk\.clients\.rest\.exceptions\s+import\s*\(\s*.*?\bRestTLSError\b.*?\)\s*$",
        content,
        flags=re.MULTILINE | re.DOTALL,
    ):
        return content

    # The target import statement we want with trailing newline to preserve spacing
    new_import = (
        "from hatchet_sdk.clients.rest.exceptions import (\n"
        "    ApiException,\n"
        "    ApiValueError,\n"
        "    RestConnectionError,\n"
        "    RestProtocolError,\n"
        "    RestTimeoutError,\n"
        "    RestTLSError,\n"
        ")\n"
    )

    # Single line import
    # Matches: from hatchet_sdk.clients.rest.exceptions import ApiException, ApiValueError
    single_line_pattern = (
        r"^from\s+hatchet_sdk\.clients\.rest\.exceptions\s+import\s+"
        r"ApiException\s*,\s*ApiValueError\s*$"
    )

    modified = re.sub(single_line_pattern, new_import, content, flags=re.MULTILINE)
    if modified != content:
        return modified

    # More flexible parenthesized import which matches any order, with or without trailing comma
    # This handles cases where ApiException and ApiValueError might be in different orders
    flexible_paren_pattern = (
        r"^from\s+hatchet_sdk\.clients\.rest\.exceptions\s+import\s*\("
        r"[^)]*?"  # Non-greedy match of contents (only ApiException/ApiValueError expected)
        r"\)"
    )

    # Only apply if the block contains just ApiException and/or ApiValueError (no Rest* yet)
    match = re.search(flexible_paren_pattern, content, flags=re.MULTILINE | re.DOTALL)
    if match:
        block = match.group(0)
        # Verify it only has ApiException/ApiValueError, not our new exceptions
        if not any(exc in block for exc in required_exceptions):
            if "ApiException" in block or "ApiValueError" in block:
                modified = (
                    content[: match.start()] + new_import + content[match.end() :]
                )
                return modified

    return content


def patch_rest_error_diagnostics(content: str) -> str:
    """Patch rest.py exception handlers to use typed exceptions.

    Replaces the generic ApiException handler with typed exception handlers.
    Handler ordering is critical: NewConnectionError must be caught before
    ConnectTimeoutError because it inherits from ConnectTimeoutError in urllib3.
    """
    # This pattern matches either the original SSLError only handler or
    # the previously patched multi-exception handler raising ApiException
    pattern_original = (
        r"(?ms)^([ \t]*)except urllib3\.exceptions\.SSLError as e:\s*\n"
        r"^\1[ \t]*msg = \"\\n\"\.join\(\[type\(e\)\.__name__, str\(e\)\]\)\s*\n"
        r"^\1[ \t]*raise ApiException\(status=0, reason=msg\)\s*\n"
    )

    pattern_expanded = (
        r"(?ms)^([ \t]*)except \(\s*\n"
        r"^\1[ \t]*urllib3\.exceptions\.SSLError,\s*\n"
        r"^\1[ \t]*urllib3\.exceptions\.ConnectTimeoutError,\s*\n"
        r"^\1[ \t]*urllib3\.exceptions\.ReadTimeoutError,\s*\n"
        r"^\1[ \t]*urllib3\.exceptions\.MaxRetryError,\s*\n"
        r"^\1[ \t]*urllib3\.exceptions\.NewConnectionError,\s*\n"
        r"^\1[ \t]*urllib3\.exceptions\.ProtocolError,\s*\n"
        r"^\1\) as e:\s*\n"
        r'^\1[ \t]*msg = "\\n"\.join\(\s*\n'
        r"^\1[ \t]*\[\s*\n"
        r"^\1[ \t]*type\(e\)\.__name__,\s*\n"
        r"^\1[ \t]*str\(e\),\s*\n"
        r'^\1[ \t]*f"method=\{method\}",\s*\n'
        r'^\1[ \t]*f"url=\{url\}",\s*\n'
        r'^\1[ \t]*f"timeout=\{_request_timeout\}",\s*\n'
        r"^\1[ \t]*\]\s*\n"
        r"^\1[ \t]*\)\s*\n"
        r"^\1[ \t]*raise ApiException\(status=0, reason=msg\)\s*\n"
    )

    # Check if already using typed exceptions
    if "raise RestTLSError" in content:
        return content

    # Build typed replacement with proper handler ordering
    # NewConnectionError inherits from ConnectTimeoutError, so must be caught first
    replacement = (
        r"\1except urllib3.exceptions.SSLError as e:\n"
        r'\1    msg = "\\n".join(\n'
        r"\1        [\n"
        r"\1            type(e).__name__,\n"
        r"\1            str(e),\n"
        r'\1            f"method={method}",\n'
        r'\1            f"url={url}",\n'
        r'\1            f"timeout={_request_timeout}",\n'
        r"\1        ]\n"
        r"\1    )\n"
        r"\1    raise RestTLSError(status=0, reason=msg) from e\n"
        r"\1except (\n"
        r"\1    urllib3.exceptions.MaxRetryError,\n"
        r"\1    urllib3.exceptions.NewConnectionError,\n"
        r"\1) as e:\n"
        r"\1    # NewConnectionError inherits from ConnectTimeoutError, so must be caught first\n"
        r'\1    msg = "\\n".join(\n'
        r"\1        [\n"
        r"\1            type(e).__name__,\n"
        r"\1            str(e),\n"
        r'\1            f"method={method}",\n'
        r'\1            f"url={url}",\n'
        r'\1            f"timeout={_request_timeout}",\n'
        r"\1        ]\n"
        r"\1    )\n"
        r"\1    raise RestConnectionError(status=0, reason=msg) from e\n"
        r"\1except (\n"
        r"\1    urllib3.exceptions.ConnectTimeoutError,\n"
        r"\1    urllib3.exceptions.ReadTimeoutError,\n"
        r"\1) as e:\n"
        r'\1    msg = "\\n".join(\n'
        r"\1        [\n"
        r"\1            type(e).__name__,\n"
        r"\1            str(e),\n"
        r'\1            f"method={method}",\n'
        r'\1            f"url={url}",\n'
        r'\1            f"timeout={_request_timeout}",\n'
        r"\1        ]\n"
        r"\1    )\n"
        r"\1    raise RestTimeoutError(status=0, reason=msg) from e\n"
        r"\1except urllib3.exceptions.ProtocolError as e:\n"
        r'\1    msg = "\\n".join(\n'
        r"\1        [\n"
        r"\1            type(e).__name__,\n"
        r"\1            str(e),\n"
        r'\1            f"method={method}",\n'
        r'\1            f"url={url}",\n'
        r'\1            f"timeout={_request_timeout}",\n'
        r"\1        ]\n"
        r"\1    )\n"
        r"\1    raise RestProtocolError(status=0, reason=msg) from e\n"
    )

    # Try expanded pattern first. Relevant if previously patched with ApiException
    modified = re.sub(pattern_expanded, replacement, content)
    if modified != content:
        return modified

    # Otherwise try original pattern
    return re.sub(pattern_original, replacement, content)


def apply_patches_to_matching_files(
    root: str, glob: str, patch_funcs: list[Callable[[str], str]]
) -> None:
    for file_path in Path(root).rglob(glob):
        atomically_patch_file(str(file_path), patch_funcs)


def patch_api_client_datetime_format_on_post(content: str) -> str:
    content = prepend_import(content, "from hatchet_sdk.logger import logger")
    pattern = r"([ \t]*)elif isinstance\(obj, \(datetime\.datetime, datetime\.date\)\):\s*\n\1[ \t]*return obj\.isoformat\(\)"

    replacement = (
        r"\1## IMPORTANT: Checking `datetime` must come before `date` since `datetime` is a subclass of `date`\n"
        r"\1elif isinstance(obj, datetime.datetime):\n"
        r"\1    if not obj.tzinfo:\n"
        r"\1        current_tz = (datetime.datetime.now(datetime.timezone(datetime.timedelta(0))).astimezone().tzinfo or datetime.timezone.utc)\n"
        r'\1        logger.warning(f"timezone-naive datetime found. assuming {current_tz}.")\n'
        r"\1        obj = obj.replace(tzinfo=current_tz)\n\n"
        r"\1    return obj.isoformat()\n"
        r"\1elif isinstance(obj, datetime.date):\n"
        r"\1    return obj.isoformat()"
    )
    return apply_patch(content, pattern, replacement)


def patch_workflow_run_metrics_counts_return_type(content: str) -> str:
    content = prepend_import(
        content,
        "from hatchet_sdk.clients.rest.models.workflow_runs_metrics_counts import WorkflowRunsMetricsCounts",
    )
    pattern = r"([ \t]*)counts: Optional\[Dict\[str, Any\]\] = None"
    replacement = r"\1counts: Optional[WorkflowRunsMetricsCounts] = None"
    return apply_patch(content, pattern, replacement)


if __name__ == "__main__":
    atomically_patch_file(
        "hatchet_sdk/clients/rest/api_client.py",
        [patch_api_client_datetime_format_on_post],
    )
    atomically_patch_file(
        "hatchet_sdk/clients/rest/models/workflow_runs_metrics.py",
        [patch_workflow_run_metrics_counts_return_type],
    )

    atomically_patch_file(
        "hatchet_sdk/clients/rest/exceptions.py",
        [patch_rest_transport_exceptions, patch_rest_429_exception],
    )
    atomically_patch_file(
        "hatchet_sdk/clients/rest/rest.py",
        [patch_rest_imports, patch_rest_error_diagnostics],
    )

    grpc_patches: list[Callable[[str], str]] = [
        patch_contract_import_paths,
        patch_grpc_dispatcher_import,
        patch_grpc_events_import,
        patch_grpc_workflows_import,
        patch_grpc_init_signature,
    ]

    pb2_patches: list[Callable[[str], str]] = [
        patch_contract_import_paths,
    ]

    apply_patches_to_matching_files("hatchet_sdk/contracts", "*_grpc.py", grpc_patches)
    apply_patches_to_matching_files("hatchet_sdk/contracts", "*_pb2.py", pb2_patches)
    apply_patches_to_matching_files("hatchet_sdk/contracts", "*_pb2.pyi", pb2_patches)
