/**
 * Example: using the custom workflow builder.
 *
 * Because `createWorkflowBuilder` is annotated with `@hatchet-workflow`,
 * the VS Code extension detects `reportWorkflow` as a workflow variable and
 * shows a "Show Hatchet DAG" CodeLens above its declaration — no config needed.
 */
import { createWorkflowBuilder } from './custom-wrapper';

interface ReportInput {
  reportId: string;
  format: 'pdf' | 'csv';
}

// The extension detects this as a workflow (via the @hatchet-workflow annotation
// on createWorkflowBuilder) and places a CodeLens above this line.
const reportWorkflow = createWorkflowBuilder<ReportInput>({
  name: 'generate-report',
  createServices: async (_input) => ({
    db: null, // replace with real DB client
  }),
});

const fetchData = reportWorkflow.task('fetch-data', {
  fn: async ({ input, services }) => {
    return { rows: [] as unknown[] };
  },
});

const formatData = reportWorkflow.task('format-data', {
  parents: [fetchData],
  fn: async ({ input, services }) => {
    return { formatted: '' };
  },
});

const generateFile = reportWorkflow.task('generate-file', {
  parents: [formatData],
  fn: async ({ input, services }) => {
    return { fileUrl: '' };
  },
});

const sendNotification = reportWorkflow.task('send-notification', {
  parents: [generateFile],
  fn: async ({ input, services }) => {
    return { sent: true };
  },
});

export default reportWorkflow.build();
