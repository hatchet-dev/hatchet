
import { crazyWorkflow, declaredType, inferredType, inferredTypeDurable } from './workflow';

async function main() {
  const declaredTypeRun = declaredType.run({
    Message: 'hello',
  });

  const inferredTypeRun = inferredType.run({
    Message: 'hello',
  });

  const crazyWorkflowRun = crazyWorkflow.run({
    Message: 'hello',
  });

  const inferredTypeDurableRun = inferredTypeDurable.run({
    Message: 'Durable Task',
  });

  const [declaredTypeResult, inferredTypeResult, inferredTypeDurableResult, crazyWorkflowResult] =
    await Promise.all([declaredTypeRun, inferredTypeRun, inferredTypeDurableRun, crazyWorkflowRun]);

  console.log('declaredTypeResult', declaredTypeResult);
  console.log('inferredTypeResult', inferredTypeResult);
  console.log('inferredTypeDurableResult', inferredTypeDurableResult);
  console.log('crazyWorkflowResult', crazyWorkflowResult);
  console.log('declaredTypeResult.TransformedMessage', declaredTypeResult.TransformedMessage);
  console.log('inferredTypeResult.TransformedMessage', inferredTypeResult.TransformedMessage);
  console.log(
    'inferredTypeDurableResult.TransformedMessage',
    inferredTypeDurableResult.TransformedMessage
  );
  console.log('crazyWorkflowResult.TransformedMessage', crazyWorkflowResult.TransformedMessage);
}

if (require.main === module) {
  main();
}
