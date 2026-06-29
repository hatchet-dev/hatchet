# Wrapper usage. `create_workflow` is a reusable factory marked with the
# `@hatchet-workflow` comment; the DAG is defined at the USAGE site below and
# renders on `orders_dag`.


# @hatchet-workflow
def create_workflow(hatchet, name):
    return hatchet.workflow(name=name)


# ── Usage: the DAG shape is defined here and renders on `orders_dag` ──
orders_dag = create_workflow(hatchet, name="orders-dag")


@orders_dag.task()
def start(input, ctx):
    return {}


@orders_dag.task(parents=[start])
def branch_a(input, ctx):
    return {}


@orders_dag.task(parents=[start])
def branch_b(input, ctx):
    return {}


@orders_dag.task(parents=[branch_a, branch_b])
def join(input, ctx):
    return {}
