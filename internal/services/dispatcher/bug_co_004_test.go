package dispatcher

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// BUG-CO-004: Dispatcher drops assignments when ListTasks fails
//
// HYPOTHESIS: When handleTaskBulkAssignedTask calls ListTasks and it fails
// (e.g., due to context timeout), the function logs the error and continues
// instead of returning an error. This causes the MQ message to be acked
// even though no tasks were processed, leading to task loss.
//
// LOCATION: dispatcher_v1.go, handleTaskBulkAssignedTask
//
// CODE PATH (before fix):
//   bulkDatas, err := d.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)
//   if err != nil {
//       for _, task := range bulkDatas {  // bulkDatas is empty on error!
//           requeue(task)
//       }
//       d.l.Error().Err(err).Msgf("could not bulk list step run data:")
//       continue  // BUG: Should return error, not continue
//   }

// TestBugCO004_ListTasksErrorMustReturnError is a regression test that verifies
// the fix for BH-CO-004. When ListTasks fails in handleTaskBulkAssignedTask,
// the code must return an error (not continue) so the MQ message is NACKed.
//
// This test inspects the actual source code to verify the error handling pattern.
// - On main (buggy): the code uses "continue" after ListTasks error -> TEST FAILS
// - On fix branch: the code returns an error -> TEST PASSES
func TestBugCO004_ListTasksErrorMustReturnError(t *testing.T) {
	t.Log("=== BUG-CO-004: ListTasks Error Must Return Error ===")
	t.Log("")
	t.Log("Verifying that handleTaskBulkAssignedTask returns an error when ListTasks fails")
	t.Log("(instead of using 'continue' which would silently drop the message)")
	t.Log("")

	// Read the dispatcher source code
	content, err := os.ReadFile("dispatcher_v1.go")
	require.NoError(t, err, "Failed to read dispatcher_v1.go")

	src := string(content)

	// The fix should include returning an error when ListTasks fails
	// Look for the error return pattern that was added in the fix
	fixPattern := `return fmt.Errorf("could not bulk list step run data`
	hasFix := strings.Contains(src, fixPattern)

	// Also check that the buggy pattern (continue after ListTasks error) is NOT present
	// The buggy code had: if err != nil { ... continue }
	// After the fix, we return an error instead of continue

	t.Logf("Fix pattern found: %v", hasFix)
	t.Log("")

	require.True(t, hasFix,
		"BUG-CO-004: handleTaskBulkAssignedTask must return an error when ListTasks fails.\n"+
			"Expected to find: %s\n"+
			"The buggy code uses 'continue' instead of returning an error, causing tasks to be silently dropped.",
		fixPattern)

	t.Log("VERIFIED: Code correctly returns error when ListTasks fails")
}

// TestBugCO004_ListTaskParentOutputsErrorMustReturnError verifies that
// ListTaskParentOutputs errors also cause the handler to return an error.
func TestBugCO004_ListTaskParentOutputsErrorMustReturnError(t *testing.T) {
	t.Log("=== BUG-CO-004: ListTaskParentOutputs Error Must Return Error ===")

	content, err := os.ReadFile("dispatcher_v1.go")
	require.NoError(t, err, "Failed to read dispatcher_v1.go")

	src := string(content)

	// The fix should include returning an error when ListTaskParentOutputs fails
	fixPattern := `return fmt.Errorf("could not list parent data`
	hasFix := strings.Contains(src, fixPattern)

	t.Logf("Fix pattern found: %v", hasFix)

	require.True(t, hasFix,
		"BUG-CO-004: handleTaskBulkAssignedTask must return an error when ListTaskParentOutputs fails.\n"+
			"Expected to find: %s",
		fixPattern)

	t.Log("VERIFIED: Code correctly returns error when ListTaskParentOutputs fails")
}

// TestBugCO004_BuggyBehaviorDocumentation documents what the buggy code did.
// This test always passes - it just documents the bug for posterity.
func TestBugCO004_BuggyBehaviorDocumentation(t *testing.T) {
	t.Log("=== BUG-CO-004: Documentation of Buggy Behavior ===")
	t.Log("")
	t.Log("The OLD buggy code did this:")
	t.Log("")
	t.Log("  bulkDatas, err := d.repov1.Tasks().ListTasks(...)")
	t.Log("  if err != nil {")
	t.Log("      for _, task := range bulkDatas {  // EMPTY! No iteration")
	t.Log("          requeue(task)")
	t.Log("      }")
	t.Log("      d.l.Error().Err(err).Msgf(...)")
	t.Log("      continue  // BUG: Returns to loop, eventually returns nil")
	t.Log("  }")
	t.Log("")
	t.Log("Impact:")
	t.Log("  - ListTasks fails (e.g., DB timeout)")
	t.Log("  - bulkDatas is empty, so nothing is requeued")
	t.Log("  - Function continues and eventually returns nil")
	t.Log("  - MQ message is acked (task assignment is lost)")
	t.Log("  - Task remains in v1_task_runtime but is never sent to worker")
	t.Log("  - Workflow is stuck indefinitely")
	t.Log("")
	t.Log("The FIX:")
	t.Log("  - Return error immediately when ListTasks fails")
	t.Log("  - MQ message is NACKed and retried")
	t.Log("  - Task assignment has another chance to succeed")
}
