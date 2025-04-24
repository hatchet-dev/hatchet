import { hatchet } from './hatchet-client';

function processImage(
  imageUrl: string,
  filters: string[]
): Promise<{ url: string; size: number; format: string }> {
  // Do some image processing
  return Promise.resolve({ url: imageUrl, size: 100, format: 'png' });
}
// ❓ Before (Mergent)
export async function processImageTask(req: { body: { imageUrl: string; filters: string[] } }) {
  const { imageUrl, filters } = req.body;
  try {
    const result = await processImage(imageUrl, filters);
    return { success: true, processedUrl: result.url };
  } catch (error) {
    console.error('Image processing failed:', error);
    throw error;
  }
}
// !!

// ❓ After (Hatchet)
type ImageProcessInput = {
  imageUrl: string;
  filters: string[];
};

type ImageProcessOutput = {
  processedUrl: string;
  metadata: {
    size: number;
    format: string;
    appliedFilters: string[];
  };
};

export const imageProcessor = hatchet.task({
  name: 'image-processor',
  retries: 3,
  executionTimeout: '10m',
  fn: async ({ imageUrl, filters }: ImageProcessInput): Promise<ImageProcessOutput> => {
    // Do some image processing
    const result = await processImage(imageUrl, filters);

    if (!result.url) throw new Error('Processing failed to generate URL');

    return {
      processedUrl: result.url,
      metadata: {
        size: result.size,
        format: result.format,
        appliedFilters: filters,
      },
    };
  },
});
// !!

async function run() {
  // ❓ Running a task (Mergent)
  const options = {
    method: 'POST',
    headers: { Authorization: 'Bearer <token>', 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: '4cf95241-fa19-47ef-8a67-71e483747649',
      queue: 'default',
      request: {
        url: 'https://example.com',
        headers: { Authorization: '8BOHec9yUJMJ4sJFqUuC5w==', 'Content-Type': 'application/json' },
        body: 'Hello, world!',
      },
    }),
  };

  fetch('https://api.mergent.co/v2/tasks', options)
    .then((response) => response.json())
    .then((response) => console.log(response))
    .catch((err) => console.error(err));
  // !!

  // ❓ Running a task (Hatchet)
  const result = await imageProcessor.run({
    imageUrl: 'https://example.com/image.png',
    filters: ['blur'],
  });

  // you can await fully typed results
  console.log(result);
  // !!
}

async function schedule() {
  // ❓ Scheduling tasks (Mergent)
  const options = {
    // same options as before
    body: JSON.stringify({
      // same body as before
      delay: '5m',
    }),
  };
  // !!

  // ❓ Scheduling tasks (Hatchet)
  // Schedule the task to run at a specific time
  const runAt = new Date(Date.now() + 1000 * 60 * 60 * 24);
  imageProcessor.schedule(runAt, {
    imageUrl: 'https://example.com/image.png',
    filters: ['blur'],
  });

  // Schedule the task to run every hour
  imageProcessor.cron('run-hourly', '0 * * * *', {
    imageUrl: 'https://example.com/image.png',
    filters: ['blur'],
  });
  // !!
}
