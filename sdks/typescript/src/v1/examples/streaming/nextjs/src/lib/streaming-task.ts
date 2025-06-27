// Simple standalone streaming task for Next.js demo

interface WorkflowRunRef {
  getWorkflowRunId(): Promise<string>;
}

interface StreamingTask {
  runNoWait(params: any): Promise<WorkflowRunRef>;
}

class SimpleWorkflowRunRef implements WorkflowRunRef {
  private workflowRunId: string;

  constructor() {
    // Generate a unique workflow run ID
    this.workflowRunId = `workflow-run-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  async getWorkflowRunId(): Promise<string> {
    return this.workflowRunId;
  }
}

class SimpleStreamingTask implements StreamingTask {
  async runNoWait(params: any): Promise<WorkflowRunRef> {
    // Simulate starting a workflow
    return new SimpleWorkflowRunRef();
  }
}

export const streamingTask = new SimpleStreamingTask();
