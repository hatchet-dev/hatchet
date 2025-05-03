import re


def prepend_import(file_path: str, import_statement: str) -> None:
    with open(file_path, "r") as f:
        content = f.read()

    if import_statement in content:
        print(f"{import_statement} already exists in {file_path}")
        return

    match = re.search(r"^import\s+|^from\s+", content, re.MULTILINE)
    if match:
        insert_position = match.start()
    else:
        insert_position = 0

    new_content = (
        content[:insert_position] + import_statement + "\n" + content[insert_position:]
    )

    with open(file_path, "w") as f:
        f.write(new_content)


def apply_patch(file_path: str, pattern: str, replacement: str) -> None:
    with open(file_path, "r") as f:
        content = f.read()

    new_content = re.sub(pattern, replacement, content)

    if new_content == content:
        print(f"No changes made to {file_path}")
        return

    with open(file_path, "w") as f:
        f.write(new_content)

    print(f"Successfully updated {file_path}")


def patch_api_client_datetime_format_on_post() -> None:
    file_path = "hatchet_sdk/clients/rest/api_client.py"
    pattern = r"([ \t]*)elif isinstance\(obj, \(datetime\.datetime, datetime\.date\)\):\s*\n\1[ \t]*return obj\.isoformat\(\)"
    replacement = r"\1elif isinstance(obj, (datetime.datetime, datetime.date)):\n\1    if not obj.tzinfo:\n\1        logger.warning('timezone-naive datetime found. assuming UTC.')\n\1        obj = obj.replace(tzinfo=datetime.timezone.utc)\n\n\1    return obj.isoformat()"

    prepend_import(file_path, "from hatchet_sdk.logger import logger")
    apply_patch(file_path, pattern, replacement)


def patch_workflow_run_metrics_counts_return_type() -> None:
    file_path = "hatchet_sdk/clients/rest/models/workflow_runs_metrics.py"
    pattern = r"([ \t]*)counts: Optional\[Dict\[str, Any\]\] = None"
    replacement = r"\1counts: Optional[WorkflowRunsMetricsCounts] = None"

    prepend_import(
        file_path,
        "from hatchet_sdk.clients.rest.models.workflow_runs_metrics_counts import WorkflowRunsMetricsCounts",
    )
    apply_patch(file_path, pattern, replacement)


if __name__ == "__main__":
    patch_api_client_datetime_format_on_post()
    patch_workflow_run_metrics_counts_return_type()
