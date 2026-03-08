import dynamic from "next/dynamic";

const DocumentProcessingDiagram = dynamic(
  () => import("./DocumentProcessingDiagram"),
  {
    ssr: false,
  },
);

export default DocumentProcessingDiagram;
