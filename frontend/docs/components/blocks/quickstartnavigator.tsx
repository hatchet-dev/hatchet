"use client";

import { SiGo, SiPython } from "react-icons/si";
import { BiLogoTypescript } from "react-icons/bi";
import CardBlock from "./card-block";

export default function QuickstartNavigator() {
  return (
    <section>
      <div className="relative max-w-full mx-auto">
        <CardBlock
          title="SDK Quickstarts"
          description=""
          items={[
            {
              icon: <SiPython size={18} />,
              title: "Python Quickstart",
              description:
                "Get up and running quickly with our Python quickstart.",
              link: "https://github.com/hatchet-dev/hatchet-python-quickstart",
            },
            {
              icon: <BiLogoTypescript size={22} />,
              title: "Typescript SDK",
              description:
                "Get up and running quickly with our Typescript quickstart.",
              link: "https://github.com/hatchet-dev/hatchet-typescript-quickstart",
            },
            {
              icon: <SiGo size={22} />,
              title: "Go SDK",
              description: "Get up and running quickly with our Go quickstart.",
              link: "https://github.com/hatchet-dev/hatchet-go-quickstart",
            },
          ]}
        />
      </div>
    </section>
  );
}
