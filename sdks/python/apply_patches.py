import re
from copy import deepcopy
from pathlib import Path
from typing import Callable


def prepend_import(content: str, import_statement: str) -> str:
    if import_statement in content:
        return content

    match = re.search(r"^import\s+|^from\s+", content, re.MULTILINE)
    insert_position = match.start() if match else 0

    return (
        content[:insert_position] + import_statement + "\n" + content[insert_position:]
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


def apply_patches_to_matching_files(
    root: str, glob: str, patch_funcs: list[Callable[[str], str]]
) -> None:
    for file_path in Path(root).rglob(glob):
        atomically_patch_file(str(file_path), patch_funcs)


def patch_api_client_datetime_format_on_post(content: str) -> str:
    content = prepend_import(content, "from hatchet_sdk.logger import logger")
    pattern = r"([ \t]*)elif isinstance\(obj, \(datetime\.datetime, datetime\.date\)\):\s*\n\1[ \t]*return obj\.isoformat\(\)"

    replacement = (
        r"\1elif isinstance(obj, (datetime.datetime, datetime.date)):\n"
        r"\1    if not obj.tzinfo:\n"
        r"\1        current_tz = (datetime.datetime.now(datetime.timezone(datetime.timedelta(0))).astimezone().tzinfo or datetime.timezone.utc)\n"
        r'\1        logger.warning(f"timezone-naive datetime found. assuming {current_tz}.")\n'
        r"\1        obj = obj.replace(tzinfo=current_tz)\n\n"
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
