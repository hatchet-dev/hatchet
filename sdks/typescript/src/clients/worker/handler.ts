import HatchetError from '@util/errors/hatchet-error';
import { createHmac } from 'crypto';
import { IncomingMessage, ServerResponse } from 'http';
import { Workflow } from '@hatchet/workflow';
import { Worker } from './worker';

export interface HandlerOpts {
  secret: string;
}

const okMessage = 'The Hatchet webhooks endpoint is up and running!';

export class WebhookHandler {
  // eslint-disable-next-line no-useless-constructor
  constructor(
    private worker: Worker,
    private workflows: Workflow[]
    // eslint-disable-next-line no-empty-function
  ) {}

  /**
   * Handles a request with a provided body, secret, and signature.
   *
   * @param {string | undefined} body - The body of the request.
   * @param {string | undefined} secret - The secret used for signature verification.
   * @param {string | string[] | undefined | null} signature - The signature of the request.
   *
   * @throws {HatchetError} - If no signature is provided or the signature is not a string.
   * @throws {HatchetError} - If no secret is provided.
   * @throws {HatchetError} - If no body is provided.
   */
  async handle(
    body: string | undefined,
    signature: string | string[] | undefined | null,
    secret: string | undefined
  ) {
    this.checkSignature(body, signature, secret);

    const action = JSON.parse(body!);

    await this.worker.handleAction(action);
  }

  private checkSignature(
    body: string | undefined,
    signature: string | string[] | undefined | null,
    secret: string | undefined
  ) {
    if (!signature || typeof signature !== 'string') {
      throw new HatchetError('No signature provided');
    }
    if (!secret) {
      throw new HatchetError('No secret provided');
    }
    if (!body) {
      throw new HatchetError('No body provided');
    }

    // verify hmac signature
    const actualSignature = createHmac('sha256', secret).update(body).digest('hex');
    if (actualSignature !== signature) {
      throw new HatchetError(`Invalid signature, expected ${actualSignature}, got ${signature}`);
    }
  }

  private async getHealthcheckResponse(
    body: string | undefined,
    signature: string | string[] | undefined | null,
    secret: string | undefined
  ) {
    this.checkSignature(body, signature, secret);

    for (const workflow of this.workflows) {
      await this.worker.registerWorkflow(workflow);
    }

    return {
      actions: Object.keys(this.worker.action_registry),
    };
  }

  /**
   * Express Handler
   *
   * This method is an asynchronous function that returns an Express middleware handler.
   * The handler function is responsible for handling incoming requests and invoking the
   * corresponding logic based on the provided secret.
   */
  expressHandler({ secret }: HandlerOpts) {
    return (req: any, res: any) => {
      if (req.method === 'GET') {
        res.status(200).send(okMessage);
        return;
      }

      if (req.method === 'PUT') {
        let { body } = req;

        if (typeof body !== 'string') {
          body = JSON.stringify(body);
        }

        this.getHealthcheckResponse(body, req.headers['x-hatchet-signature'], secret)
          .then((resp) => {
            res.status(200).json(resp);
          })
          .catch((err) => {
            res.status(500);
            this.worker.logger.error(`Error handling request: ${err.message}`);
          });
        return;
      }

      if (req.method !== 'POST') {
        res.status(405).json({ error: 'Method not allowed' });
        return;
      }

      let action = req.body;

      if (typeof action !== 'string') {
        action = JSON.stringify(action);
      }

      this.handle(action, req.headers['x-hatchet-signature'], secret)
        .then(() => {
          res.status(200);
        })
        .catch((err) => {
          res.status(500);
          this.worker.logger.error(`Error handling request: ${err.message}`);
        });
    };
  }

  /**
   * A method that returns an HTTP request handler.
   */
  httpHandler({ secret }: HandlerOpts) {
    return (req: IncomingMessage, res: ServerResponse) => {
      const handle = async () => {
        if (req.method === 'GET') {
          res.writeHead(200, { 'Content-Type': 'application/json' });
          res.write(okMessage);
          res.end();
          return;
        }

        const body = await this.getBody(req);

        if (req.method === 'PUT') {
          const resp = await this.getHealthcheckResponse(
            body,
            req.headers['x-hatchet-signature'],
            secret
          );
          res.writeHead(200, { 'Content-Type': 'application/json' });
          res.write(JSON.stringify(resp));
          res.end();
          return;
        }

        if (req.method !== 'POST') {
          res.writeHead(405, { 'Content-Type': 'application/json' });
          res.write(JSON.stringify({ error: 'Method not allowed' }));
          res.end();
          return;
        }

        await this.handle(body, req.headers['x-hatchet-signature'], secret);

        res.writeHead(200, 'OK');
        res.end();
      };

      handle().catch((e) => {
        this.worker.logger.error(`Error handling request: ${e.message}`);
        res.writeHead(500, 'Internal server error');
        res.end();
      });
    };
  }

  /**
   * A method that returns a Next.js pages router request handler.
   */
  nextJSPagesHandler({ secret }: HandlerOpts) {
    return async (req: any, res: any) => {
      if (req.method === 'GET') {
        return res.status(200).send(okMessage);
      }
      const sig = req.headers['x-hatchet-signature'];
      const body = JSON.stringify(req.body);
      if (req.method === 'PUT') {
        const resp = await this.getHealthcheckResponse(body, sig, secret);
        return res.status(200).send(JSON.stringify(resp));
      }
      if (req.method !== 'POST') {
        return res.status(405).send('Method not allowed');
      }
      await this.handle(body, sig, secret);
      return res.status(200).send('ok');
    };
  }

  /**
   * A method that returns a Next.js request handler.
   */
  nextJSHandler({ secret }: HandlerOpts) {
    const ok = async () => {
      return new Response(okMessage, { status: 200 });
    };
    const f = async (req: Request) => {
      const sig = req.headers.get('x-hatchet-signature');
      const body = await req.text();
      if (req.method === 'PUT') {
        const resp = await this.getHealthcheckResponse(body, sig, secret);
        return new Response(JSON.stringify(resp), { status: 200 });
      }
      if (req.method !== 'POST') {
        return new Response('Method not allowed', { status: 405 });
      }
      await this.handle(body, sig, secret);
      return new Response('ok', { status: 200 });
    };
    return {
      GET: ok,
      POST: f,
      PUT: f,
    };
  }

  private getBody(req: IncomingMessage): Promise<string> {
    return new Promise((resolve) => {
      let body = '';
      req.on('data', (chunk) => {
        body += chunk;
      });
      req.on('end', () => {
        resolve(body);
      });
    });
  }
}
