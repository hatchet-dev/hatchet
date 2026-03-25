import { formatDuration, getTimelineData } from '../agent-prism-data';
import type { OtelSpanTree } from '../span-tree-type';
import type { SpanCardConnectorType } from './SpanCardConnector';
import { SpanCardConnector } from './SpanCardConnector';
import { SpanCardTimeline } from './SpanCardTimeline';
import { SpanCardToggle } from './SpanCardToggle';
import * as Collapsible from '@radix-ui/react-collapsible';
import cn from 'classnames';
import type { FC, KeyboardEvent, MouseEvent } from 'react';
import { useCallback } from 'react';

const LAYOUT_CONSTANTS = {
  CONNECTOR_WIDTH: 20,
  CONTENT_BASE_WIDTH: 320,
} as const;

type ExpandButtonPlacement = 'inside' | 'outside';

export type SpanCardViewOptions = {
  withStatus?: boolean;
  expandButton?: ExpandButtonPlacement;
};

const DEFAULT_VIEW_OPTIONS: Required<SpanCardViewOptions> = {
  withStatus: true,
  expandButton: 'inside',
};

interface SpanCardProps {
  data: OtelSpanTree;
  level?: number;
  selectedSpan?: OtelSpanTree;
  onSpanSelect?: (span: OtelSpanTree) => void;
  minStart: number;
  maxEnd: number;
  isLastChild: boolean;
  prevLevelConnectors?: SpanCardConnectorType[];
  expandedSpansIds: string[];
  onExpandSpansIdsChange: (ids: string[]) => void;
  viewOptions?: SpanCardViewOptions;
}

interface SpanCardState {
  isExpanded: boolean;
  hasChildren: boolean;
  isSelected: boolean;
}

const getContentWidth = ({
  level,
  hasExpandButton,
  contentPadding,
  expandButton,
}: {
  level: number;
  hasExpandButton: boolean;
  contentPadding: number;
  expandButton: ExpandButtonPlacement;
}) => {
  let width =
    LAYOUT_CONSTANTS.CONTENT_BASE_WIDTH -
    level * LAYOUT_CONSTANTS.CONNECTOR_WIDTH;

  if (hasExpandButton && expandButton === 'inside') {
    width -= LAYOUT_CONSTANTS.CONNECTOR_WIDTH;
  }

  if (expandButton === 'outside' && level === 0) {
    width -= LAYOUT_CONSTANTS.CONNECTOR_WIDTH;
  }

  return Math.max(width - contentPadding, 40);
};

const getGridTemplateColumns = ({
  connectorsColumnWidth,
  expandButton,
}: {
  connectorsColumnWidth: number;
  expandButton: ExpandButtonPlacement;
}) => {
  if (expandButton === 'inside') {
    return `${connectorsColumnWidth}px 1fr`;
  }

  return `${connectorsColumnWidth}px 1fr ${LAYOUT_CONSTANTS.CONNECTOR_WIDTH}px`;
};

const getContentPadding = ({
  level,
  hasExpandButton,
}: {
  level: number;
  hasExpandButton: boolean;
}) => {
  if (level === 0) {
    return 0;
  }

  if (hasExpandButton) {
    return 4;
  }

  return 8;
};

const getConnectorsLayout = ({
  level,
  hasExpandButton,
  isLastChild,
  prevConnectors,
  expandButton,
}: {
  hasExpandButton: boolean;
  isLastChild: boolean;
  level: number;
  prevConnectors: SpanCardConnectorType[];
  expandButton: ExpandButtonPlacement;
}): {
  connectors: SpanCardConnectorType[];
  connectorsColumnWidth: number;
} => {
  const connectors: SpanCardConnectorType[] = [];

  if (level === 0) {
    return {
      connectors: expandButton === 'inside' ? [] : ['vertical'],
      connectorsColumnWidth: 20,
    };
  }

  for (let i = 0; i < level - 1; i++) {
    connectors.push('vertical');
  }

  if (!isLastChild) {
    connectors.push('t-right');
  }

  if (isLastChild) {
    connectors.push('corner-top-right');
  }

  let connectorsColumnWidth =
    connectors.length * LAYOUT_CONSTANTS.CONNECTOR_WIDTH;

  if (hasExpandButton) {
    connectorsColumnWidth += LAYOUT_CONSTANTS.CONNECTOR_WIDTH;
  }

  for (let i = 0; i < prevConnectors.length; i++) {
    if (
      prevConnectors[i] === 'empty' ||
      prevConnectors[i] === 'corner-top-right'
    ) {
      connectors[i] = 'empty';
    }
  }

  return {
    connectors,
    connectorsColumnWidth,
  };
};

const useSpanCardEventHandlers = (
  data: OtelSpanTree,
  onSpanSelect?: (span: OtelSpanTree) => void,
) => {
  const handleCardClick = useCallback((): void => {
    onSpanSelect?.(data);
  }, [data, onSpanSelect]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent): void => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        handleCardClick();
      }
    },
    [handleCardClick],
  );

  const handleToggleClick = useCallback(
    (e: MouseEvent | KeyboardEvent): void => {
      e.stopPropagation();
    },
    [],
  );

  return {
    handleCardClick,
    handleKeyDown,
    handleToggleClick,
  };
};

const SpanCardChildren: FC<{
  data: OtelSpanTree;
  level: number;
  selectedSpan?: OtelSpanTree;
  onSpanSelect?: (span: OtelSpanTree) => void;
  minStart: number;
  maxEnd: number;
  prevLevelConnectors: SpanCardConnectorType[];
  expandedSpansIds: string[];
  onExpandSpansIdsChange: (ids: string[]) => void;
  viewOptions?: SpanCardViewOptions;
}> = ({
  data,
  level,
  selectedSpan,
  onSpanSelect,
  minStart,
  maxEnd,
  prevLevelConnectors,
  expandedSpansIds,
  onExpandSpansIdsChange,
  viewOptions = DEFAULT_VIEW_OPTIONS,
}) => {
  if (!data.children?.length) {
    return null;
  }

  return (
    <div className="relative">
      <Collapsible.Content>
        <ul role="group">
          {data.children.map((child, idx) => (
            <SpanCard
              viewOptions={viewOptions}
              key={child.spanId}
              data={child}
              minStart={minStart}
              maxEnd={maxEnd}
              level={level + 1}
              selectedSpan={selectedSpan}
              onSpanSelect={onSpanSelect}
              isLastChild={idx === (data.children || []).length - 1}
              prevLevelConnectors={prevLevelConnectors}
              expandedSpansIds={expandedSpansIds}
              onExpandSpansIdsChange={onExpandSpansIdsChange}
            />
          ))}
        </ul>
      </Collapsible.Content>
    </div>
  );
};

export const SpanCard: FC<SpanCardProps> = ({
  data,
  level = 0,
  selectedSpan,
  onSpanSelect,
  viewOptions = DEFAULT_VIEW_OPTIONS,
  minStart,
  maxEnd,
  isLastChild,
  prevLevelConnectors = [],
  expandedSpansIds,
  onExpandSpansIdsChange,
}) => {
  const isExpanded = expandedSpansIds.includes(data.spanId);

  const expandButton =
    viewOptions.expandButton || DEFAULT_VIEW_OPTIONS.expandButton;

  const handleToggleClick = useCallback(
    (expanded: boolean) => {
      const alreadyExpanded = expandedSpansIds.includes(data.spanId);

      if (alreadyExpanded && !expanded) {
        onExpandSpansIdsChange(
          expandedSpansIds.filter((id) => id !== data.spanId),
        );
      }

      if (!alreadyExpanded && expanded) {
        onExpandSpansIdsChange([...expandedSpansIds, data.spanId]);
      }
    },
    [expandedSpansIds, data.spanId, onExpandSpansIdsChange],
  );

  const state: SpanCardState = {
    isExpanded,
    hasChildren: Boolean(data.children?.length),
    isSelected: selectedSpan?.spanId === data.spanId,
  };

  const eventHandlers = useSpanCardEventHandlers(data, onSpanSelect);

  const isStepRunSpan = data.spanName === 'hatchet.start_step_run';
  const stepName = isStepRunSpan
    ? data.spanAttributes?.['hatchet.step_name']
    : undefined;
  const hasStepLink =
    isStepRunSpan &&
    Boolean(onSpanSelect) &&
    Boolean(data.spanAttributes?.['hatchet.step_run_id']);

  const { durationMs } = getTimelineData({
    spanCard: data,
    minStart,
    maxEnd,
  });

  const hasExpandButtonAsFirstChild =
    expandButton === 'inside' && state.hasChildren;

  const contentPadding = getContentPadding({
    level,
    hasExpandButton: hasExpandButtonAsFirstChild,
  });

  const contentWidth = getContentWidth({
    level,
    hasExpandButton: hasExpandButtonAsFirstChild,
    contentPadding,
    expandButton,
  });

  const { connectors, connectorsColumnWidth } = getConnectorsLayout({
    level,
    hasExpandButton: hasExpandButtonAsFirstChild,
    isLastChild,
    prevConnectors: prevLevelConnectors,
    expandButton,
  });

  const gridTemplateColumns = getGridTemplateColumns({
    connectorsColumnWidth,
    expandButton,
  });

  return (
    <li
      role="treeitem"
      aria-selected={
        state.isSelected ? true : selectedSpan ? false : undefined
      }
      aria-expanded={state.hasChildren ? state.isExpanded : undefined}
      className="list-none"
    >
      <Collapsible.Root
        open={state.isExpanded}
        onOpenChange={handleToggleClick}
      >
        <div
          className={cn(
            'relative grid w-full',
            onSpanSelect && 'cursor-pointer',
            state.isSelected &&
              'before:bg-agentprism-muted/75 before:absolute before:-top-2 before:h-2 before:w-full',
            state.isSelected &&
              'from-agentprism-muted/75 to-agentprism-muted/75 bg-gradient-to-b',
          )}
          style={{
            gridTemplateColumns,
            backgroundSize: 'auto calc(100% - 8px)',
            backgroundPosition: 'top',
            backgroundRepeat: 'no-repeat',
          }}
          {...(onSpanSelect && {
            onClick: eventHandlers.handleCardClick,
            onKeyDown: eventHandlers.handleKeyDown,
            tabIndex: 0,
            role: 'button',
            'aria-pressed': state.isSelected,
            'aria-label': `${state.isSelected ? 'Selected' : 'Not selected'} span card for ${data.spanName} at level ${level}`,
          })}
          aria-describedby={`span-card-desc-${data.spanId}`}
          aria-expanded={state.hasChildren ? state.isExpanded : undefined}
        >
          <div className="flex flex-nowrap">
            {connectors.map((connector, idx) => (
              <SpanCardConnector
                key={`${connector}-${idx}`}
                type={connector}
              />
            ))}

            {hasExpandButtonAsFirstChild && (
              <div className="flex w-5 flex-col items-center">
                <SpanCardToggle
                  isExpanded={state.isExpanded}
                  title={data.spanName}
                  onToggleClick={eventHandlers.handleToggleClick}
                />

                {state.isExpanded && (
                  <SpanCardConnector type="vertical" />
                )}
              </div>
            )}
          </div>
          <div
            className={cn(
              'flex flex-wrap items-start gap-x-2 gap-y-1',
              'mb-3 min-h-5 w-full',
              level !== 0 && !hasExpandButtonAsFirstChild && 'pl-2',
              level !== 0 && hasExpandButtonAsFirstChild && 'pl-1',
            )}
          >
            <Collapsible.Trigger asChild disabled={!state.hasChildren}>
              <div
                className={cn(
                  'relative flex min-h-4 shrink-0 flex-wrap items-center gap-1.5',
                  state.hasChildren && 'cursor-pointer',
                )}
                style={{
                  width: `min(${contentWidth}px, 100%)`,
                }}
                onClick={eventHandlers.handleToggleClick}
              >
                <h3
                  className="text-agentprism-foreground truncate text-sm leading-[14px]"
                  title={data.spanName}
                >
                  {data.spanName}
                </h3>
                {stepName &&
                  (hasStepLink ? (
                    <span
                      className="cursor-pointer truncate text-xs text-muted-foreground underline decoration-dotted hover:text-foreground"
                      title={`View task run: ${stepName}`}
                      onClick={(e: MouseEvent) => {
                        e.stopPropagation();
                        onSpanSelect?.(data);
                      }}
                      role="link"
                    >
                      {stepName}
                    </span>
                  ) : (
                    <span className="truncate text-xs text-muted-foreground">
                      {stepName}
                    </span>
                  ))}
              </div>
            </Collapsible.Trigger>

            <div className="flex grow flex-wrap items-center justify-end gap-1">
              <SpanCardTimeline
                minStart={minStart}
                maxEnd={maxEnd}
                spanCard={data}
              />

              <div className="flex items-center gap-2">
                <span className="text-agentprism-foreground inline-block w-14 flex-1 shrink-0 whitespace-nowrap px-1 text-right text-xs">
                  {formatDuration(durationMs)}
                </span>
              </div>
            </div>
          </div>

          {expandButton === 'outside' &&
            (state.hasChildren ? (
              <SpanCardToggle
                isExpanded={state.isExpanded}
                title={data.spanName}
                onToggleClick={eventHandlers.handleToggleClick}
              />
            ) : (
              <div />
            ))}
        </div>

        <SpanCardChildren
          minStart={minStart}
          maxEnd={maxEnd}
          viewOptions={viewOptions}
          data={data}
          level={level}
          selectedSpan={selectedSpan}
          onSpanSelect={onSpanSelect}
          prevLevelConnectors={connectors}
          expandedSpansIds={expandedSpansIds}
          onExpandSpansIdsChange={onExpandSpansIdsChange}
        />
      </Collapsible.Root>
    </li>
  );
};
