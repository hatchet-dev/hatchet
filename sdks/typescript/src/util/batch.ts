interface BatchedItem<T> {
  batchIndex: number;
  payloads: T[];
  originalIndices: number[];
}

const utf8 = new TextEncoder();

export function batch<T>(payloads: T[], numElm: number, maxBytes: number): Array<BatchedItem<T>> {
  const batches: Array<BatchedItem<T>> = [];

  let currentBatchPayloads: T[] = [];
  let currentBatchIndices: number[] = [];
  let currentBatchSize = 0;

  for (let i = 0; i < payloads.length; i += 1) {
    const request = payloads[i];
    const requestSize = utf8.encode(JSON.stringify(request)).byteLength;

    // Check if adding this request would exceed either the payload limit or batch size
    if (
      currentBatchPayloads.length > 0 &&
      (currentBatchSize + requestSize > maxBytes || currentBatchPayloads.length >= numElm)
    ) {
      batches.push({
        batchIndex: batches.length,
        payloads: currentBatchPayloads,
        originalIndices: currentBatchIndices,
      });
      currentBatchPayloads = [];
      currentBatchIndices = [];
      currentBatchSize = 0;
    }

    // Add the request to the current batch
    currentBatchPayloads.push(request);
    currentBatchIndices.push(i);
    currentBatchSize += requestSize;
  }

  if (currentBatchPayloads.length > 0) {
    batches.push({
      batchIndex: batches.length,
      payloads: currentBatchPayloads,
      originalIndices: currentBatchIndices,
    });
  }

  return batches;
}
