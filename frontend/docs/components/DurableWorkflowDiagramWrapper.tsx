import dynamic from "next/dynamic";

const DurableWorkflowDiagram = dynamic(
  () => import("./DurableWorkflowDiagram"),
  {
    ssr: false,
  },
);

export default DurableWorkflowDiagram;
