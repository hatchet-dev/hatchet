export function LogViewer({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <details className="mt-4 rounded-lg">
      <summary className="px-4 py-2 cursor-pointer bg-gray-50 dark:bg-gray-800 font-medium">
        {title}
      </summary>
      <div className="py-4 max-h-96 overflow-auto">
        <pre className="text-sm">{children}</pre>
      </div>
    </details>
  );
}
