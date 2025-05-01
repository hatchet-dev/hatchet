import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { crazyWorkflow, declaredType, inferredType, inferredTypeDurable } from './workflow';\n\nasync function main() {\n  const declaredTypeRun = declaredType.run({\n    Message: 'hello',\n  });\n\n  const inferredTypeRun = inferredType.run({\n    Message: 'hello',\n  });\n\n  const crazyWorkflowRun = crazyWorkflow.run({\n    Message: 'hello',\n  });\n\n  const inferredTypeDurableRun = inferredTypeDurable.run({\n    Message: 'Durable Task',\n  });\n\n  const [declaredTypeResult, inferredTypeResult, inferredTypeDurableResult, crazyWorkflowResult] =\n    await Promise.all([declaredTypeRun, inferredTypeRun, inferredTypeDurableRun, crazyWorkflowRun]);\n\n  console.log('declaredTypeResult', declaredTypeResult);\n  console.log('inferredTypeResult', inferredTypeResult);\n  console.log('inferredTypeDurableResult', inferredTypeDurableResult);\n  console.log('crazyWorkflowResult', crazyWorkflowResult);\n  console.log('declaredTypeResult.TransformedMessage', declaredTypeResult.TransformedMessage);\n  console.log('inferredTypeResult.TransformedMessage', inferredTypeResult.TransformedMessage);\n  console.log(\n    'inferredTypeDurableResult.TransformedMessage',\n    inferredTypeDurableResult.TransformedMessage\n  );\n  console.log('crazyWorkflowResult.TransformedMessage', crazyWorkflowResult.TransformedMessage);\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/inferred-typing/run.ts',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
