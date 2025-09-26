import type { NextApiRequest, NextApiResponse } from "next";
import fs from "fs";
import path from "path";
import yaml from "js-yaml";

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse,
) {
  if (req.method !== "GET") {
    return res.status(405).json({ error: "Method not allowed" });
  }

  try {
    const filePath = path.join(process.cwd(), "lib", "management.openapi.yaml");
    const fileContents = fs.readFileSync(filePath, "utf8");
    const spec = yaml.load(fileContents) as any;

    const html = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="SwaggerUI" />
    <title>Hatchet Cloud Management API</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@4.18.2/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.18.2/swagger-ui-bundle.js" crossorigin></script>
    <script>
      window.onload = () => {
        window.ui = SwaggerUIBundle({
          spec: ${JSON.stringify(spec)},
          dom_id: '#swagger-ui',
          deepLinking: true,
          presets: [
            SwaggerUIBundle.presets.apis,
            SwaggerUIBundle.presets.standalone,
          ],
          plugins: [
            SwaggerUIBundle.plugins.DownloadUrl
          ]
        });
      };
    </script>
  </body>
</html>`;

    res.setHeader("Content-Type", "text/html");
    return res.status(200).send(html);
  } catch (error) {
    console.error("Error loading OpenAPI spec:", error);
    return res.status(500).json({ error: "Failed to load OpenAPI spec" });
  }
}
