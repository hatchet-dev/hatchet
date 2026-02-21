/**
 * Minimal shape for DAG visualization.
 * Mirrors WorkflowRunShapeItemForWorkflowRunDetails from the API,
 * with optional runtime fields for live workflow run decoration.
 */
export interface DagNode {
  /** Unique stable ID (variable name from parser, or stepId from API) */
  stepId: string;
  /** Display name (the `name` property in .task({name:'...'})) */
  taskName: string;
  /** IDs of tasks that depend on this task (downstream) */
  childrenStepIds: string[];
  /** Optional runtime status for live workflow run decoration */
  status?: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  /** Optional runtime duration in milliseconds */
  durationMs?: number;
  /** Whether this task was skipped */
  isSkipped?: boolean;
}

export type DagShape = DagNode[];

/**
 * Data passed to each ReactFlow node renderer.
 */
export interface NodeDisplayData {
  taskName: string;
  graphVariant: 'default' | 'input_only' | 'output_only' | 'none';
  status?: DagNode['status'];
  durationMs?: number;
  isSkipped?: boolean;
  onClick?: () => void;
}
