import { BaseInputTemplateProps, getInputProps } from '@rjsf/utils';
import { ChangeEvent, FocusEvent, useEffect, useRef } from 'react';

export function DynamicSizeInputTemplate(props: BaseInputTemplateProps) {
  const {
    schema,
    id,
    options,
    label,
    value,
    type,
    placeholder,
    required,
    disabled,
    readonly,
    autofocus,
    onChange,
    onChangeOverride,
    onBlur,
    onFocus,
    rawErrors,
    hideError,
    uiSchema,
    registry,
    formContext,
    ...rest
  } = props;

  const ref = useRef<HTMLTextAreaElement>(null);

  const onTextChange = ({
    target: { value: val },
  }: ChangeEvent<HTMLTextAreaElement>) => {
    // Use the options.emptyValue if it is specified and newVal is also an empty string
    onChange(val === '' ? options.emptyValue || '' : val);
  };
  const onTextBlur = ({
    target: { value: val },
  }: FocusEvent<HTMLTextAreaElement>) => onBlur(id, val);
  const onTextFocus = ({
    target: { value: val },
  }: FocusEvent<HTMLTextAreaElement>) => onFocus(id, val);

  const inputProps = { ...rest, ...getInputProps(schema, type, options) };

  const setHeight = (e: HTMLTextAreaElement) => {
    e.style.height = 'auto';
    e.style.height = `${e.scrollHeight}px`;
  };

  useEffect(() => {
    // Call adjustHeight whenever the watched value changes externally
    if (!ref.current) {
      return;
    }
    setHeight(ref.current);
  }, [ref.current, ref.current?.value]);

  return (
    <textarea
      ref={ref}
      id={id}
      value={value}
      placeholder={placeholder}
      disabled={disabled}
      readOnly={readonly}
      autoFocus={autofocus}
      className="overflow-y-hidden"
      onChange={
        (onChangeOverride as
          | ((event: ChangeEvent<HTMLTextAreaElement>) => void)
          | undefined) || onTextChange
      }
      onBlur={onTextBlur}
      onFocus={onTextFocus}
      {...inputProps}
    />
  );
}
