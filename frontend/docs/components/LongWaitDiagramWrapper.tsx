import dynamic from "next/dynamic";

const LongWaitDiagram = dynamic(() => import("./LongWaitDiagram"), {
  ssr: false,
});

export default LongWaitDiagram;
