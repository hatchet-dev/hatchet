import { cn } from '@/lib/utils';
import { RJSFSchema } from '@rjsf/utils';
import validator from '@rjsf/validator-ajv8';
import Form from '@rjsf/core';
import { PlayIcon } from '@radix-ui/react-icons';
import { Button } from './button';

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
  const schema = json as RJSFSchema;

  const uiSchema = {
    input: {
      test: {
        'ui:widget': 'textarea',
      },
    },
  };

  console.log('schema', schema);

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
