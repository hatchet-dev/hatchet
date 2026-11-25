// Serves the example's page. It never talks to a session directly: a SessionClient publishes to the session's inbox,
// wakes the session, and relays its views to the browser as server-sent events.
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';
import express, { type NextFunction, type Request, type Response } from 'express';
import { z } from 'zod';
import { ClientMessage, SessionClient, SessionId } from '@hatchet-dev/pi-durable';
import { hatchet } from './agent.js';

const PORT = Number(process.env.PORT ?? 3334);
const HOST = process.env.HOST ?? '127.0.0.1';
const PAGE = path.join(path.dirname(fileURLToPath(import.meta.url)), 'index.html');

const sessions = new SessionClient(hatchet, { taskName: 'pi-session' });

const app = express();
app.use(express.json({ limit: '200kb' }));

app.get('/', (_req, res) => res.sendFile(PAGE));

app.post('/api/sessions', async (_req, res) => {
  res.status(201).json({ sessionId: await sessions.create() });
});

app.param('sessionId', (_req, res, next, value) => {
  if (SessionId.safeParse(value).success) next();
  else res.status(404).end();
});

app.post('/api/sessions/:sessionId/wake', async (req, res) => {
  await sessions.wake(req.params.sessionId);
  res.status(202).json({});
});

app.post('/api/sessions/:sessionId/inbox', async (req, res) => {
  const parsed = ClientMessage.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ error: z.prettifyError(parsed.error) });
    return;
  }
  res.status(202).json({ requestId: await sessions.send(req.params.sessionId, parsed.data) });
});

app.get('/api/sessions/:sessionId/view', async (req, res) => {
  const abort = new AbortController();
  req.on('close', () => abort.abort());
  res.set({
    'content-type': 'text/event-stream',
    'cache-control': 'no-cache',
    connection: 'keep-alive',
  });
  res.flushHeaders();
  res.write(': connected\n\n');
  try {
    for await (const view of sessions.views(req.params.sessionId, abort.signal))
      res.write(`data: ${JSON.stringify(view)}\n\n`);
  } catch (error) {
    if (!abort.signal.aborted) console.error(`view ${req.params.sessionId}:`, error);
  }
  res.end();
});

app.use((error: Error & { status?: number }, _req: Request, res: Response, _next: NextFunction) => {
  // express.json() rejects a malformed body with status 400; anything else is ours.
  const status = error.status ?? 500;
  if (status >= 500) console.error(error);
  if (!res.headersSent) {
    res.status(status).json({ error: status >= 500 ? 'Internal server error' : error.message });
  }
});

app.listen(PORT, HOST, () => console.log(`pi on Hatchet example on http://${HOST}:${PORT}`));
