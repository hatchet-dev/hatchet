import React, { useMemo } from 'react';
import { WorkflowRunShape } from '@/lib/api';
import { WorkflowRunInputDialog } from '../workflow-run-input';
import StepRunOutput from './step-run-output';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { ScrollArea } from '@/components/ui/scroll-area';

interface StepRunDetailProps {
  stepRunId: string;
  workflowRun: WorkflowRunShape;
}

const StepRunDetail: React.FC<StepRunDetailProps> = ({
  stepRunId,
  workflowRun,
}) => {
  const [displayName, runId] = useMemo(() => {
    const parts = workflowRun?.displayName?.split('-');
    if (!parts) {
      return [null, null];
    }

    return [parts[0], parts[1]];
  }, [workflowRun?.displayName]);

  const stepRun = useMemo(() => {
    return workflowRun.jobRuns
      ?.flatMap((jr) => jr.stepRuns)
      .filter((x) => !!x)
      .find((x) => x.metadata.id === stepRunId);
  }, [workflowRun, stepRunId]);

  const step = useMemo(() => {
    return workflowRun.jobRuns
      ?.flatMap((jr) => jr.job?.steps)
      .filter((x) => !!x)
      .find((x) => x.metadata.id === stepRun?.stepId);
  }, [workflowRun, stepRun]);

  return (
    <Card className="w-80 h-screen">
      <CardHeader>
        <CardTitle>{step?.readableId || 'Step Run Detail'}</CardTitle>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[calc(100vh-100px)]">
          <Accordion
            type="single"
            defaultValue="output"
            collapsible
            className="w-full"
          >
            <AccordionItem value="output">
              <AccordionTrigger>Output</AccordionTrigger>
              <AccordionContent>
                {stepRun && (
                  <StepRunOutput stepRun={stepRun} workflowRun={workflowRun} />
                )}
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="parent-step-data">
              <AccordionTrigger>Parent Step Data</AccordionTrigger>
              <AccordionContent>
                {stepRun && <WorkflowRunInputDialog run={workflowRun} />}
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="workflow-run-input">
              <AccordionTrigger>Workflow Run Input</AccordionTrigger>
              <AccordionContent>
                <WorkflowRunInputDialog run={workflowRun} />
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </ScrollArea>
      </CardContent>
    </Card>
  );
};

export default StepRunDetail;
