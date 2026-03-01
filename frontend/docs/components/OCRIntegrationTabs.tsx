import { Callout, Tabs } from "nextra/components";
import { snippets } from "@/lib/generated/snippets";
import { Snippet } from "@/components/code";
import PackageManagerInstall from "@/components/PackageManagerInstall";
import UniversalTabs from "@/components/UniversalTabs";

/** Nested tabs: Provider → Language. Wire into get_ocr_service().parse(). */
export function OCRIntegrationTabs() {
  return (
    <Tabs items={["Tesseract", "Unstructured", "Reducto", "Google Vision"]}>
      <Tabs.Tab title="Tesseract">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]}>
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "pytesseract" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.ocr_tesseract
                  .tesseract_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ typescript: "tesseract.js" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.ocr_tesseract
                  .tesseract_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ go: "github.com/otiai10/gosseract/v2" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.go.guides.integrations.ocr_tesseract.tesseract_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ ruby: "rtesseract" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.ruby.guides.integrations.ocr_tesseract.tesseract_usage
              }
            />
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Unstructured">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]} variant="hidden">
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ python: '"unstructured[pdf]"' }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.ocr_unstructured
                  .unstructured_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <Callout type="info">
              Unstructured is Python-only. Use Tesseract or Reducto API for
              Node.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">Unstructured is Python-only.</Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">Unstructured is Python-only.</Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Reducto">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]} variant="hidden">
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "reductoai" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.ocr_reducto.reducto_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <Callout type="info">
              Reducto: use <code>fetch</code> to{" "}
              <a href="https://docs.reducto.ai/" target="_blank" rel="noopener">
                platform.reducto.ai
              </a>{" "}
              API.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Reducto: use <code>net/http</code> to platform.reducto.ai. See{" "}
              <a href="https://docs.reducto.ai/" target="_blank" rel="noopener">
                docs
              </a>
              .
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Reducto: use HTTP client. See{" "}
              <a href="https://docs.reducto.ai/" target="_blank" rel="noopener">
                docs
              </a>
              .
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Google Vision">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]} variant="hidden">
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ python: "google-cloud-vision" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.ocr_google_vision
                  .google_vision_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ typescript: "@google-cloud/vision" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.ocr_google_vision
                  .google_vision_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Google Vision:{" "}
              <code>go get cloud.google.com/go/vision/apiv1</code>.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Google Vision: <code>bundle add google-cloud-vision</code>.
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
    </Tabs>
  );
}

export default OCRIntegrationTabs;
