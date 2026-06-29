# Wrapper usage. `create_workflow` is a reusable factory marked with the
# `@hatchet-workflow` comment; the DAG is defined at the USAGE site below and
# renders on `orders_dag`.

# @hatchet-workflow
def create_workflow(hatchet, name)
  hatchet.workflow(name: name)
end

# ── Usage: the DAG shape is defined here and renders on `orders_dag` ──
orders_dag = create_workflow(hatchet, name: "orders-dag")

start = orders_dag.task(:start) do |input, ctx|
  {}
end

branch_a = orders_dag.task(:branch_a, parents: [start]) do |input, ctx|
  {}
end

branch_b = orders_dag.task(:branch_b, parents: [start]) do |input, ctx|
  {}
end

orders_dag.task(:join, parents: [branch_a, branch_b]) do |input, ctx|
  {}
end
