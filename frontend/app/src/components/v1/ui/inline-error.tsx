import { Alert, AlertDescription } from '@/components/v1/ui/alert';

function capitalize(message: string): string {
  if (!message) {
    return message;
  }

  return message.charAt(0).toUpperCase() + message.slice(1);
}

// Inline error banner for use inside modals/dialogs, where a global toast would
// render behind the overlay and be invisible. Feed it the `string[]` from
// `useApiError({ setErrors })`.
export function InlineError({
  errors,
  className,
}: {
  errors: string[];
  className?: string;
}) {
  if (errors.length === 0) {
    return null;
  }

  return (
    <Alert variant="destructive" className={className}>
      <AlertDescription>
        {errors.length === 1 ? (
          capitalize(errors[0])
        ) : (
          <ul className="list-disc space-y-1 pl-4">
            {errors.map((error, i) => (
              <li key={i}>{capitalize(error)}</li>
            ))}
          </ul>
        )}
      </AlertDescription>
    </Alert>
  );
}
