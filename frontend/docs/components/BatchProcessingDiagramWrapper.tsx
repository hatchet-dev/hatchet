import dynamic from "next/dynamic";

const BatchProcessingDiagram = dynamic(
  () => import("./BatchProcessingDiagram"),
  { ssr: false },
);

export default BatchProcessingDiagram;
