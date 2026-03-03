import type { TraceSpan } from "@evilmartians/agent-prism-types";
import type { ReactElement, ReactNode } from "react";

import cn from "classnames";
import { SquareTerminal, Tags, ArrowRightLeft } from "lucide-react";
import { useState } from "react";

import type { AvatarProps } from "../Avatar";
import type { TabItem } from "../Tabs";

import { TabSelector } from "../TabSelector";
import { DetailsViewAttributesTab } from "./DetailsViewAttributesTab";
import { DetailsViewHeader } from "./DetailsViewHeader";
import { DetailsViewInputOutputTab } from "./DetailsViewInputOutputTab";
import { DetailsViewRawDataTab } from "./DetailsViewRawDataTab";

type DetailsViewTab = "input-output" | "attributes" | "raw";

export interface DetailsViewProps {
  /**
   * The span data to display in the details view
   */
  data: TraceSpan;

  /**
   * Optional avatar configuration for the header
   */
  avatar?: AvatarProps;

  /**
   * The initially selected tab
   */
  defaultTab?: DetailsViewTab;

  /**
   * Optional className for the root container
   */
  className?: string;

  /**
   * Configuration for the copy button functionality
   */
  copyButton?: {
    isEnabled?: boolean;
    onCopy?: (data: TraceSpan) => void;
  };

  /**
   * Custom header actions to render
   * Can be a ReactNode or a render function that receives the data
   */
  headerActions?: ReactNode | ((data: TraceSpan) => ReactNode);

  /**
   * Optional custom header component to replace the default
   */
  customHeader?: ReactNode | ((props: { data: TraceSpan }) => ReactNode);

  /**
   * Callback fired when the active tab changes
   */
  onTabChange?: (tabValue: DetailsViewTab) => void;
}

const TAB_ITEMS: TabItem<DetailsViewTab>[] = [
  {
    value: "input-output",
    label: "In/Out",
    icon: <ArrowRightLeft className="size-4" />,
  },
  {
    value: "attributes",
    label: "Attributes",
    icon: <Tags className="size-4" />,
  },
  {
    value: "raw",
    label: "RAW",
    icon: <SquareTerminal className="size-4" />,
  },
];

export const DetailsView = ({
  data,
  avatar,
  defaultTab = "input-output",
  className,
  copyButton,
  headerActions,
  customHeader,
  onTabChange,
}: DetailsViewProps): ReactElement => {
  const [tab, setTab] = useState<DetailsViewTab>(defaultTab);

  const handleTabChange = (tabValue: DetailsViewTab) => {
    setTab(tabValue);
    onTabChange?.(tabValue);
  };

  const resolvedHeaderActions =
    typeof headerActions === "function" ? headerActions(data) : headerActions;

  const headerContent = customHeader ? (
    typeof customHeader === "function" ? (
      customHeader({ data })
    ) : (
      customHeader
    )
  ) : (
    <DetailsViewHeader
      data={data}
      avatar={avatar}
      copyButton={copyButton}
      actions={resolvedHeaderActions}
    />
  );

  return (
    <div
      className={cn(
        "border-agentprism-border bg-agentprism-background flex h-full min-h-0 flex-col rounded-md border p-4",
        className,
      )}
    >
      <div className="mb-4 shrink-0">{headerContent}</div>
      <div className="shrink-0">
        <TabSelector
          items={TAB_ITEMS}
          value={tab}
          onValueChange={handleTabChange}
          theme="underline"
          defaultValue={defaultTab}
        />
      </div>

      <div key={tab} className="min-h-0 flex-1 overflow-y-auto py-4">
        {tab === "input-output" && <DetailsViewInputOutputTab data={data} />}
        {tab === "attributes" && <DetailsViewAttributesTab data={data} />}
        {tab === "raw" && <DetailsViewRawDataTab data={data} />}
      </div>
    </div>
  );
};
