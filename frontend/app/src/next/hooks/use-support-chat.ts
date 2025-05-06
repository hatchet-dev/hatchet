import useApiMeta from "./use-api-meta";

export default function useSupportChat() {
  const { oss } = useApiMeta();
  return { 
    show: () => (window as any).Pylon('show'),
    isEnabled: () => !!oss?.pylonAppId,
  };
}
