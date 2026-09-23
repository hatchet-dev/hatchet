import { AxiosError } from 'axios';
import { V1CELDebugResponseStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';

export type DebugCELInput = {
  expression: string;
  input: Record<string, any>;
  additionalMetadata?: Record<string, string>;
  filterPayload?: Record<string, any>;
};

export type CELOutputType = 'bool' | 'string' | 'int';

const VALID_OUTPUT_TYPES: readonly CELOutputType[] = ['bool', 'string', 'int'] as const;

export type CELSuccess = {
  status: V1CELDebugResponseStatus.SUCCESS;
  output?: boolean;
  outputStr?: string;
  outputInt?: number;
  outputType: CELOutputType;
  asBoolean(): boolean;
  asString(): string;
  asNumber(): number;
};

export type CELEvaluationResult =
  | CELSuccess
  | {
      status: V1CELDebugResponseStatus.ERROR;
      error: string;
    };

function makeCELSuccess(
  output: boolean | undefined,
  outputStr: string | undefined,
  outputInt: number | undefined,
  outputType: CELOutputType,
): CELSuccess {
  return {
    status: V1CELDebugResponseStatus.SUCCESS,
    output,
    outputStr,
    outputInt,
    outputType,
    asBoolean() {
      if (outputType !== 'bool' || output === undefined) {
        throw new Error(`Cannot convert ${outputType} result to boolean`);
      }
      return output;
    },
    asString() {
      if (outputType !== 'string' || outputStr === undefined) {
        throw new Error(`Cannot convert ${outputType} result to string`);
      }
      return outputStr;
    },
    asNumber() {
      if (outputType !== 'int' || outputInt === undefined) {
        throw new Error(`Cannot convert ${outputType} result to number`);
      }
      return outputInt;
    },
  };
}

/**
 * Client for debugging CEL expressions.
 */
export class CELClient {
  api: HatchetClient['api'];
  tenantId: string;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
  }

  /**
   * Debug a CEL expression with the provided input, filter payload, and optional metadata. Useful for testing and validating CEL expressions and debugging issues in production.
   *
   * @param input - The input data to evaluate the CEL expression against.
   * @returns A promise that resolves to the evaluation result, which can either be a success with an output or an error with a message.
   * @throws Will throw an error if the input is invalid or the API call fails.
   */
  async debug(input: DebugCELInput): Promise<CELEvaluationResult> {
    try {
      const response = await this.api.v1CelDebug(this.tenantId, input);

      if (response.data.status === V1CELDebugResponseStatus.ERROR) {
        if (!response.data.error) {
          throw new Error('No error message received from CEL debug API.');
        }

        return {
          status: V1CELDebugResponseStatus.ERROR,
          error: response.data.error,
        };
      }

      if (response.data.outputType === undefined) {
        throw new Error('No output received from CEL debug API.');
      }

      const rawType = response.data.outputType;
      if (!rawType || !VALID_OUTPUT_TYPES.includes(rawType as CELOutputType)) {
        throw new Error(`Unknown outputType received from CEL debug API: ${rawType}`);
      }

      return makeCELSuccess(
        response.data.output,
        response.data.outputStr,
        response.data.outputInt,
        rawType as CELOutputType,
      );
    } catch (err) {
      if (err instanceof AxiosError) {
        throw new Error(JSON.stringify(err.response?.data.errors), { cause: err });
      }
      throw err;
    }
  }
}
