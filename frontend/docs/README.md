# Nextra Docs Template

This is a template for creating documentation with [Nextra](https://nextra.site).

[**Live Demo →**](https://nextra-docs-template.vercel.app)

[![](.github/screenshot.png)](https://nextra-docs-template.vercel.app)

## Quick Start

Click the button to clone this repository and deploy it on Vercel:

[![](https://vercel.com/button)](https://vercel.com/new/clone?s=https%3A%2F%2Fgithub.com%2Fshuding%2Fnextra-docs-template&showOptionalTeamCreation=false)

## Local Development

First, run `pnpm install` to install the dependencies.

Then, run `pnpm dev` to start the development server and visit localhost:3000.

## License

This project is licensed under the MIT License.


### Documentation Overhaul Checklist

- [ ] **User Guide** - `pages/home`
- [ ] **Managed Compute** - `pages/compute`
- [ ] **SDK Reference** - `pages/sdks`
- [ ] **Self Hosting** - `pages/self-hosting`
- [ ] **Blog** - `pages/blog`
- [ ] **Why we moved off Prisma** - `pages/blog/migrating-off-prisma`
- [ ] **The problems with Celery** - `pages/blog/problems-with-celery`
- [ ] **An unfair advantage: multi-tenant queues in Postgres** - `pages/blog/multi-tenant-queues`
- [ ] **Managed Compute** - `pages/compute/-- Managed Compute`
- [ ] **Overview** - `pages/compute/index`
- [ ] **Getting Started** - `pages/compute/getting-started`
- [ ] **CPU Machine Types** - `pages/compute/cpu`
- [ ] **GPU Machine Types** - `pages/compute/gpu`
- [ ] **SDK Deployment Guides** - `pages/compute/-- SDKs`
- [ ] **Python ↗** - `pages/compute/python`
- [ ] **TypeScript ↗** - `pages/compute/typescript`
- [ ] **Golang ↗** - `pages/compute/golang`
- [ ] **Contributing** - `pages/contributing/index`
- [ ] **GitHub App Setup** - `pages/contributing/github-app-setup`
- [ ] **SDKs** - `pages/contributing/sdks`
- [ ] **Introduction** - `pages/home/index`
- [ ] **Hatchet Cloud Quickstart** - `pages/home/hatchet-cloud-quickstart`
- [ ] **Guide** - `pages/home/--guide`
- [ ] **Working With Hatchet** - `pages/home/basics`
- [ ] **Tutorials** - `pages/home/tutorials`
- [ ] **Features** - `pages/home/features`
- [ ] **More** - `pages/home/--more`
- [ ] **About Hatchet ↗** - `pages/home/about`
- [ ] **Overview** - `pages/home/basics/overview`
- [ ] **Understanding Steps** - `pages/home/basics/steps`
- [ ] **Understanding Workflows** - `pages/home/basics/workflows`
- [ ] **Understanding Workers** - `pages/home/basics/workers`
- [ ] **Dashboard** - `pages/home/basics/dashboard`
- [ ] **Managing Environments** - `pages/home/basics/environments`
- [ ] **Concurrency Strategies** - `pages/home/features/concurrency`
- [ ] **Durable Execution** - `pages/home/features/durable-execution`
- [ ] **Retries** - `pages/home/features/retries`
- [ ] **Timeouts** - `pages/home/features/timeouts`
- [ ] **Errors and Logging** - `pages/home/features/errors-and-logging`
- [x] **On Failure Step** - `pages/home/features/on-failure-step`
- [ ] **Streaming** - `pages/home/features/streaming`
- [ ] **Triggering Runs** - `pages/home/features/triggering-runs`
- [ ] **Rate Limits** - `pages/home/features/rate-limits`
- [ ] **Worker Assignment** - `pages/home/features/worker-assignment`
- [ ] **Additional Metadata** - `pages/home/features/additional-metadata`
- [ ] **Advanced** - `pages/home/features/advanced`
- [ ] **Manual Slot Release** - `pages/home/features/advanced/manual-slot-release`
- [ ] **Overview** - `pages/home/features/concurrency/overview`
- [ ] **Cancel In Progress** - `pages/home/features/concurrency/cancel-in-progress`
- [ ] **Round Robin** - `pages/home/features/concurrency/round-robin`
- [ ] **Overview** - `pages/home/features/retries/overview`
- [ ] **Simple Auto Retry** - `pages/home/features/retries/simple`
- [ ] **Manual Retries** - `pages/home/features/retries/manual`
- [ ] **Event Trigger** - `pages/home/features/triggering-runs/event-trigger`
- [ ] **Cron Scheduling** - `pages/home/features/triggering-runs/cron-trigger`
- [ ] **Schedule Trigger** - `pages/home/features/triggering-runs/schedule-trigger`
- [ ] **Overview** - `pages/home/features/worker-assignment/overview`
- [ ] **Sticky Assignment** - `pages/home/features/worker-assignment/sticky-assignment`
- [ ] **Worker Affinity** - `pages/home/features/worker-assignment/worker-affinity`
- [ ] **Fullstack - FastAPI/React** - `pages/home/tutorials/fastapi-react`
- [ ] **Project Setup** - `pages/home/tutorials/fastapi-react/project-setup`
- [ ] **Building the Workflow** - `pages/home/tutorials/fastapi-react/building-the-workflow`
- [ ] **API Server Setup** - `pages/home/tutorials/fastapi-react/api-server-setup`
- [ ] **Server Result Streaming** - `pages/home/tutorials/fastapi-react/result-streaming`
- [ ] **Simple Frontend** - `pages/home/tutorials/fastapi-react/simple-frontend`
- [ ] **Go SDK** - `pages/sdks/go-sdk/Go SDK`
- [ ] **Introduction** - `pages/sdks/go-sdk/index`
- [ ] **Quickstart ↗** - `pages/sdks/go-sdk/quickstart`
- [ ] **Creating a Workflow** - `pages/sdks/go-sdk/creating-a-workflow`
- [ ] **Creating a Worker** - `pages/sdks/go-sdk/creating-a-worker`
- [ ] **Pushing Events** - `pages/sdks/go-sdk/pushing-events`
- [ ] **Scheduling Workflows** - `pages/sdks/go-sdk/scheduling-workflows`
- [ ] **Python SDK** - `pages/sdks/python-sdk/Python SDK`
- [ ] **Introduction** - `pages/sdks/python-sdk/index`
- [ ] **Quickstart ↗** - `pages/sdks/python-sdk/quickstart`
- [ ] **Configuration** - `pages/sdks/python-sdk/--- Configuration`
- [ ] **Client** - `pages/sdks/python-sdk/client`
- [ ] **Worker** - `pages/sdks/python-sdk/worker`
- [ ] **Workflow** - `pages/sdks/python-sdk/workflow`
- [ ] **Running Workflows** - `pages/sdks/python-sdk/--- Running Workflows`
- [ ] **API-Triggered Workflows** - `pages/sdks/python-sdk/run-workflow-api`
- [ ] **Child Workflows** - `pages/sdks/python-sdk/run-workflow-child`
- [ ] **Event-Triggered Workflows** - `pages/sdks/python-sdk/run-workflow-events`
- [ ] **Cron Workflows** - `pages/sdks/python-sdk/run-workflow-cron`
- [ ] **Scheduled Workflows** - `pages/sdks/python-sdk/run-workflow-schedule`
- [ ] **Deploying Workers** - `pages/sdks/python-sdk/--- Deploying Workers`
- [ ] **Docker** - `pages/sdks/python-sdk/docker`
- [ ] **Managed Compute** - `pages/sdks/python-sdk/managed-compute`
- [ ] **Self-Hosted** - `pages/sdks/python-sdk/self-hosted`
- [ ] **Getting Workflow Results** - `pages/sdks/python-sdk/--- Getting Workflow Results`
- [ ] **Getting Workflow Run Results** - `pages/sdks/python-sdk/get-workflow-results`
- [ ] **Advanced** - `pages/sdks/python-sdk/--- Advanced`
- [ ] **Concurrency and Fairness** - `pages/sdks/python-sdk/fairness`
- [ ] **Logging** - `pages/sdks/python-sdk/logging`
- [ ] **REST API** - `pages/sdks/python-sdk/api`
- [ ] **Working with AsyncIO** - `pages/sdks/python-sdk/asyncio`
- [ ] **Admin** - `pages/sdks/typescript-sdk/_api/admin-client`
- [ ] **Typescript SDK** - `pages/sdks/typescript-sdk/Typescript SDK`
- [ ] **Introduction** - `pages/sdks/typescript-sdk/index`
- [ ] **Quickstart ↗** - `pages/sdks/typescript-sdk/quickstart`
- [ ] **Configuration** - `pages/sdks/typescript-sdk/--- Configuration`
- [ ] **Client** - `pages/sdks/typescript-sdk/client`
- [ ] **Worker** - `pages/sdks/typescript-sdk/worker`
- [ ] **Workflow** - `pages/sdks/typescript-sdk/workflow`
- [ ] **Running Workflows** - `pages/sdks/typescript-sdk/--- Running Workflows`
- [ ] **API-Triggered Workflows** - `pages/sdks/typescript-sdk/run-workflow-api`
- [ ] **Child Workflows** - `pages/sdks/typescript-sdk/run-workflow-child`
- [ ] **Event-Triggered Workflows** - `pages/sdks/typescript-sdk/run-workflow-events`
- [ ] **Cron Workflows** - `pages/sdks/typescript-sdk/run-workflow-cron`
- [ ] **Scheduled Workflows** - `pages/sdks/typescript-sdk/run-workflow-schedule`
- [ ] **Deploying Workers** - `pages/sdks/typescript-sdk/--- Deploying Workers`
- [ ] **Docker** - `pages/sdks/typescript-sdk/docker`
- [ ] **Managed Compute** - `pages/sdks/typescript-sdk/managed-compute`
- [ ] **Self-Hosted** - `pages/sdks/typescript-sdk/self-hosted`
- [ ] **Getting Workflow Results** - `pages/sdks/typescript-sdk/--- Getting Workflow Results`
- [ ] **Getting Workflow Run Results** - `pages/sdks/typescript-sdk/get-workflow-results`
- [ ] **Advanced** - `pages/sdks/typescript-sdk/--- Advanced`
- [ ] **Concurrency and Fairness** - `pages/sdks/typescript-sdk/fairness`
- [ ] **Logging** - `pages/sdks/typescript-sdk/logging`
- [ ] **Introduction** - `pages/self-hosting/index`
- [ ] **Docker** - `pages/self-hosting/-- Docker`
- [ ] **Hatchet Lite** - `pages/self-hosting/hatchet-lite`
- [ ] **Docker Compose** - `pages/self-hosting/docker-compose`
- [ ] **Kubernetes** - `pages/self-hosting/-- Kubernetes`
- [ ] **Quickstart** - `pages/self-hosting/kubernetes-quickstart`
- [ ] **Installing with Glasskube** - `pages/self-hosting/kubernetes-glasskube`
- [ ] **Networking** - `pages/self-hosting/networking`
- [ ] **Setting up an External Database** - `pages/self-hosting/kubernetes-external-database`
- [ ] **Managing Hatchet** - `pages/self-hosting/-- Managing Hatchet`
- [ ] **Configuration Options** - `pages/self-hosting/configuration-options`
- [ ] **Data Retention** - `pages/self-hosting/data-retention`
- [ ] **Improving Performance** - `pages/self-hosting/improving-performance`
