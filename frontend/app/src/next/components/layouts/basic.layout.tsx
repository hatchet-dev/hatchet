export default function BasicLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex-grow h-full w-full">
      <div className="px-4 py-8">{children}</div>
    </div>
  );
}
