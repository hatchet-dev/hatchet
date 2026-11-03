let telemetryStarted = false;

export async function setupTelemetry(): Promise<void> {
  if (
    telemetryStarted ||
    process.env.HATCHET_SANDBOX_OTEL === 'false' ||
    process.env.BRICKIE_OTEL === 'false'
  ) {
    return;
  }
  telemetryStarted = true;
  try {
    const [{ registerInstrumentations }, { HatchetInstrumentor }] = await Promise.all([
      import('@opentelemetry/instrumentation'),
      import('@hatchet-dev/typescript-sdk/opentelemetry/index.js'),
    ]);
    registerInstrumentations({ instrumentations: [new HatchetInstrumentor()] });
  } catch (err) {
    console.error(`sandbox: opentelemetry disabled: ${(err as Error).message}`);
  }
}
