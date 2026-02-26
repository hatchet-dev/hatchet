// @ts-nocheck
// These snippets demonstrate common middleware patterns.
// They reference external packages (@aws-sdk/*) that are NOT
// dependencies of the Hatchet SDK — install them in your own project.

// > End-to-end encryption
import { HatchetClient, HatchetMiddleware } from '@hatchet-dev/typescript-sdk/v1';
import { randomUUID, createCipheriv, createDecipheriv, randomBytes } from 'crypto';

// > Offloading large payloads to S3
import { S3Client, PutObjectCommand, GetObjectCommand } from '@aws-sdk/client-s3';
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';

const ALGORITHM = 'aes-256-gcm';
const KEY = Buffer.from(process.env.ENCRYPTION_KEY!, 'hex');

type EncryptedEnvelope = { ciphertext: string; iv: string; tag: string };

function encrypt(plaintext: string): EncryptedEnvelope {
  const iv = randomBytes(16);
  const cipher = createCipheriv(ALGORITHM, KEY, iv);
  const encrypted = Buffer.concat([cipher.update(plaintext, 'utf8'), cipher.final()]);
  return {
    ciphertext: encrypted.toString('base64'),
    iv: iv.toString('base64'),
    tag: cipher.getAuthTag().toString('base64'),
  };
}

function decrypt(ciphertext: string, iv: string, tag: string): string {
  const decipher = createDecipheriv(ALGORITHM, KEY, Buffer.from(iv, 'base64'));
  decipher.setAuthTag(Buffer.from(tag, 'base64'));
  return decipher.update(ciphertext, 'base64', 'utf8') + decipher.final('utf8');
}

type EncryptedInput = { encrypted?: EncryptedEnvelope };

const e2eEncryption: HatchetMiddleware<EncryptedInput> = {
  before: (input) => {
    if (!input.encrypted) return input;
    const { ciphertext, iv, tag } = input.encrypted;
    const decrypted = JSON.parse(decrypt(ciphertext, iv, tag));
    return { ...input, ...decrypted, encrypted: undefined };
  },
  after: (output) => {
    const payload = JSON.stringify(output);
    return { encrypted: encrypt(payload) };
  },
};

const encryptionClient = HatchetClient.init<EncryptedInput>().withMiddleware(e2eEncryption);

const s3 = new S3Client({ region: process.env.AWS_REGION });
const BUCKET = process.env.S3_BUCKET!;
const PAYLOAD_THRESHOLD = 256 * 1024; // 256 KB

async function uploadToS3(data: unknown): Promise<string> {
  const key = `hatchet-payloads/${randomUUID()}.json`;
  await s3.send(
    new PutObjectCommand({
      Bucket: BUCKET,
      Key: key,
      Body: JSON.stringify(data),
      ContentType: 'application/json',
    })
  );
  return getSignedUrl(s3, new GetObjectCommand({ Bucket: BUCKET, Key: key }), {
    expiresIn: 3600,
  });
}

async function downloadFromS3(url: string): Promise<unknown> {
  const res = await fetch(url);
  return res.json();
}

type S3Input = { s3Url?: string };

const s3Offload: HatchetMiddleware<S3Input> = {
  before: async (input) => {
    if (input.s3Url) {
      const restored = (await downloadFromS3(input.s3Url)) as Record<string, any>;
      return { ...restored, s3Url: undefined };
    }
    return input;
  },
  after: async (output) => {
    const serialized = JSON.stringify(output);
    if (serialized.length > PAYLOAD_THRESHOLD) {
      const url = await uploadToS3(output);
      return { s3Url: url };
    }
    return output;
  },
};

const s3Client = HatchetClient.init<S3Input>().withMiddleware(s3Offload);
