import dynamic from "next/dynamic";

const TaskEvictionDiagram = dynamic(() => import("./TaskEvictionDiagram"), {
  ssr: false,
});

export default TaskEvictionDiagram;
