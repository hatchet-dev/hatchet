import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "# > Simple\n\nfrom typing import Annotated\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Depends, DurableContext, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n# > Declare dependencies\nasync def async_dep() -> str:\n    return \"async_dependency_value\"\n\n\ndef sync_dep() -> str:\n    return \"sync_dependency_value\"\n\n\n\n\nclass Output(BaseModel):\n    sync_dep: str\n    async_dep: str\n\n\n# > Inject dependencies\n@hatchet.task()\nasync def async_task_with_dependencies(\n    input: EmptyModel,\n    ctx: Context,\n    async_dep: Annotated[str, Depends(async_dep)],\n    sync_dep: Annotated[str, Depends(sync_dep)],\n) -> Output:\n    return Output(\n        sync_dep=sync_dep,\n        async_dep=async_dep,\n    )\n\n\n\n\n@hatchet.task()\ndef sync_task_with_dependencies(\n    input: EmptyModel,\n    ctx: Context,\n    async_dep: Annotated[str, Depends(async_dep)],\n    sync_dep: Annotated[str, Depends(sync_dep)],\n) -> Output:\n    return Output(\n        sync_dep=sync_dep,\n        async_dep=async_dep,\n    )\n\n\n@hatchet.durable_task()\nasync def durable_async_task_with_dependencies(\n    input: EmptyModel,\n    ctx: DurableContext,\n    async_dep: Annotated[str, Depends(async_dep)],\n    sync_dep: Annotated[str, Depends(sync_dep)],\n) -> Output:\n    return Output(\n        sync_dep=sync_dep,\n        async_dep=async_dep,\n    )\n\n\n@hatchet.durable_task()\ndef durable_sync_task_with_dependencies(\n    input: EmptyModel,\n    ctx: DurableContext,\n    async_dep: Annotated[str, Depends(async_dep)],\n    sync_dep: Annotated[str, Depends(sync_dep)],\n) -> Output:\n    return Output(\n        sync_dep=sync_dep,\n        async_dep=async_dep,\n    )\n\n\ndi_workflow = hatchet.workflow(\n    name=\"dependency-injection-workflow\",\n)\n\n\n@di_workflow.task()\nasync def wf_async_task_with_dependencies(\n    input: EmptyModel,\n    ctx: Context,\n    async_dep: Annotated[str, Depends(async_dep)],\n    sync_dep: Annotated[str, Depends(sync_dep)],\n) -> Output:\n    return Output(\n        sync_dep=sync_dep,\n        async_dep=async_dep,\n    )\n\n\n@di_workflow.task()\ndef wf_sync_task_with_dependencies(\n    input: EmptyModel,\n    ctx: Context,\n    async_dep: Annotated[str, Depends(async_dep)],\n    sync_dep: Annotated[str, Depends(sync_dep)],\n) -> Output:\n    return Output(\n        sync_dep=sync_dep,\n        async_dep=async_dep,\n    )\n\n\n@di_workflow.durable_task()\nasync def wf_durable_async_task_with_dependencies(\n    input: EmptyModel,\n    ctx: DurableContext,\n    async_dep: Annotated[str, Depends(async_dep)],\n    sync_dep: Annotated[str, Depends(sync_dep)],\n) -> Output:\n    return Output(\n        sync_dep=sync_dep,\n        async_dep=async_dep,\n    )\n\n\n@di_workflow.durable_task()\ndef wf_durable_sync_task_with_dependencies(\n    input: EmptyModel,\n    ctx: DurableContext,\n    async_dep: Annotated[str, Depends(async_dep)],\n    sync_dep: Annotated[str, Depends(sync_dep)],\n) -> Output:\n    return Output(\n        sync_dep=sync_dep,\n        async_dep=async_dep,\n    )\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"dependency-injection-worker\",\n        workflows=[\n            async_task_with_dependencies,\n            sync_task_with_dependencies,\n            durable_async_task_with_dependencies,\n            durable_async_task_with_dependencies,\n            di_workflow,\n        ],\n    )\n    worker.start()\n\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/dependency_injection/worker.py",
  "blocks": {
    "declare_dependencies": {
      "start": 13,
      "stop": 20
    },
    "inject_dependencies": {
      "start": 29,
      "stop": 41
    }
  },
  "highlights": {}
};

export default snippet;
