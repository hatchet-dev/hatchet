import React from "react";
import IconProps from "./icon-props";

function Xmark(props: IconProps) {
  const fill = props.fill || "currentColor";
  const secondaryfill = props.secondaryfill || fill;

  return (
    <svg
      height="18"
      width="18"
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill={fill} stroke={fill}>
        <line
          fill="none"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x1="14"
          x2="4"
          y1="4"
          y2="14"
        />
        <line
          fill="none"
          stroke={fill}
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x1="4"
          x2="14"
          y1="4"
          y2="14"
        />
      </g>
    </svg>
  );
}

export default Xmark;
