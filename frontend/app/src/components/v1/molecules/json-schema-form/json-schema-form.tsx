import { prepareSchema } from './prepare-schema';
import { templates, widgets } from './templates';
import Form from '@rjsf/core';
import { ErrorTransformer, getDefaultFormState } from '@rjsf/utils';
import validator from '@rjsf/validator-ajv8';
import { useEffect, useMemo, useRef, useState } from 'react';

const transformErrors: ErrorTransformer = (errors) =>
  errors.map((error) =>
    error.name === 'required' ? { ...error, message: 'Required' } : error,
  );

type JsonSchemaFormProps = {
  schema: object;
  // read once on mount: the form owns its state afterwards
  initialValue: unknown;
  onChange: (value: unknown) => void;
};

// Default export so callers can React.lazy() it and keep rjsf and ajv out of
// the main bundle.
export default function JsonSchemaForm({
  schema,
  initialValue,
  onChange,
}: JsonSchemaFormProps) {
  const { schema: rjsfSchema, uiSchema } = useMemo(
    () => prepareSchema(schema),
    [schema],
  );
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const [formData] = useState(() =>
    getDefaultFormState(validator, rjsfSchema, initialValue, rjsfSchema),
  );

  // the form displays schema defaults, so report that value rather than
  // leaving the caller with the pre-form input
  useEffect(() => {
    onChangeRef.current(formData);
  }, [formData]);

  return (
    <Form
      className="rjsf"
      schema={rjsfSchema}
      uiSchema={uiSchema}
      templates={templates}
      widgets={widgets}
      validator={validator}
      transformErrors={transformErrors}
      formData={formData}
      onChange={(e) => onChangeRef.current(e.formData)}
      liveValidate="onChange"
      showErrorList={false}
      noHtml5Validate
    />
  );
}
