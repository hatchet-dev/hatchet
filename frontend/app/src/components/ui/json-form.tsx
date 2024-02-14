import { cn } from '@/lib/utils';
import { ObjectFieldTemplateProps, RJSFSchema, UiSchema } from '@rjsf/utils';
import validator from '@rjsf/validator-ajv8';
import Form from '@rjsf/core';
import { PlayIcon } from '@radix-ui/react-icons';
import { Button } from './button';
import { useState } from 'react';

type JSONPrimitive = string | number | boolean | null;
type JSONType = { [key: string]: JSONType | JSONPrimitive };

const DEFAULT_COLLAPSED = ['advanced', 'user data'];

export const CollapsibleSection = (props: ObjectFieldTemplateProps) => {
  const [open, setOpen] = useState(!DEFAULT_COLLAPSED.includes(props.title));

  return (
    <div>
      {props.title && (
        <div
          onClick={() => setOpen((x) => !x)}
          className="border-b-2 mb-2 border-gray-500 pb-2 text-xl font-bold flex items-center cursor-pointer"
        >
          <svg
            className={`mr-2 h-6 w-6 ${open ? 'rotate-180' : ''}`}
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M19 9l-7 7-7-7" />
          </svg>

          {props.title}
        </div>
      )}
      {props.description}
      {open &&
        (props.properties.length > 0 ? (
          props.properties.map((element, i) => (
            <div className="property-wrapper ml-4" key={i}>
              {element.content}
            </div>
          ))
        ) : (
          <div className="ml-4">empty state</div>
        ))}
    </div>
  );
};

export function JsonForm({
  inputSchema,
  inputData,
  className,
  setInput,
  disabled,
  onSubmit,
}: {
  inputSchema: JSONType;
  className?: string;
  inputData: JSONType;
  setInput: React.Dispatch<React.SetStateAction<string>>;
  disabled?: boolean;
  onSubmit: () => void;
}) {
  const schema = {
    ...inputSchema,
    required: undefined,
    $schema: undefined,
    properties: {
      ...(inputSchema.properties as any),
      triggered_by: undefined,
      advanced: {
        // Transform the schema to wrap the triggered by field
        type: 'object',
        properties: {
          triggered_by: inputSchema.properties
            ? (inputSchema.properties as any).triggered_by
            : undefined,
        },
      },
    },
  } as RJSFSchema;

  delete schema.properties?.triggered_by;

  const uiSchema: UiSchema<any, RJSFSchema, any> = {
    input: {
      'ui:title': 'workflow input',
    },
    parents: {
      'ui:title': 'parent step data',
    },
    overrides: {
      'ui:title': 'step overrides',
    },
    user_data: {
      'ui:title': 'user data',
    },
    'ui:order': ['input', 'overrides', 'parents', '*'],
  };

  return (
    <div
      className={cn(
        className,
        'w-full h-fit relative rounded-lg overflow-hidden',
      )}
    >
      <Form
        formData={inputData}
        schema={schema}
        disabled={disabled}
        templates={{
          ObjectFieldTemplate: CollapsibleSection,
        }}
        uiSchema={uiSchema}
        validator={validator}
        onChange={(data) => {
          // Transform the data to unwrap the advanced fields
          const formData = { ...data.formData, ...data.formData.advanced };
          delete formData.advanced;
          setInput((prev) =>
            JSON.stringify({
              ...JSON.parse(prev),
              ...formData,
            }),
          );
        }}
        onSubmit={onSubmit}
        onError={(e) => {
          console.error(e);
        }}
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
