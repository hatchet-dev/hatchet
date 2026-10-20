import { demoTriggersEnabled } from "@/hatchet/demo";
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
      {demoTriggersEnabled() ? (
        <TriggerForm />
      ) : (
        <p>
          The demo triggers are off on this deployment. Set{" "}
          <code>HATCHET_DEMO_TRIGGERS=true</code> to let anyone who can reach this page create
          runs with the app&apos;s token.
        </p>
      )}
    </main>
  );
}
