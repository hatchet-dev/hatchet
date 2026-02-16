# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: false) unless defined?(HATCHET)

SYNC_DEPENDENCY_VALUE = "sync_dependency_value"
ASYNC_DEPENDENCY_VALUE = "async_dependency_value"
SYNC_CM_DEPENDENCY_VALUE = "sync_cm_dependency_value"
ASYNC_CM_DEPENDENCY_VALUE = "async_cm_dependency_value"
CHAINED_CM_VALUE = "chained_cm_value"
CHAINED_ASYNC_CM_VALUE = "chained_async_cm_value"

# > Declare dependencies (Ruby uses callable objects instead of Python's Depends)
sync_dep = ->(_input, _ctx) { SYNC_DEPENDENCY_VALUE }
async_dep = ->(_input, _ctx) { ASYNC_DEPENDENCY_VALUE }

sync_cm_dep = lambda { |_input, _ctx, deps|
  "#{SYNC_CM_DEPENDENCY_VALUE}_#{deps[:sync_dep]}"
}

async_cm_dep = lambda { |_input, _ctx, deps|
  "#{ASYNC_CM_DEPENDENCY_VALUE}_#{deps[:async_dep]}"
}

chained_dep = ->(_input, _ctx, deps) { "chained_#{CHAINED_CM_VALUE}" }
chained_async_dep = ->(_input, _ctx, deps) { "chained_#{CHAINED_ASYNC_CM_VALUE}" }

# !!

# > Inject dependencies
ASYNC_TASK_WITH_DEPS = HATCHET.task(
  name: "async_task_with_dependencies",
  deps: {
    sync_dep: sync_dep,
    async_dep: async_dep,
    sync_cm_dep: sync_cm_dep,
    async_cm_dep: async_cm_dep,
    chained_dep: chained_dep,
    chained_async_dep: chained_async_dep
  }
) do |input, ctx|
  {
    "sync_dep" => ctx.deps[:sync_dep],
    "async_dep" => ctx.deps[:async_dep],
    "async_cm_dep" => ctx.deps[:async_cm_dep],
    "sync_cm_dep" => ctx.deps[:sync_cm_dep],
    "chained_dep" => ctx.deps[:chained_dep],
    "chained_async_dep" => ctx.deps[:chained_async_dep]
  }
end

SYNC_TASK_WITH_DEPS = HATCHET.task(
  name: "sync_task_with_dependencies",
  deps: {
    sync_dep: sync_dep,
    async_dep: async_dep,
    sync_cm_dep: sync_cm_dep,
    async_cm_dep: async_cm_dep,
    chained_dep: chained_dep,
    chained_async_dep: chained_async_dep
  }
) do |input, ctx|
  {
    "sync_dep" => ctx.deps[:sync_dep],
    "async_dep" => ctx.deps[:async_dep],
    "async_cm_dep" => ctx.deps[:async_cm_dep],
    "sync_cm_dep" => ctx.deps[:sync_cm_dep],
    "chained_dep" => ctx.deps[:chained_dep],
    "chained_async_dep" => ctx.deps[:chained_async_dep]
  }
end

DURABLE_ASYNC_TASK_WITH_DEPS = HATCHET.durable_task(
  name: "durable_async_task_with_dependencies",
  deps: {
    sync_dep: sync_dep,
    async_dep: async_dep,
    sync_cm_dep: sync_cm_dep,
    async_cm_dep: async_cm_dep,
    chained_dep: chained_dep,
    chained_async_dep: chained_async_dep
  }
) do |input, ctx|
  {
    "sync_dep" => ctx.deps[:sync_dep],
    "async_dep" => ctx.deps[:async_dep],
    "async_cm_dep" => ctx.deps[:async_cm_dep],
    "sync_cm_dep" => ctx.deps[:sync_cm_dep],
    "chained_dep" => ctx.deps[:chained_dep],
    "chained_async_dep" => ctx.deps[:chained_async_dep]
  }
end

DURABLE_SYNC_TASK_WITH_DEPS = HATCHET.durable_task(
  name: "durable_sync_task_with_dependencies",
  deps: {
    sync_dep: sync_dep,
    async_dep: async_dep,
    sync_cm_dep: sync_cm_dep,
    async_cm_dep: async_cm_dep,
    chained_dep: chained_dep,
    chained_async_dep: chained_async_dep
  }
) do |input, ctx|
  {
    "sync_dep" => ctx.deps[:sync_dep],
    "async_dep" => ctx.deps[:async_dep],
    "async_cm_dep" => ctx.deps[:async_cm_dep],
    "sync_cm_dep" => ctx.deps[:sync_cm_dep],
    "chained_dep" => ctx.deps[:chained_dep],
    "chained_async_dep" => ctx.deps[:chained_async_dep]
  }
end

DI_WORKFLOW = HATCHET.workflow(name: "dependency-injection-workflow")

# Workflow tasks with dependencies follow the same pattern
DI_WORKFLOW.task(:wf_task_with_dependencies) do |input, ctx|
  {
    "sync_dep" => SYNC_DEPENDENCY_VALUE,
    "async_dep" => ASYNC_DEPENDENCY_VALUE
  }
end

# !!

def main
  worker = HATCHET.worker(
    "dependency-injection-worker",
    workflows: [
      ASYNC_TASK_WITH_DEPS,
      SYNC_TASK_WITH_DEPS,
      DURABLE_ASYNC_TASK_WITH_DEPS,
      DURABLE_SYNC_TASK_WITH_DEPS,
      DI_WORKFLOW
    ]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
