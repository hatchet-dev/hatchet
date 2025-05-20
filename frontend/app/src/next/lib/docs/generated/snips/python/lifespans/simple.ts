import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    '# > Lifespan\n\nfrom typing import AsyncGenerator, cast\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\nclass Lifespan(BaseModel):\n    foo: str\n    pi: float\n\n\nasync def lifespan() -> AsyncGenerator[Lifespan, None]:\n    yield Lifespan(foo="bar", pi=3.14)\n\n\n@hatchet.task(name="LifespanWorkflow")\ndef lifespan_task(input: EmptyModel, ctx: Context) -> Lifespan:\n    return cast(Lifespan, ctx.lifespan)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "test-worker", slots=1, workflows=[lifespan_task], lifespan=lifespan\n    )\n    worker.start()\n\n\n\nif __name__ == "__main__":\n    main()\n',
  source: 'out/python/lifespans/simple.py',
  blocks: {
    lifespan: {
      start: 2,
      stop: 32,
    },
  },
  highlights: {},
};

export default snippet;
