export default function useSupportChat() {
  return { show: () => (window as any).Pylon('show') };
}
