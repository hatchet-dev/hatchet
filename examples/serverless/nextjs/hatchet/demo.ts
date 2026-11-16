/**
 * The demo triggers (the form on the page, its server action and `GET /api/echo`) create
 * runs with the app's tenant token for anyone who can reach the app, and carry no
 * authorization of their own. They are on under `next dev` and `vc dev`, and off on a
 * deployment unless HATCHET_DEMO_TRIGGERS=true is set. The signed endpoint under
 * app/api/hatchet is separate: the operator's signature protects it, not these routes.
 */
export function demoTriggersEnabled(): boolean {
  return process.env.HATCHET_DEMO_TRIGGERS === "true" || process.env.NODE_ENV === "development";
}

export const DEMO_TRIGGERS_OFF =
  "demo triggers are off on this deployment; set HATCHET_DEMO_TRIGGERS=true to enable them";

/** The message a demo trigger accepts: a string of at most 200 characters. */
export function demoMessage(value: unknown): string {
  return String(value ?? "hello").slice(0, 200);
}
