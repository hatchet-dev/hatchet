export default function BasicLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex-grow h-full w-full">
      <div className="p-4 md:p-8 lg:p-12">{children}</div>
    </div>
  );
}
