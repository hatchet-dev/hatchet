/**
 * Example Hatchet serverless endpoint on Cloudflare Workers.
 *
 * Routes (served by @hatchet-dev/serverless/cloudflare):
 *   POST /hatchet/healthcheck   signed; answers the workflows this endpoint serves
 *   POST /hatchet/trigger       signed; runs one non-durable task and returns its output
 *   anything else               404
 *
 * The operator (pkg/serverlessoperator) is the only caller. It polls the healthcheck and POSTs
 * every assigned non-durable task to the trigger URL. The signing secret comes from the
 * HATCHET_SIGNING_SECRET Worker secret.
 */
import { cloudflare } from "@hatchet-dev/serverless/cloudflare";
import { workflows } from "./tasks";

export default cloudflare({ workflows });
