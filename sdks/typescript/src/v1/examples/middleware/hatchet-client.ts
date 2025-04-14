/* eslint-disable max-classes-per-file */
import { JsonObject } from '@hatchet/v1/types';
import { Serializable, Middleware } from '@hatchet/v1/next';
import { HatchetClient } from '../../client/client';

class EncodeSerializer implements Serializable<unknown> {
  deserialize(input: JsonObject) {
    console.log('client-encode-deserialize', input);

    if (input.encrypted && typeof input.encrypted === 'string') {
      console.warn('WARNING THIS IS NOT REAL ENCRYPTION');
      const decrypted = Buffer.from(input.encrypted, 'base64').toString('utf-8');
      return JSON.parse(decrypted);
    }

    return input;
  }

  serialize(input: unknown) {
    console.warn('WARNING THIS IS NOT REAL ENCRYPTION');
    const encrypted = Buffer.from(JSON.stringify(input)).toString('base64');
    console.log('client-encode-serialize', input);
    return {
      encrypted,
    };
  }
}

class EncodeMiddleware implements Middleware {
  input = new EncodeSerializer();
  output = new EncodeSerializer();
}

export const hatchet = HatchetClient.init({
  middleware: [new EncodeMiddleware()],
});
