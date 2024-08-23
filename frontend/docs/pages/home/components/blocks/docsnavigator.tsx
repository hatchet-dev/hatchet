"use client";

import { SiGo, SiPython } from "react-icons/si";
import { BiLogoTypescript } from "react-icons/bi";
import CardBlock from "./card-block";


export default function DocsNavigator() {
  return (
    <section>
      <div className="relative max-w-full mx-auto">
        <CardBlock
          title="Get started with Hatchet"
          description="Get up and running quickly with Hatchet quickstart SDK tutorials"
          items={[
            {
              icon: <SiPython size={18}/>,
              title: "Python SDK",
              description:
                "Learn how to integrate Hatchet with your Python applications.",
              link: "/sdks/python-sdk",
            },
            {
              icon: <BiLogoTypescript size={22} />,
              title: "Typescript SDK",
              description:
                "Get started with the TypeScript SDK and leverage Hatchet in your JavaScript applications.",
              link: "/sdks/typescript-sdk",
            },
            {
              icon: <SiGo size={22}/>,
              title: "Go SDK",
              description:
                "Get started with the Go SDK and learn how to integrate Hatchet into your Go applications.",
              link: "/sdks/go-sdk",
            },
          ]}
        />
      </div>
    </section>
  );
}
