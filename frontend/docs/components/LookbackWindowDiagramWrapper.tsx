import dynamic from "next/dynamic";

const LookbackWindowDiagram = dynamic(() => import("./LookbackWindowDiagram"), {
  ssr: false,
});

export default LookbackWindowDiagram;
