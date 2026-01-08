export default function Overview() {
  return (
    <div className="flex h-full w-full flex-col gap-4 p-6">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-sm text-muted-foreground">
          Get a quick overview of your workflows, runs, and workers.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {/* Placeholder cards for overview content */}
        <div className="rounded-lg border bg-card p-6">
          <h3 className="text-sm font-medium">Total Runs</h3>
          <p className="mt-2 text-3xl font-bold">0</p>
        </div>

        <div className="rounded-lg border bg-card p-6">
          <h3 className="text-sm font-medium">Active Workers</h3>
          <p className="mt-2 text-3xl font-bold">0</p>
        </div>

        <div className="rounded-lg border bg-card p-6">
          <h3 className="text-sm font-medium">Workflows</h3>
          <p className="mt-2 text-3xl font-bold">0</p>
        </div>
      </div>
    </div>
  );
}
