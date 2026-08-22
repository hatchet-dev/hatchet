import fs from "node:fs";
import path from "node:path";
import yaml from "js-yaml";

export async function GET() {
  try {
    const filePath = path.join(process.cwd(), "lib", "management.openapi.yaml");
    const fileContents = fs.readFileSync(filePath, "utf8");
    const spec = yaml.load(fileContents) as object;

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

    return new Response(html, {
      headers: { "Content-Type": "text/html" },
    });
  } catch (error) {
    console.error("Error loading OpenAPI spec:", error);
    return Response.json(
      { error: "Failed to load OpenAPI spec" },
      { status: 500 },
    );
  }
}
