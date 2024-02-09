import CopyToClipboard from './copy-to-clipboard';
import { cn } from '@/lib/utils';
import { RJSFSchema } from '@rjsf/utils';
import validator from '@rjsf/validator-ajv8';
import Form from '@rjsf/core';

const schema: RJSFSchema = {
  title: 'Todo',
  type: 'object',
  required: ['title'],
  properties: {
    title: { type: 'string', title: 'Title', default: 'A new task' },
    done: { type: 'boolean', title: 'Done?', default: false },
  },
};

const log = (type) => console.log.bind(console, type);

type JSONPrimitive = string | number | boolean | null;
type JSONType = { [key: string]: JSONType | JSONPrimitive };

export function JsonForm({
  json,
  className,
}: {
  json: JSONType;
  className?: string;
}) {
  const input = !!json ? json['input'] : json || ({} as JSONType);

  const schema: RJSFSchema = {
    type: 'object',
    $schema: 'https://json-schema.org/draft/2020-12/schema',
    required: ['parents', 'overrides', 'user_data', 'triggered_by', 'input'],
    properties: {
      input: {
        type: 'object',
        properties: {},
        additionalProperties: false,
      },
      parents: {
        type: 'object',
        properties: {},
        additionalProperties: false,
      },
      overrides: {
        type: 'object',
        required: ['test', 'test2'],
        properties: {
          test: {
            type: 'string',
            default: 'test',
          },
          test2: {
            type: 'integer',
            default: 100,
          },
        },
        additionalProperties: false,
      },
      user_data: {
        type: 'object',
        properties: {},
        additionalProperties: false,
      },
      triggered_by: {
        type: 'string',
        default: 'schedule',
      },
    },
    additionalProperties: false,
  };

  return (
    <div
      className={cn(
        className,
        'w-full h-fit relative rounded-lg overflow-hidden',
      )}
    >
      {JSON.stringify(input)}
      <Form
        schema={schema}
        validator={validator}
        onChange={log('changed')}
        onSubmit={log('submitted')}
        onError={log('errors')}
      />
      ,
    </div>
  );
}
