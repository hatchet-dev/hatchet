"use client";
import Lottie, { LottieProps } from "react-lottie-player";
import * as data from "./multi-modal.json";

const MultiModal: React.FC<LottieProps> = () => {
  return <Lottie play loop animationData={data} height={600} width={600} />;
};

export default MultiModal;
