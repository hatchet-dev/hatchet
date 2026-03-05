import dynamic from "next/dynamic";

const RAGPipelineDiagram = dynamic(() => import("./RAGPipelineDiagram"), {
  ssr: false,
});

export default RAGPipelineDiagram;
