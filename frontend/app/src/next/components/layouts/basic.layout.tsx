export default function BasicLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex-grow h-full w-full">
      <div className="px-8 py-12 overflow-y-scroll">{children}</div>
    </div>
  );
}
