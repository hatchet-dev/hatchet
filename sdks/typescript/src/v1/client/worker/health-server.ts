import { createServer, Server, IncomingMessage, ServerResponse } from 'node:http';
import { Logger } from '@hatchet/util/logger';

export enum WorkerStatus {
  INITIALIZED = 'INITIALIZED',
  STARTING = 'STARTING',
  HEALTHY = 'HEALTHY',
  UNHEALTHY = 'UNHEALTHY',
}

interface HealthCheckResponse {
  status: string;
  name: string;
  slots: number;
  actions: string[];
  labels: Record<string, string | number>;
  nodeVersion: string;
}

export class HealthServer {
  private server: Server | null = null;
  private register: any = null;
  private workerStatusGauge: any = null;
  private workerSlotsGauge: any = null;
  private workerActionsGauge: any = null;
  private metricsInitialized: boolean = false;

  constructor(
    private port: number,
    private getStatus: () => WorkerStatus,
    private workerName: string,
    private getSlots: () => number,
    private getActions: () => string[],
    private getLabels: () => Record<string, string | number>,
    private logger: Logger
  ) {
    this.initializeMetrics();
  }

  private async handleRequest(req: IncomingMessage, res: ServerResponse): Promise<void> {
    const url = req.url || '';

    if (url === '/health' && req.method === 'GET') {
      await this.handleHealth(res);
    } else if (url === '/metrics' && req.method === 'GET') {
      await this.handleMetrics(res);
    } else {
      res.writeHead(404, { 'Content-Type': 'text/plain' });
      res.end('Not Found');
    }
  }

  private async handleHealth(res: ServerResponse): Promise<void> {
    const response: HealthCheckResponse = {
      status: this.getStatus(),
      name: this.workerName,
      slots: this.getSlots(),
      actions: this.getActions(),
      labels: this.getLabels(),
      nodeVersion: process.version,
    };

    res.writeHead(200, { 'Content-Type': 'application/json' });
    await res.end(JSON.stringify(response));
  }

  private initializeMetrics(): void {
    try {
      // @ts-ignore - prom-client is an optional dependency
      const { Registry, Gauge, collectDefaultMetrics } = require('prom-client');

      this.register = new Registry();
      collectDefaultMetrics({ register: this.register });

      this.workerStatusGauge = new Gauge({
        name: 'hatchet_worker_status',
        help: 'Current status of the Hatchet worker',
        registers: [this.register],
        collect: () => {
          this.workerStatusGauge!.set(this.getStatus() === WorkerStatus.HEALTHY ? 1 : 0);
        },
      });

      this.workerSlotsGauge = new Gauge({
        name: 'hatchet_worker_slots',
        help: 'Total slots available on the worker',
        registers: [this.register],
        collect: () => {
          this.workerSlotsGauge!.set(this.getSlots());
        },
      });

      this.workerActionsGauge = new Gauge({
        name: 'hatchet_worker_actions',
        help: 'Number of registered actions on the worker',
        registers: [this.register],
        collect: () => {
          this.workerActionsGauge!.set(this.getActions().length);
        },
      });
      this.metricsInitialized = true;
    } catch (error) {
      this.metricsInitialized = false;
      this.logger.error('Metrics initialization failed - prom-client dependency not installed');
    }
  }

  private async handleMetrics(res: ServerResponse): Promise<void> {
    if (!this.metricsInitialized || !this.register) {
      this.logger.error('Metrics initialization failed - prom-client dependency not installed');
      res.writeHead(503, { 'Content-Type': 'text/plain' });
      res.end('Metrics initialization failed');
      return;
    }

    try {
      const metrics = await this.register.metrics();
      res.writeHead(200, { 'Content-Type': this.register.contentType });
      res.end(metrics);
    } catch (error) {
      this.logger.error(`Error generating metrics: ${error}`);
      res.writeHead(500, { 'Content-Type': 'text/plain' });
      res.end('Error generating metrics');
    }
  }

  async start(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.server = createServer((req, res) => {
          this.handleRequest(req, res);
        });

        this.server.listen(this.port, '0.0.0.0', () => {
          this.logger.info(`Health check server running on port ${this.port}`);
          resolve();
        });

        this.server.on('error', (error) => {
          this.logger.error(`Failed to start health check server: ${error.message}`);
          reject(error);
        });
      } catch (error) {
        this.logger.error(`Failed to start health check server: ${error}`);
        reject(error);
      }
    });
  }

  async stop(): Promise<void> {
    if (this.server) {
      return new Promise((resolve) => {
        this.server!.close(() => {
          this.logger.info('Health check server stopped!');
          resolve();
        });
      });
    }
  }
}
