import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from './hatchet-client';\n\nfunction processImage(\n  imageUrl: string,\n  filters: string[]\n): Promise<{ url: string; size: number; format: string }> {\n  // Do some image processing\n  return Promise.resolve({ url: imageUrl, size: 100, format: 'png' });\n}\n// > Before (Mergent)\nexport async function processImageTask(req: { body: { imageUrl: string; filters: string[] } }) {\n  const { imageUrl, filters } = req.body;\n  try {\n    const result = await processImage(imageUrl, filters);\n    return { success: true, processedUrl: result.url };\n  } catch (error) {\n    console.error('Image processing failed:', error);\n    throw error;\n  }\n}\n\n// > After (Hatchet)\ntype ImageProcessInput = {\n  imageUrl: string;\n  filters: string[];\n};\n\ntype ImageProcessOutput = {\n  processedUrl: string;\n  metadata: {\n    size: number;\n    format: string;\n    appliedFilters: string[];\n  };\n};\n\nexport const imageProcessor = hatchet.task({\n  name: 'image-processor',\n  retries: 3,\n  executionTimeout: '10m',\n  fn: async ({ imageUrl, filters }: ImageProcessInput): Promise<ImageProcessOutput> => {\n    // Do some image processing\n    const result = await processImage(imageUrl, filters);\n\n    if (!result.url) throw new Error('Processing failed to generate URL');\n\n    return {\n      processedUrl: result.url,\n      metadata: {\n        size: result.size,\n        format: result.format,\n        appliedFilters: filters,\n      },\n    };\n  },\n});\n\nasync function run() {\n  // > Running a task (Mergent)\n  const options = {\n    method: 'POST',\n    headers: { Authorization: 'Bearer <token>', 'Content-Type': 'application/json' },\n    body: JSON.stringify({\n      name: '4cf95241-fa19-47ef-8a67-71e483747649',\n      queue: 'default',\n      request: {\n        url: 'https://example.com',\n        headers: { Authorization: 'fake-secret-token', 'Content-Type': 'application/json' },\n        body: 'Hello, world!',\n      },\n    }),\n  };\n\n  fetch('https://api.mergent.co/v2/tasks', options)\n    .then((response) => response.json())\n    .then((response) => console.log(response))\n    .catch((err) => console.error(err));\n\n  // > Running a task (Hatchet)\n  const result = await imageProcessor.run({\n    imageUrl: 'https://example.com/image.png',\n    filters: ['blur'],\n  });\n\n  // you can await fully typed results\n  console.log(result);\n}\n\nasync function schedule() {\n  // > Scheduling tasks (Mergent)\n  const options = {\n    // same options as before\n    body: JSON.stringify({\n      // same body as before\n      delay: '5m',\n    }),\n  };\n\n  // > Scheduling tasks (Hatchet)\n  // Schedule the task to run at a specific time\n  const runAt = new Date(Date.now() + 1000 * 60 * 60 * 24);\n  imageProcessor.schedule(runAt, {\n    imageUrl: 'https://example.com/image.png',\n    filters: ['blur'],\n  });\n\n  // Schedule the task to run every hour\n  imageProcessor.cron('run-hourly', '0 * * * *', {\n    imageUrl: 'https://example.com/image.png',\n    filters: ['blur'],\n  });\n}\n",
  source: 'out/typescript/migration-guides/mergent.ts',
  blocks: {
    before_mergent: {
      start: 11,
      stop: 20,
    },
    after_hatchet: {
      start: 23,
      stop: 56,
    },
    running_a_task_mergent: {
      start: 60,
      stop: 77,
    },
    running_a_task_hatchet: {
      start: 80,
      stop: 86,
    },
    scheduling_tasks_mergent: {
      start: 91,
      stop: 97,
    },
    scheduling_tasks_hatchet: {
      start: 100,
      stop: 111,
    },
  },
  highlights: {},
};

export default snippet;
