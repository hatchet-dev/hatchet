import type { TraceSpan } from "@evilmartians/agent-prism-types";

import { type ReactElement } from "react";

import { CopyButton } from "../CopyButton";
import { DetailsViewJsonOutput } from "./DetailsViewJsonOutput";

interface RawDataTabProps {
  data: TraceSpan;
}

export const DetailsViewRawDataTab = ({
  data,
}: RawDataTabProps): ReactElement => (
  <div className="border-agentprism-border rounded-md border bg-transparent">
    <div className="relative">
      <div className="pointer-events-none sticky top-0 z-10 flex justify-end p-1.5">
        <div className="pointer-events-auto">
          <CopyButton label="Raw" content={data.raw} />
        </div>
      </div>

      <div className="-mt-12">
        <DetailsViewJsonOutput
          content={data.raw}
          id={data.id || "span-details"}
        />
      </div>
    </div>
  </div>
);
