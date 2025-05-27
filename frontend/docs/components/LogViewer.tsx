export function LogViewer({ children }: { children: React.ReactNode }) {
  return (
    <div className="py-4 my-4 max-h-96 overflow-auto">
      <pre className="text-sm">{children}</pre>
    </div>
  );
}
