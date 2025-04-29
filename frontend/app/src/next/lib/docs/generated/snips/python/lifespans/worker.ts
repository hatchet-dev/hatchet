import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from typing import AsyncGenerator, cast\nfrom uuid import UUID\n\nfrom psycopg_pool import ConnectionPool\nfrom pydantic import BaseModel, ConfigDict\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n# > Use the lifespan in a task\nclass TaskOutput(BaseModel):\n    num_rows: int\n    external_ids: list[UUID]\n\n\nlifespan_workflow = hatchet.workflow(name='LifespanWorkflow')\n\n\n@lifespan_workflow.task()\ndef sync_lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:\n    pool = cast(Lifespan, ctx.lifespan).pool\n\n    with pool.connection() as conn:\n        query = conn.execute('SELECT * FROM v1_lookup_table_olap LIMIT 5;')\n        rows = query.fetchall()\n\n        for row in rows:\n            print(row)\n\n        print('executed sync task with lifespan', ctx.lifespan)\n\n        return TaskOutput(\n            num_rows=len(rows),\n            external_ids=[cast(UUID, row[0]) for row in rows],\n        )\n\n\n\n\n\n@lifespan_workflow.task()\nasync def async_lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:\n    pool = cast(Lifespan, ctx.lifespan).pool\n\n    with pool.connection() as conn:\n        query = conn.execute('SELECT * FROM v1_lookup_table_olap LIMIT 5;')\n        rows = query.fetchall()\n\n        for row in rows:\n            print(row)\n\n        print('executed async task with lifespan', ctx.lifespan)\n\n        return TaskOutput(\n            num_rows=len(rows),\n            external_ids=[cast(UUID, row[0]) for row in rows],\n        )\n\n\n# > Define a lifespan\nclass Lifespan(BaseModel):\n    model_config = ConfigDict(arbitrary_types_allowed=True)\n\n    foo: str\n    pool: ConnectionPool\n\n\nasync def lifespan() -> AsyncGenerator[Lifespan, None]:\n    print('Running lifespan!')\n    with ConnectionPool('postgres://hatchet:hatchet@localhost:5431/hatchet') as pool:\n        yield Lifespan(\n            foo='bar',\n            pool=pool,\n        )\n\n    print('Cleaning up lifespan!')\n\n\nworker = hatchet.worker(\n    'test-worker', slots=1, workflows=[lifespan_workflow], lifespan=lifespan\n)\n\n\n\ndef main() -> None:\n    worker.start()\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/lifespans/worker.py',
  blocks: {
    use_the_lifespan_in_a_task: {
      start: 13,
      stop: 39,
    },
    define_a_lifespan: {
      start: 62,
      stop: 82,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
