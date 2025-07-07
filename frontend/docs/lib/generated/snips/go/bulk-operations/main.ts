import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\t\"time\"\n\n\t\"github.com/google/uuid\"\n\t\"github.com/oapi-codegen/runtime/types\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client/rest\"\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n)\n\nfunc main() {\n\t// > Setup\n\n\thatchet, err := v1.NewHatchetClient()\n\tif err != nil {\n\t\tlog.Fatalf(\"failed to create hatchet client: %v\", err)\n\t}\n\n\tctx := context.Background()\n\n\tworkflows, err := hatchet.Workflows().List(ctx, nil)\n\tif err != nil {\n\t\tlog.Fatalf(\"failed to list workflows: %v\", err)\n\t}\n\n\tif workflows == nil || workflows.Rows == nil || len(*workflows.Rows) == 0 {\n\t\tlog.Fatalf(\"no workflows found\")\n\t}\n\n\tselectedWorkflow := (*workflows.Rows)[0]\n\tselectedWorkflowUUID := uuid.MustParse(selectedWorkflow.Metadata.Id)\n\n\n\t// > List runs\n\tworkflowRuns, err := hatchet.Runs().List(ctx, rest.V1WorkflowRunListParams{\n\t\tWorkflowIds: &[]types.UUID{selectedWorkflowUUID},\n\t})\n\tif err != nil || workflowRuns == nil || workflowRuns.JSON200 == nil || workflowRuns.JSON200.Rows == nil {\n\t\tlog.Fatalf(\"failed to list workflow runs for workflow %s: %v\", selectedWorkflow.Name, err)\n\t}\n\n\tvar runIds []types.UUID\n\n\tfor _, run := range workflowRuns.JSON200.Rows {\n\t\trunIds = append(runIds, uuid.MustParse(run.Metadata.Id))\n\t}\n\n\n\t// > Cancel by run ids\n\t_, err = hatchet.Runs().Cancel(ctx, rest.V1CancelTaskRequest{\n\t\tExternalIds: &runIds,\n\t})\n\tif err != nil {\n\t\tlog.Fatalf(\"failed to cancel runs by ids: %v\", err)\n\t}\n\n\n\t// > Cancel by filters\n\ttNow := time.Now().UTC()\n\n\t_, err = hatchet.Runs().Cancel(ctx, rest.V1CancelTaskRequest{\n\t\tFilter: &rest.V1TaskFilter{\n\t\t\tSince:              tNow.Add(-24 * time.Hour),\n\t\t\tUntil:              &tNow,\n\t\t\tStatuses:           &[]rest.V1TaskStatus{rest.V1TaskStatusRUNNING},\n\t\t\tWorkflowIds:        &[]types.UUID{selectedWorkflowUUID},\n\t\t\tAdditionalMetadata: &[]string{`{\"key\": \"value\"}`},\n\t\t},\n\t})\n\tif err != nil {\n\t\tlog.Fatalf(\"failed to cancel runs by filters: %v\", err)\n\t}\n\n\n\tfmt.Println(\"cancelled all runs for workflow\", selectedWorkflow.Name)\n}\n",
  "source": "out/go/bulk-operations/main.go",
  "blocks": {
    "setup": {
      "start": 18,
      "stop": 37
    },
    "list_runs": {
      "start": 40,
      "stop": 52
    },
    "cancel_by_run_ids": {
      "start": 55,
      "stop": 61
    },
    "cancel_by_filters": {
      "start": 64,
      "stop": 78
    }
  },
  "highlights": {}
};

export default snippet;
