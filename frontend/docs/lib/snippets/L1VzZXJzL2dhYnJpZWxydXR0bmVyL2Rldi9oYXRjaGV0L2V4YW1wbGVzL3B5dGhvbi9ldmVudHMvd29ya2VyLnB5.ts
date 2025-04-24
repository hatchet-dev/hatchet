// Generated from /Users/gabrielruttner/dev/hatchet/examples/python/events/worker.py
export const content = "from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n# ❓ Event trigger\nevent_workflow = hatchet.workflow(name=\"EventWorkflow\", on_events=[\"user:create\"])\n# ‼️\n\n\n@event_workflow.task()\ndef task(input: EmptyModel, ctx: Context) -> None:\n    print(\"event received\")\n";
export const language = "py";
