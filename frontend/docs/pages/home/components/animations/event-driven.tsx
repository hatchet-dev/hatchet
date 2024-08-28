"use client";
import Lottie, { LottieProps } from "react-lottie-player";
import * as data from "./event-driven.json";

const EventDriven: React.FC<LottieProps> = () => {
  return <Lottie play loop animationData={data} height={600} width={600} />;
};

export default EventDriven;
