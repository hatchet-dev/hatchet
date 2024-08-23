"use client";

import Particles from "../particles";
import { HighlighterItem } from "../highlighter";
import { cn } from "../../../../lib/utils";
import Link from "next/link";
import { SiGo, SiPython } from "react-icons/si";
import { HiOutlineCloud } from "react-icons/hi";
import { IoServer } from "react-icons/io5";
import { BiLogoTypescript } from "react-icons/bi";


export default function DocsNavigator() {
  return (
    <section>
      <div className="relative max-w-full mx-auto">
        <DocsNavSection
          title="Get started with Hatchet"
          description="Get up and running quickly with Hatchet quickstart tutorials"
          items={[
            {
              icon: <HiOutlineCloud size={18}/>,
              title: "Hatchet Cloud Quickstart",
              description:
                "Quickly set up Hatchet Cloud, register API keys, and deploy your first workflow.",
              link: "/hatchet-cloud-quickstart",
            },
            {
              icon: <IoServer size={18} />,
              title: "Hatchet Self-Hosted Quickstart",
              description:
                "Learn to deploy and manage Hatchet on your own infrastructure for powerful workflow orchestration.",
              link: "/self-hosting",
            },
          ]}
        />

        <DocsNavSection
          title="We also have a number of guides for getting started with the Hatchet SDKs"
          description="Explore quickstart guides for various SDKs."
          items={[
            {
              icon: <SiGo size={22}/>,
              title: "Go SDK Quickstart",
              description:
                "Get started with the Go SDK and learn how to integrate Hatchet into your Go applications.",
              link: "/sdks/go-sdk",
            },
            {
              icon: <SiPython size={18}/>,
              title: "Python SDK Quickstart",
              description:
                "Learn how to integrate Hatchet with your Python applications.",
              link: "/sdks/python-sdk",
            },
            {
              icon: <BiLogoTypescript size={22} />,
              title: "Typescript SDK Quickstart",
              description:
                "Get started with the TypeScript SDK and leverage Hatchet in your JavaScript applications.",
              link: "/sdks/typescript-sdk",
            },
          ]}
        />
      </div>
    </section>
  );
}

// DocsNavSection Component
function DocsNavSection({
  title,
  description,
  items,
}: {
  title: string;
  description: string;
  items: Array<{
    icon: React.ReactNode;
    title: string;
    description: string;
    link: string;
  }>;
}) {
  return (
    <div className="pt-4" data-aos="fade-down">
      <h3 className="h3 mb-4">{description}</h3>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {items.map((item, index) => (
          <DocsNavItem
            key={index}
            icon={item.icon}
            title={item.title}
            description={item.description}
            link={item.link}
          />
        ))}
      </div>
    </div>
  );
}

// DocsNavItem Component
function DocsNavItem({
  icon,
  title,
  description,
  link,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  link: string;
}) {
  return (
    <HighlighterItem>
      <Link
        className={cn(
          "block cursor-pointer relative h-full rounded-[inherit] z-20 overflow-hidden border",
          "bg-slate-100 hover:border-indigo-600/70 hover:shadow-slate-600/25 border-slate-200",
          "dark:bg-slate-900 dark:opacity-75 dark:hover:border-indigo-600/70 dark:hover:shadow dark:hover:shadow-indigo-600/25 dark:border-slate-700"
        )}
        href={link}
        passHref
      >
        {/* Particles animation */}
        <Particles
          className="absolute inset-0 -z-10 opacity-0 group-hover/slide:opacity-100 transition-opacity duration-500 ease-in-out"
          quantity={3}
        />
        {/* Radial gradient */}
        <div
          className="absolute bottom-0 translate-y-1/2 left-1/2 -translate-x-1/2 pointer-events-none -z-10 w-1/3 aspect-square"
          aria-hidden="true"
        >
          <div className="absolute inset-0 translate-z-0 rounded-full dark:bg-slate-800 transition-colors duration-500 ease-in-out blur-[60px] dark:hover:bg-indigo-500" />
        </div>
        <div className="flex flex-col p-6 h-full">
          <div className="grow flex flex-col gap-1">
            <div className="flex flex-row gap-2 items-center">
              <span className="shrink-0 fill-slate-300">{icon}</span>

              <span className="font-bold text-base">{title}</span>
            </div>

            <div className="text-sm text-slate-600 dark:text-slate-400 text-left">
              {description}
            </div>
          </div>
        </div>
      </Link>
    </HighlighterItem>
  );
}
