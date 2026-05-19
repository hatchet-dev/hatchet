import dynamic from "next/dynamic";

const EventDrivenDiagram = dynamic(() => import("./EventDrivenDiagram"), {
  ssr: false,
});

export default EventDrivenDiagram;
