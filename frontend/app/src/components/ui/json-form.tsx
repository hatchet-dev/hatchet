import CopyToClipboard from './copy-to-clipboard';
import { cn } from '@/lib/utils';
import { RJSFSchema } from '@rjsf/utils';
import validator from '@rjsf/validator-ajv8';
import Form from '@rjsf/core';
import { set } from 'react-hook-form';
import { PlayIcon } from '@radix-ui/react-icons';
import { Button } from './button';

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
  setInput,
  disabled,
  onSubmit,
}: {
  json: JSONType;
  className?: string;
  setInput: (input: string) => void;
  disabled?: boolean;
  onSubmit: () => void;
}) {
  const input = !!json ? json['input'] : json || ({} as JSONType);

  const schema: RJSFSchema = {
    type: 'object',
    properties: {
      input: {
        type: 'object',
        properties: {
          test: {
            type: 'string',
            default: 'test',
          },
        },
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

  const uiSchema = {
    input: {
      test: {
        'ui:widget': 'textarea',
      },
    },
  };

  return (
    <div
      className={cn(
        className,
        'w-full h-fit relative rounded-lg overflow-hidden',
      )}
    >
      <Form
        schema={schema}
        disabled={disabled}
        uiSchema={uiSchema}
        validator={validator}
        onChange={(data) => {
          setInput(JSON.stringify(data.formData));
        }}
        onSubmit={onSubmit}
        onError={log('errors')}
      >
        <Button className="w-fit" disabled={disabled}>
          <PlayIcon
            className={cn(disabled ? 'rotate-180' : '', 'h-4 w-4 mr-2')}
          />
          Play Step
        </Button>
      </Form>
    </div>
  );
}
