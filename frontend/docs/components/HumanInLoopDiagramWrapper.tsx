import dynamic from "next/dynamic";

const HumanInLoopDiagram = dynamic(() => import("./HumanInLoopDiagram"), {
  ssr: false,
});

export default HumanInLoopDiagram;
