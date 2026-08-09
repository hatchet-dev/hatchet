import { LoopsClient } from "loops";

const mailingLists = {
  newsletter: "cmapskb8v00za0iyib5ux3r6i",
};

export async function POST(req: Request) {
  let email: string | undefined;
  try {
    ({ email } = await req.json());
  } catch {
    void 0;
  }

  if (!email) {
    return Response.json({ error: "Email is required" }, { status: 400 });
  }

  if (!process.env.LOOPS_API_KEY) {
    return Response.json(
      { error: "Server configuration error" },
      { status: 500 },
    );
  }

  const loops = new LoopsClient(process.env.LOOPS_API_KEY);

  try {
    await loops.createContact(
      email,
      {},
      {
        [mailingLists.newsletter]: true,
      },
    );
    return Response.json({ success: true });
  } catch (error) {
    console.error("Subscription error:", error);
    return Response.json({ error: "Failed to subscribe" }, { status: 500 });
  }
}
