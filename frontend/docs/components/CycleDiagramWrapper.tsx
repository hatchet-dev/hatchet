import dynamic from "next/dynamic";

const CycleDiagram = dynamic(() => import("./CycleDiagram"), {
  ssr: false,
});

export default CycleDiagram;
