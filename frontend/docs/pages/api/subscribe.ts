import { LoopsClient } from "loops";
import type { NextApiRequest, NextApiResponse } from 'next';

const mailingLists = {
  'newsletter': 'cmapskb8v00za0iyib5ux3r6i'
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { email } = req.body;

  if (!email) {
    return res.status(400).json({ error: 'Email is required' });
  }

  if (!process.env.LOOPS_API_KEY) {
    return res.status(500).json({ error: 'Server configuration error' });
  }

  const loops = new LoopsClient(process.env.LOOPS_API_KEY);

  try {
    await loops.createContact(email, {}, {
      [mailingLists.newsletter]: true,
    });
    return res.status(200).json({ success: true });
  } catch (error) {
    console.error('Subscription error:', error);
    return res.status(500).json({ error: 'Failed to subscribe' });
  }
} 