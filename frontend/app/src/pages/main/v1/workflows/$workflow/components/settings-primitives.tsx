export function EmptyState({ message }: { message: string }) {
  return <p className="text-xs italic text-muted-foreground">{message}</p>;
}

export function FieldLabel({ children }: { children: React.ReactNode }) {
  return <div className="mb-1 text-xs text-muted-foreground">{children}</div>;
}
