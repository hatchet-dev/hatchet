import dynamic from "next/dynamic";

const AgentLoopDiagram = dynamic(() => import("./AgentLoopDiagram"), {
  ssr: false,
});

export default AgentLoopDiagram;
