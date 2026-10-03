import { TriggerForm } from "./trigger-form";

export default function Page() {
  return (
    <main>
      <h1>Hatchet serverless on Next.js</h1>
      <p>
        This app serves <code>echo</code> and <code>sleep-then-echo</code> to the Hatchet
        serverless operator under <code>/api/hatchet</code>, and triggers them from a server
        action with the SDK&apos;s core client.
      </p>
      <TriggerForm />
    </main>
  );
}
