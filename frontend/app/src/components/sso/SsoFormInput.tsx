import { SsoErrorText } from './SsoErrorText';
import { Input } from '@/components/v1/ui/input';

export function SsoFormInput({
  id,
  type = 'text',
  placeholder,
  value,
  onChange,
  error,
  readOnly,
  autoComplete,
  ...dataAttributes
}: {
  id: string;
  type?: string;
  placeholder?: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  readOnly?: boolean;
  autoComplete?: string;
} & Record<`data-${string}`, string | boolean | undefined>) {
  return (
    <>
      <Input
        id={id}
        type={type}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        readOnly={readOnly}
        autoComplete={autoComplete}
        {...dataAttributes}
      />
      {error && <SsoErrorText>{error}</SsoErrorText>}
    </>
  );
}
