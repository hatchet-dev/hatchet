import {
  S3Client,
  paginateListBuckets,
  paginateListObjectsV2,
  GetObjectCommand,
  DeleteObjectCommand,
  NoSuchKey,
  NoSuchBucket,
} from '@aws-sdk/client-s3';
import { ConcurrencyLimitStrategy } from '@hatchet/v1';
import { hatchet } from '../../hatchet-client';

const BUCKET_PREFIX = process.env.S3_WORKER_BUCKET_PREFIX ?? 'bucket-';
const MAX_CONCURRENT_BUCKET_POLLERS = parseInt(
  process.env.S3_WORKER_MAX_CONCURRENT_BUCKET_POLLERS ?? '10'
);
const MAX_RUNS_PER_BUCKET = parseInt(process.env.S3_WORKER_MAX_RUNS_PER_BUCKET ?? '20');
const SLOTS = parseInt(process.env.S3_WORKER_SLOTS ?? '40');

// > Client Setup
const s3 = new S3Client({ forcePathStyle: true });
// !!

// > Models
type ListObjectsInput = {
  bucket: string;
};

type ProcessObjectInput = {
  bucket: string;
  key: string;
};
// !!

// > Fetch S3 Buckets
const fetchBucketsWorkflow = hatchet.workflow({
  name: 'fetch_s3_buckets',
  on: {
    cron: '* * * * *',
  },
  concurrency: {
    expression: "'singleton'",
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.CANCEL_NEWEST,
  },
});
// !!

// > Fetch S3 Objects
const fetchObjectsWorkflow = hatchet.workflow<ListObjectsInput>({
  name: 'fetch_s3_objects',
  concurrency: [
    {
      expression: 'input.bucket',
      maxRuns: 1,
      limitStrategy: ConcurrencyLimitStrategy.CANCEL_NEWEST,
    },
    {
      expression: "'constant'",
      maxRuns: MAX_CONCURRENT_BUCKET_POLLERS,
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    },
  ],
});
// !!

// > Process S3 Objects
const processObjectWorkflow = hatchet.workflow<ProcessObjectInput>({
  name: 'process_object',
  concurrency: {
    expression: 'input.bucket',
    maxRuns: MAX_RUNS_PER_BUCKET,
    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
  },
});
// !!

// > Fetch S3 Buckets Task
fetchBucketsWorkflow.task({
  name: 'fetch_buckets',
  fn: async () => {
    for await (const page of paginateListBuckets(
      { client: s3, pageSize: 10 },
      { Prefix: BUCKET_PREFIX }
    )) {
      const items = (page.Buckets ?? [])
        .filter((bucket): bucket is { Name: string } => bucket.Name !== undefined)
        .map((bucket) => ({
          input: { bucket: bucket.Name },
          opts: {
            childKey: bucket.Name,
            additionalMetadata: { 'bucket-name': bucket.Name },
          },
        }));

      if (items.length > 0) {
        await fetchObjectsWorkflow.runManyNoWait(items);
      }
    }

    return {};
  },
});
// !!

// > Fetch S3 Objects Task
fetchObjectsWorkflow.task({
  name: 'fetch_objects',
  fn: async (input: ListObjectsInput) => {
    for await (const page of paginateListObjectsV2(
      { client: s3, pageSize: 100 },
      { Bucket: input.bucket }
    )) {
      const items = (page.Contents ?? [])
        .filter((obj): obj is { Key: string } => obj.Key !== undefined)
        .map((obj) => ({
          input: { bucket: input.bucket, key: obj.Key },
          opts: {
            childKey: `${input.bucket}/${obj.Key}`,
          },
        }));

      if (items.length > 0) {
        await processObjectWorkflow.runManyNoWait(items);
      }
    }

    return {};
  },
});
// !!

// > Download and Process S3 Objects Task
processObjectWorkflow.task({
  name: 'process_object',
  fn: async (input: ProcessObjectInput, ctx) => {
    let body: Buffer;

    try {
      const response = await s3.send(
        new GetObjectCommand({ Bucket: input.bucket, Key: input.key })
      );
      if (!response.Body) {
        return {};
      }
      body = Buffer.from(await response.Body.transformToByteArray());
    } catch (err) {
      if (err instanceof NoSuchKey || err instanceof NoSuchBucket) {
        await ctx.log(`skipping ${input.bucket}/${input.key}: not found`);
        return {};
      }
      throw err;
    }

    // TODO: actual image processing here

    await s3.send(new DeleteObjectCommand({ Bucket: input.bucket, Key: input.key }));
    return {};
  },
});
// !!

async function main() {
  const worker = await hatchet.worker('s3-worker', {
    workflows: [fetchBucketsWorkflow, fetchObjectsWorkflow, processObjectWorkflow],
    slots: SLOTS,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
