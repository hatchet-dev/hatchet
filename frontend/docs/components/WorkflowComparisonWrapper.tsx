import dynamic from "next/dynamic";

const WorkflowComparison = dynamic(() => import("./WorkflowComparison"), {
  ssr: false,
});

export default WorkflowComparison;
