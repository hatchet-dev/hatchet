import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import sleep from \'@hatchet-dev/typescript-sdk/util/sleep\';\nimport { Or } from \'@hatchet-dev/typescript-sdk/v1/conditions\';\nimport { hatchet } from \'../hatchet-client\';\n\ntype DagInput = {};\n\ntype DagOutput = {\n  \'first-task\': {\n    Completed: boolean;\n  };\n  \'second-task\': {\n    Completed: boolean;\n  };\n};\n\nexport const dagWithConditions = hatchet.workflow<DagInput, DagOutput>({\n  name: \'simple\',\n});\n\nconst firstTask = dagWithConditions.task({\n  name: \'first-task\',\n  fn: async () => {\n    await sleep(2000);\n    return {\n      Completed: true,\n    };\n  },\n});\n\ndagWithConditions.task({\n  name: \'second-task\',\n  parents: [firstTask],\n  waitFor: Or({ eventKey: \'user:event\' }, { sleepFor: \'10s\' }),\n  fn: async (_, ctx) => {\n    console.log(\'triggered by condition\', ctx.triggers());\n\n    return {\n      Completed: true,\n    };\n  },\n});\n',
  'source': 'out/typescript/dag_match_condition/workflow.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
