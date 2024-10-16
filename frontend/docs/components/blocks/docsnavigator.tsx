"use client";

import { SiGo, SiPython } from "react-icons/si";
import { BiLogoTypescript } from "react-icons/bi";
import CardBlock from "./card-block";

export default function DocsNavigator() {
  return (
    <section>
      <div className="relative max-w-full mx-auto">
        <CardBlock
          title="SDK References"
          description="Reference APIs for integrating Hatchet into your application."
          items={[
            {
              icon: <SiPython size={18} />,
              title: "Python SDK",
              description:
                "Learn how to integrate Hatchet with your Python applications.",
              link: "/sdks/python-sdk",
            },
            {
              icon: <BiLogoTypescript size={22} />,
              title: "Typescript SDK",
              description:
                "Learn how to integrate Hatchet with your Python applications.",
              link: "/sdks/typescript-sdk",
            },
            {
              icon: <SiGo size={22} />,
              title: "Go SDK",
              description:
                "Learn how to integrate Hatchet with your Go applications.",
              link: "/sdks/go-sdk",
            },
          ]}
        />
      </div>
    </section>
  );
}
