"use client";

import { useState, useTransition } from "react";
import { triggerTask, type TriggerResult } from "./actions";

export function TriggerForm() {
  const [result, setResult] = useState<TriggerResult | undefined>();
  const [pending, startTransition] = useTransition();

  return (
    <form
      action={(formData) => startTransition(async () => setResult(await triggerTask(formData)))}
    >
      <label>
        Task{" "}
        <select name="task" defaultValue="echo">
          <option value="echo">echo</option>
          <option value="sleep-then-echo">sleep-then-echo (durable)</option>
          <option value="stream-progress">stream-progress (durable stream)</option>
          <option value="stream-echo">stream-echo (durable stream, read back over the socket)</option>
        </select>
      </label>{" "}
      <label>
        Message <input name="message" defaultValue="hello" />
      </label>{" "}
      <button type="submit" disabled={pending}>
        {pending ? "Running" : "Run"}
      </button>
      {result && (
        <pre style={{ background: "#f4f4f4", padding: "1rem", overflowX: "auto" }}>
          {JSON.stringify(result, null, 2)}
        </pre>
      )}
    </form>
  );
}
