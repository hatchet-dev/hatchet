import dynamic from "next/dynamic";

const AmazonS3ImagePipelineDiagram = dynamic(
  () => import("./AmazonS3ImagePipelineDiagram"),
  {
    ssr: false,
  },
);

export default AmazonS3ImagePipelineDiagram;
