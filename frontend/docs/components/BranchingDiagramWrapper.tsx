import dynamic from "next/dynamic";

const BranchingDiagram = dynamic(() => import("./BranchingDiagram"), {
  ssr: false,
});

export default BranchingDiagram;
