import { createElement } from "react";
import { icons } from "lucide-react";
import { docs } from "@/.source/server";
import { loader } from "fumadocs-core/source";

export const source = loader({
  baseUrl: "/",
  source: docs.toFumadocsSource(),
  icon(iconName) {
    if (iconName && iconName in icons) {
      return createElement(icons[iconName as keyof typeof icons], {
        className: "size-4",
        strokeWidth: 1.5,
      });
    }
  },
});
