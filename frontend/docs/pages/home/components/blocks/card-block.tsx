import React from "react";
import Particles from "../particles";
import { HighlighterItem } from "../highlighter";
import { cn } from "../../../../lib/utils";
import Link from "next/link";

export default function CardBlock({
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
          <Card
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

function Card({
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
          "dark:bg-slate-900 dark:opacity-75 dark:hover:border-indigo-600/70 dark:hover:shadow dark:hover:shadow-indigo-600/25 dark:border-slate-700",
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
