//go:build ignore

// Wrapper usage. `CreateWorkflow` is a reusable factory marked with the
// `@hatchet-workflow` comment; the DAG is defined at the USAGE site below
// (in Register) and renders on `ordersDag`. (build-ignored: illustrative,
// the types are not real.)

package wrapperdag

type Input struct{}

// @hatchet-workflow
func CreateWorkflow(client HatchetClient, name string) WorkflowBase {
	return client.NewWorkflow(name)
}

// ── Usage: the DAG shape is defined here and renders on `ordersDag` ──
func Register(client HatchetClient) WorkflowBase {
	run := func(ctx HatchetContext, input Input) (any, error) { return nil, nil }

	ordersDag := CreateWorkflow(client, "orders-dag")
	start := ordersDag.NewTask("start", run)
	branchA := ordersDag.NewTask("branch-a", run, WithParents(start))
	branchB := ordersDag.NewTask("branch-b", run, WithParents(start))
	_ = ordersDag.NewTask("join", run, WithParents(branchA, branchB))

	return ordersDag
}
