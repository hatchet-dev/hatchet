// Generated from /Users/gabrielruttner/dev/hatchet/examples/python/simple/worker.py
export const content = "# ❓ Simple\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n@hatchet.task(name=\"SimpleWorkflow\")\ndef step1(input: EmptyModel, ctx: Context) -> None:\n    print(\"executed step1\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"test-worker\", slots=1, workflows=[step1])\n    worker.start()\n\n\n# ‼️\n\nif __name__ == \"__main__\":\n    main()\n";
export const language = "py";
