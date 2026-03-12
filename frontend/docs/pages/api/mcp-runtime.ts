/**
 * MCP (Model Context Protocol) server for Hatchet runtime monitoring.
 *
 * Exposes live system data (runs, workers, workflows, queues) to AI agents.
 * Requires Bearer token authentication; proxies requests to the Hatchet REST API.
 *
 * Endpoint: POST /api/mcp-runtime   (JSON-RPC 2.0)
 *           GET  /api/mcp-runtime   (returns server metadata)
 */
import type { NextApiRequest, NextApiResponse } from "next";
import { PostHog } from "posthog-node";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface JsonRpcRequest {
  jsonrpc: "2.0";
  id?: string | number | null;
  method: string;
  params?: Record<string, unknown>;
}

interface JsonRpcResponse {
  jsonrpc: "2.0";
  id: string | number | null;
  result?: unknown;
  error?: { code: number; message: string; data?: unknown };
}

/**
 * Extracts tenant ID from Hatchet API token JWT.
 * The API token JWT uses the `sub` claim as the tenant ID (see pkg/auth/token/token.go).
 */
function getTenantIdFromToken(token: string): string | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    let payload = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const pad = payload.length % 4;
    if (pad) payload += "=".repeat(4 - pad);
    const decoded = Buffer.from(payload, "base64").toString("utf-8");
    const parsed = JSON.parse(decoded) as { sub?: string };
    const sub = parsed.sub;
    if (!sub || typeof sub !== "string") return null;
    return sub;
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------
const PROTOCOL_VERSION = "2024-11-05";
const SERVER_NAME = "hatchet-runtime";
const SERVER_VERSION = "1.0.0";

const ALLOW_WRITES =
  process.env.HATCHET_MCP_ALLOW_WRITES?.toLowerCase() === "true";

// ---------------------------------------------------------------------------
// PostHog server-side analytics
// ---------------------------------------------------------------------------
let posthogClient: PostHog | null = null;

function getPostHog(): PostHog | null {
  if (posthogClient) return posthogClient;
  const key = process.env.NEXT_PUBLIC_POSTHOG_KEY;
  if (!key) return null;
  posthogClient = new PostHog(key, {
    host: process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://us.i.posthog.com",
    flushAt: 10,
    flushInterval: 5000,
  });
  return posthogClient;
}

function trackMcpEvent(
  req: NextApiRequest,
  method: string,
  properties?: Record<string, unknown>,
): void {
  const ph = getPostHog();
  if (!ph) return;
  const sessionId = (req.headers["mcp-session-id"] as string) || "anonymous";
  ph.capture({
    distinctId: `mcp-runtime:${sessionId}`,
    event: "mcp_runtime_request",
    properties: {
      method,
      user_agent: req.headers["user-agent"] || "",
      ...properties,
    },
  });
}

// ---------------------------------------------------------------------------
// Auth and API client
// ---------------------------------------------------------------------------
function getApiBaseUrl(): string | null {
  const url = process.env.HATCHET_CLIENT_API_URL;
  if (!url) return null;
  return url.replace(/\/$/, "");
}

function getBearerToken(req: NextApiRequest): string | null {
  const auth = req.headers.authorization;
  if (!auth || !auth.startsWith("Bearer ")) return null;
  return auth.slice(7).trim();
}

function resolveTenantId(token: string): string | null {
  return getTenantIdFromToken(token);
}

async function apiFetch(
  url: string,
  token: string,
  init?: RequestInit,
): Promise<Response> {
  return fetch(url, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });
}

// ---------------------------------------------------------------------------
// Tool handlers
// ---------------------------------------------------------------------------
async function handleListWorkflows(
  baseUrl: string,
  tenantId: string,
  token: string,
): Promise<unknown> {
  const res = await apiFetch(
    `${baseUrl}/api/v1/tenants/${tenantId}/workflows`,
    token,
  );
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API error ${res.status}: ${err}`);
  }
  const data = (await res.json()) as { rows?: Array<{ metadata: { id: string }; name: string; description?: string; version?: string }> };
  return (data.rows ?? []).map((w) => ({
    id: w.metadata?.id,
    name: w.name,
    description: w.description ?? "",
    version: w.version ?? "",
  }));
}

async function handleListRuns(
  baseUrl: string,
  tenantId: string,
  token: string,
  args: Record<string, unknown>,
): Promise<unknown> {
  const workflowName = args.workflow_name as string | undefined;
  const status = args.status as string | undefined;
  const sinceHours = (args.since_hours as number) ?? 24;
  const limit = (args.limit as number) ?? 50;

  const since = new Date();
  since.setHours(since.getHours() - sinceHours);
  const params = new URLSearchParams();
  params.set("since", since.toISOString());
  params.set("limit", String(limit));
  params.set("only_tasks", "false");
  if (status) params.set("statuses", status);

  if (workflowName) {
    const workflowsRes = await apiFetch(
      `${baseUrl}/api/v1/tenants/${tenantId}/workflows`,
      token,
    );
    if (workflowsRes.ok) {
      const workflowsData = (await workflowsRes.json()) as {
        rows?: Array<{ metadata: { id: string }; name: string }>;
      };
      const wf = workflowsData.rows?.find(
        (w) => w.name?.toLowerCase() === workflowName.toLowerCase(),
      );
      if (wf) params.set("workflow_ids", wf.metadata.id);
    }
  }

  const url = `${baseUrl}/api/v1/stable/tenants/${tenantId}/workflow-runs?${params}`;
  const res = await apiFetch(url, token);
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API error ${res.status}: ${err}`);
  }
  const data = (await res.json()) as { rows?: unknown[] };
  return data.rows ?? [];
}

async function handleGetRun(
  baseUrl: string,
  token: string,
  args: Record<string, unknown>,
): Promise<unknown> {
  const runId = args.run_id as string | undefined;
  if (!runId) throw new Error("Missing required argument: run_id");

  const res = await apiFetch(
    `${baseUrl}/api/v1/stable/workflow-runs/${runId}`,
    token,
  );
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API error ${res.status}: ${err}`);
  }
  return res.json();
}

async function handleSearchRuns(
  baseUrl: string,
  tenantId: string,
  token: string,
  args: Record<string, unknown>,
): Promise<unknown> {
  const metadataKey = args.metadata_key as string | undefined;
  const metadataValue = args.metadata_value as string | undefined;
  const status = args.status as string | undefined;
  const sinceHours = (args.since_hours as number) ?? 24;

  if (!metadataKey || !metadataValue) {
    throw new Error("Missing required arguments: metadata_key, metadata_value");
  }

  const since = new Date();
  since.setHours(since.getHours() - sinceHours);
  const params = new URLSearchParams();
  params.set("since", since.toISOString());
  params.set("only_tasks", "false");
  params.set("additional_metadata", `${metadataKey}:${metadataValue}`);
  if (status) params.set("statuses", status);

  const url = `${baseUrl}/api/v1/stable/tenants/${tenantId}/workflow-runs?${params}`;
  const res = await apiFetch(url, token);
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API error ${res.status}: ${err}`);
  }
  const data = (await res.json()) as { rows?: unknown[] };
  return data.rows ?? [];
}

async function handleGetQueueMetrics(
  baseUrl: string,
  tenantId: string,
  token: string,
  args: Record<string, unknown>,
): Promise<unknown> {
  const workflowName = args.workflow_name as string | undefined;

  const since = new Date();
  since.setHours(since.getHours() - 24);
  const params = new URLSearchParams();
  params.set("since", since.toISOString());
  params.set("limit", "1000");
  params.set("only_tasks", "false");

  if (workflowName) {
    const workflowsRes = await apiFetch(
      `${baseUrl}/api/v1/tenants/${tenantId}/workflows`,
      token,
    );
    if (workflowsRes.ok) {
      const workflowsData = (await workflowsRes.json()) as {
        rows?: Array<{ metadata: { id: string }; name: string }>;
      };
      const wf = workflowsData.rows?.find(
        (w) => w.name?.toLowerCase() === workflowName.toLowerCase(),
      );
      if (wf) params.set("workflow_ids", wf.metadata.id);
    }
  }

  const url = `${baseUrl}/api/v1/stable/tenants/${tenantId}/workflow-runs?${params}`;
  const res = await apiFetch(url, token);
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API error ${res.status}: ${err}`);
  }
  const data = (await res.json()) as {
    rows?: Array<{ status?: string }>;
  };
  const rows = data.rows ?? [];
  const counts = {
    queued: 0,
    running: 0,
    completed: 0,
    failed: 0,
    cancelled: 0,
    total: rows.length,
  };
  for (const r of rows) {
    const s = (r.status ?? "").toLowerCase();
    if (s === "queued") counts.queued++;
    else if (s === "running") counts.running++;
    else if (s === "completed" || s === "succeeded") counts.completed++;
    else if (s === "failed") counts.failed++;
    else if (s === "cancelled") counts.cancelled++;
  }
  return counts;
}

async function handleListWorkers(
  baseUrl: string,
  tenantId: string,
  token: string,
): Promise<unknown> {
  const res = await apiFetch(
    `${baseUrl}/api/v1/tenants/${tenantId}/worker`,
    token,
  );
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API error ${res.status}: ${err}`);
  }
  const data = (await res.json()) as {
    rows?: Array<{
      metadata: { id: string };
      name?: string;
      status?: string;
      lastHeartbeatAt?: string;
      slots?: number;
      workflows?: Array<{ name?: string }>;
    }>;
  };
  return (data.rows ?? []).map((w) => ({
    id: w.metadata?.id,
    name: w.name ?? "",
    status: w.status ?? "",
    lastHeartbeatAt: w.lastHeartbeatAt ?? null,
    slots: w.slots ?? 0,
    workflows: (w.workflows ?? []).map((wf) => wf.name ?? ""),
  }));
}

async function handleCancelRun(
  baseUrl: string,
  tenantId: string,
  token: string,
  args: Record<string, unknown>,
): Promise<unknown> {
  const runId = args.run_id as string | undefined;
  if (!runId) throw new Error("Missing required argument: run_id");

  const res = await apiFetch(
    `${baseUrl}/api/v1/stable/tenants/${tenantId}/tasks/cancel`,
    token,
    {
      method: "POST",
      body: JSON.stringify({ externalIds: [runId] }),
    },
  );
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API error ${res.status}: ${err}`);
  }
  const data = (await res.json()) as { cancelled?: unknown[] };
  return { success: (data.cancelled?.length ?? 0) > 0 };
}

async function handleReplayRun(
  baseUrl: string,
  tenantId: string,
  token: string,
  args: Record<string, unknown>,
): Promise<unknown> {
  const runId = args.run_id as string | undefined;
  if (!runId) throw new Error("Missing required argument: run_id");

  const res = await apiFetch(
    `${baseUrl}/api/v1/stable/tenants/${tenantId}/tasks/replay`,
    token,
    {
      method: "POST",
      body: JSON.stringify({ externalIds: [runId] }),
    },
  );
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API error ${res.status}: ${err}`);
  }
  const data = (await res.json()) as { replayed?: Array<{ metadata?: { id: string } }> };
  const first = data.replayed?.[0];
  return { new_run_id: first?.metadata?.id ?? null };
}

// ---------------------------------------------------------------------------
// MCP method handlers
// ---------------------------------------------------------------------------
function handleInitialize(id: string | number | null): JsonRpcResponse {
  return {
    jsonrpc: "2.0",
    id,
    result: {
      protocolVersion: PROTOCOL_VERSION,
      capabilities: {
        tools: {},
      },
      serverInfo: {
        name: SERVER_NAME,
        version: SERVER_VERSION,
      },
    },
  };
}

function getRuntimeTools(): Array<{
  name: string;
  description: string;
  inputSchema: { type: string; properties: Record<string, unknown>; required?: string[] };
}> {
  const tools: Array<{
    name: string;
    description: string;
    inputSchema: { type: string; properties: Record<string, unknown>; required?: string[] };
  }> = [
    {
      name: "list_workflows",
      description: "List all registered workflows for the tenant.",
      inputSchema: { type: "object", properties: {} },
    },
    {
      name: "list_runs",
      description:
        "List workflow runs with optional filters (workflow_name, status, since_hours, limit).",
      inputSchema: {
        type: "object",
        properties: {
          workflow_name: { type: "string", description: "Filter by workflow name" },
          status: {
            type: "string",
            description: "Filter by status: queued, running, completed, failed, cancelled",
          },
          since_hours: { type: "number", description: "Hours to look back (default: 24)" },
          limit: { type: "number", description: "Max results (default: 50)" },
        },
      },
    },
    {
      name: "get_run",
      description: "Get full details of a specific run by ID.",
      inputSchema: {
        type: "object",
        properties: { run_id: { type: "string", description: "Workflow run ID (UUID)" } },
        required: ["run_id"],
      },
    },
    {
      name: "search_runs",
      description:
        "Search runs by additional_metadata key-value pairs (e.g. audit_id=abc123).",
      inputSchema: {
        type: "object",
        properties: {
          metadata_key: { type: "string", description: "Metadata key (e.g. audit_id)" },
          metadata_value: { type: "string", description: "Metadata value" },
          status: { type: "string", description: "Optional status filter" },
          since_hours: { type: "number", description: "Hours to look back (default: 24)" },
        },
        required: ["metadata_key", "metadata_value"],
      },
    },
    {
      name: "get_queue_metrics",
      description:
        "Get job counts by status for the last 24h (queued, running, completed, failed, cancelled).",
      inputSchema: {
        type: "object",
        properties: {
          workflow_name: { type: "string", description: "Optional filter by workflow name" },
        },
      },
    },
    {
      name: "list_workers",
      description:
        "List all active workers and their registered workflows.",
      inputSchema: { type: "object", properties: {} },
    },
    {
      name: "cancel_run",
      description: `Cancel a specific run by ID. (Write operation${ALLOW_WRITES ? "" : " — disabled, set HATCHET_MCP_ALLOW_WRITES=true to enable"})`,
      inputSchema: {
        type: "object",
        properties: { run_id: { type: "string", description: "Workflow run ID (UUID)" } },
        required: ["run_id"],
      },
    },
    {
      name: "replay_run",
      description: `Replay a failed or cancelled run. (Write operation${ALLOW_WRITES ? "" : " — disabled, set HATCHET_MCP_ALLOW_WRITES=true to enable"})`,
      inputSchema: {
        type: "object",
        properties: { run_id: { type: "string", description: "Workflow run ID (UUID)" } },
        required: ["run_id"],
      },
    },
  ];

  return tools;
}

function handleToolsList(id: string | number | null): JsonRpcResponse {
  return {
    jsonrpc: "2.0",
    id,
    result: { tools: getRuntimeTools() },
  };
}

async function handleToolsCall(
  id: string | number | null,
  params: Record<string, unknown>,
  baseUrl: string,
  tenantId: string,
  token: string,
): Promise<JsonRpcResponse> {
  const toolName = params.name as string | undefined;
  const args = (params.arguments || {}) as Record<string, unknown>;

  try {
    let result: unknown;
    switch (toolName) {
      case "list_workflows":
        result = await handleListWorkflows(baseUrl, tenantId, token);
        break;
      case "list_runs":
        result = await handleListRuns(baseUrl, tenantId, token, args);
        break;
      case "get_run":
        result = await handleGetRun(baseUrl, token, args);
        break;
      case "search_runs":
        result = await handleSearchRuns(baseUrl, tenantId, token, args);
        break;
      case "get_queue_metrics":
        result = await handleGetQueueMetrics(baseUrl, tenantId, token, args);
        break;
      case "list_workers":
        result = await handleListWorkers(baseUrl, tenantId, token);
        break;
      case "cancel_run":
        if (!ALLOW_WRITES) {
          return {
            jsonrpc: "2.0",
            id,
            error: {
              code: -32600,
              message:
                "cancel_run is disabled. Set HATCHET_MCP_ALLOW_WRITES=true to enable write tools.",
            },
          };
        }
        result = await handleCancelRun(baseUrl, tenantId, token, args);
        break;
      case "replay_run":
        if (!ALLOW_WRITES) {
          return {
            jsonrpc: "2.0",
            id,
            error: {
              code: -32600,
              message:
                "replay_run is disabled. Set HATCHET_MCP_ALLOW_WRITES=true to enable write tools.",
            },
          };
        }
        result = await handleReplayRun(baseUrl, tenantId, token, args);
        break;
      default:
        return {
          jsonrpc: "2.0",
          id,
          error: { code: -32602, message: `Unknown tool: ${toolName}` },
        };
    }

    return {
      jsonrpc: "2.0",
      id,
      result: {
        content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }],
      },
    };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return {
      jsonrpc: "2.0",
      id,
      error: { code: -32603, message },
    };
  }
}

// ---------------------------------------------------------------------------
// Notifications (no response needed)
// ---------------------------------------------------------------------------
const NOTIFICATION_METHODS = new Set([
  "notifications/initialized",
  "notifications/cancelled",
  "notifications/progress",
]);

// ---------------------------------------------------------------------------
// Route JSON-RPC request to handler
// ---------------------------------------------------------------------------
async function routeRequest(
  rpcReq: JsonRpcRequest,
  httpReq: NextApiRequest,
): Promise<JsonRpcResponse | null> {
  const { id, method, params } = rpcReq;

  if (id === undefined || id === null) {
    if (NOTIFICATION_METHODS.has(method)) return null;
    return null;
  }

  const baseUrl = getApiBaseUrl();
  if (!baseUrl) {
    return {
      jsonrpc: "2.0",
      id,
      error: {
        code: -32603,
        message: "HATCHET_CLIENT_API_URL is not configured",
      },
    };
  }

  const token = getBearerToken(httpReq);
  if (!token) {
    return {
      jsonrpc: "2.0",
      id,
      error: {
        code: -32600,
        message: "Missing Authorization: Bearer <token> header",
      },
    };
  }

  if (method === "initialize") {
    return handleInitialize(id);
  }

  if (method === "tools/list") {
    return handleToolsList(id);
  }

  if (method === "tools/call") {
    const tenantId = resolveTenantId(token!);
    if (!tenantId) {
      return {
        jsonrpc: "2.0",
        id,
        error: {
          code: -32600,
          message: "Could not resolve tenant from token",
        },
      };
    }

    trackMcpEvent(httpReq, method, {
      tool: params?.name,
    });

    return handleToolsCall(
      id,
      params || {},
      baseUrl,
      tenantId,
      token!,
    );
  }

  if (method === "ping") {
    return { jsonrpc: "2.0", id, result: {} };
  }

  return {
    jsonrpc: "2.0",
    id,
    error: { code: -32601, message: `Method not found: ${method}` },
  };
}

// ---------------------------------------------------------------------------
// Next.js API handler
// ---------------------------------------------------------------------------

export const config = {
  api: { responseLimit: false },
};

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse,
): Promise<void> {
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS");
  res.setHeader(
    "Access-Control-Allow-Headers",
    "Content-Type, Accept, Mcp-Session-Id, Authorization",
  );
  res.setHeader("Access-Control-Expose-Headers", "Mcp-Session-Id");

  if (req.method === "OPTIONS") {
    res.status(204).end();
    return;
  }

  if (req.method === "GET") {
    const accept = (req.headers.accept || "").toLowerCase();

    if (accept.includes("text/event-stream")) {
      res.writeHead(200, {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache, no-transform",
        Connection: "keep-alive",
      });
      res.write(": connected\n\n");
      const keepAlive = setInterval(() => {
        res.write(": ping\n\n");
      }, 15_000);
      req.on("close", () => {
        clearInterval(keepAlive);
        res.end();
      });
      return;
    }

    res.status(200).json({
      name: SERVER_NAME,
      version: SERVER_VERSION,
      protocolVersion: PROTOCOL_VERSION,
      description:
        "MCP server for Hatchet runtime monitoring. Requires Bearer token. Send JSON-RPC 2.0 POST requests.",
    });
    return;
  }

  if (req.method === "DELETE") {
    res.status(200).end();
    return;
  }

  if (req.method !== "POST") {
    res.status(405).json({ error: "Method not allowed" });
    return;
  }

  const body = req.body;

  if (Array.isArray(body)) {
    const responses: JsonRpcResponse[] = [];
    for (const item of body) {
      const result = await routeRequest(item as JsonRpcRequest, req);
      if (result) responses.push(result);
    }
    if (responses.length === 0) {
      res.status(204).end();
    } else {
      res.status(200).json(responses);
    }
    return;
  }

  const result = await routeRequest(body as JsonRpcRequest, req);
  if (!result) {
    res.status(204).end();
    return;
  }

  res.status(200).json(result);
}
