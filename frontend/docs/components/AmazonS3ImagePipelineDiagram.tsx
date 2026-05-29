import React, { useState, useEffect } from "react";

const PER_KEY_CAP = 20;
const WORKER_SLOTS = 65;
const COLS = 10;
const BUCKET_ROWS = 2;
const CELL_SIZE = 26;
const CELL_GAP = 6;
const INITIAL_S3_BACKLOG = 200;

type SlotState = "processing" | "done" | "empty";

interface BucketBehavior {
  name: string;
  fillRate: number;
  completeRate: number;
  drainRate: number;
  initialClaimed: number;
  pullsFromGlobal: boolean;
}

interface BucketState {
  name: string;
  slots: SlotState[];
}

interface AppState {
  buckets: BucketState[];
  s3Backlog: number;
  status: "running" | "finished";
}

const BUCKET_BEHAVIORS: BucketBehavior[] = [
  {
    name: "bucket-0",
    fillRate: 0.55,
    completeRate: 0.35,
    drainRate: 0.55,
    initialClaimed: 0,
    pullsFromGlobal: true,
  },
  {
    name: "bucket-1",
    fillRate: 0.65,
    completeRate: 0.35,
    drainRate: 0.55,
    initialClaimed: 0,
    pullsFromGlobal: true,
  },
  {
    name: "bucket-2",
    fillRate: 0.5,
    completeRate: 0.35,
    drainRate: 0.55,
    initialClaimed: 0,
    pullsFromGlobal: true,
  },
];

const BUCKET_COLOR = "#3392FF";
const TEXT_COLOR = "#A5C5E9";
const PENDING_COLOR = "#5F5E5A";
const PROCESSING_COLOR = "#EAB308";
const DONE_COLOR = "#22C55E";

const Spinner: React.FC<{ x: number; y: number; color: string }> = ({
  x,
  y,
  color,
}) => (
  <g transform={`translate(${x - 5}, ${y - 5})`}>
    <svg
      width="10"
      height="10"
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="3"
      strokeLinecap="round"
    >
      <path d="M21 12a9 9 0 1 1-6.219-8.56">
        <animateTransform
          attributeName="transform"
          type="rotate"
          from="0 12 12"
          to="360 12 12"
          dur="1s"
          repeatCount="indefinite"
        />
      </path>
    </svg>
  </g>
);

const CheckIcon: React.FC<{ x: number; y: number; color: string }> = ({
  x,
  y,
  color,
}) => (
  <g transform={`translate(${x - 5}, ${y - 5})`}>
    <svg
      width="10"
      height="10"
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="3"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <polyline points="20 6 9 17 4 12" />
    </svg>
  </g>
);

const BucketIcon: React.FC<{ x: number; y: number; color: string }> = ({
  x,
  y,
  color,
}) => (
  <g transform={`translate(${x - 9}, ${y - 9})`}>
    <svg
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <ellipse cx="12" cy="5" rx="9" ry="3" />
      <path d="M3 5v14a9 3 0 0 0 18 0V5" />
      <path d="M3 12a9 3 0 0 0 18 0" />
    </svg>
  </g>
);

const WorkerIcon: React.FC<{ x: number; y: number; color: string }> = ({
  x,
  y,
  color,
}) => (
  <g transform={`translate(${x - 10}, ${y - 10})`}>
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="2" y="4" width="20" height="16" rx="2" ry="2"></rect>
      <line x1="6" y1="8" x2="6.01" y2="8"></line>
      <line x1="6" y1="12" x2="6.01" y2="12"></line>
      <line x1="6" y1="16" x2="6.01" y2="16"></line>
    </svg>
  </g>
);

const DatabaseIcon: React.FC<{ x: number; y: number; color: string }> = ({
  x,
  y,
  color,
}) => (
  <g transform={`translate(${x - 10}, ${y - 10})`}>
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <ellipse cx="12" cy="5" rx="9" ry="3"></ellipse>
      <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path>
      <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>
    </svg>
  </g>
);

function getInitialState(): AppState {
  return {
    buckets: BUCKET_BEHAVIORS.map((b) => {
      const slots: SlotState[] = Array(PER_KEY_CAP).fill("empty");
      for (let i = 0; i < b.initialClaimed; i++) slots[i] = "processing";
      return { name: b.name, slots };
    }),
    s3Backlog: INITIAL_S3_BACKLOG,
    status: "running",
  };
}

function advanceState(prevState: AppState): AppState {
  if (prevState.status === "finished") return prevState;

  let currentBacklog = prevState.s3Backlog;

  // Pass 1: Drain & Complete
  const nextBuckets = prevState.buckets.map((bucket, idx) => {
    const behavior = BUCKET_BEHAVIORS[idx];
    const newSlots = [...bucket.slots];

    for (let i = 0; i < newSlots.length; i++) {
      if (newSlots[i] === "done" && Math.random() < behavior.drainRate) {
        newSlots[i] = "empty";
      }
      if (
        newSlots[i] === "processing" &&
        Math.random() < behavior.completeRate
      ) {
        newSlots[i] = "done";
      }
    }
    return { ...bucket, slots: newSlots };
  });

  // Calculate free worker slots globally BEFORE filling
  let totalActive = nextBuckets.reduce(
    (sum, b) => sum + b.slots.filter((s) => s === "processing").length,
    0,
  );
  let freeWorkerSlots = WORKER_SLOTS - totalActive;

  // We randomize bucket order slightly so one doesn't always steal all the global capacity
  const bucketIndices = Array.from(
    { length: nextBuckets.length },
    (_, i) => i,
  ).sort(() => Math.random() - 0.5);

  // Pass 2: Fill from global backlog, strictly respecting the worker pool limit
  for (const b of bucketIndices) {
    const behavior = BUCKET_BEHAVIORS[b];
    const slots = nextBuckets[b].slots;

    for (let i = 0; i < slots.length; i++) {
      if (
        slots[i] === "empty" &&
        behavior.pullsFromGlobal &&
        currentBacklog > 0 &&
        freeWorkerSlots > 0
      ) {
        if (Math.random() < behavior.fillRate) {
          slots[i] = "processing";
          currentBacklog--;
          freeWorkerSlots--;
          totalActive++;
        }
      }
    }
  }

  const totalClaimed = nextBuckets.reduce(
    (sum, b) => sum + b.slots.filter((s) => s !== "empty").length,
    0,
  );

  if (currentBacklog === 0 && totalClaimed === 0) {
    return { buckets: nextBuckets, s3Backlog: 0, status: "finished" };
  }

  return { buckets: nextBuckets, s3Backlog: currentBacklog, status: "running" };
}

const MultiBucketSlotPoolDiagram: React.FC = () => {
  const [appState, setAppState] = useState<AppState>(getInitialState);
  // Incrementing this key restarts the interval with its initial pause,
  // so every cycle (first load + after reset) shows 200/0 for ~1.2s before ticking.
  const [cycleKey, setCycleKey] = useState(0);

  useEffect(() => {
    let id: NodeJS.Timeout;
    const delay = setTimeout(() => {
      id = setInterval(() => {
        setAppState((prev) => advanceState(prev));
      }, 700);
    }, 1200);
    return () => {
      clearTimeout(delay);
      clearInterval(id);
    };
  }, [cycleKey]);

  useEffect(() => {
    if (appState.status !== "finished") return;
    const timer = setTimeout(() => {
      setAppState(getInitialState());
      setCycleKey((k) => k + 1);
    }, 2000);
    return () => clearTimeout(timer);
  }, [appState.status]);

  const totalWidth = 500;
  const padding = 12;
  const bucketWidth = 92;
  const bucketHeight = 48;
  const bucketGap = 20;

  const regionWidth = COLS * CELL_SIZE + (COLS - 1) * CELL_GAP;
  const regionHeight = BUCKET_ROWS * CELL_SIZE + (BUCKET_ROWS - 1) * CELL_GAP;
  const regionPadding = 6;
  const regionBoxWidth = regionWidth + regionPadding * 2;
  const regionBoxHeight = regionHeight + regionPadding * 2;
  const regionVerticalGap = 16;

  const contentWidth = bucketWidth + bucketGap + regionBoxWidth;
  const paddingX = (totalWidth - contentWidth) / 2;

  const regionX = paddingX + bucketWidth + bucketGap;
  const firstRegionY = padding;
  const bottomAreaY =
    firstRegionY +
    appState.buckets.length * regionBoxHeight +
    (appState.buckets.length - 1) * regionVerticalGap +
    16;

  const bottomSliderHeight = 12;
  // worker section content ends ~40px below bottomAreaY; leave 22px gap + divider
  const s3SectionY = bottomAreaY + 62;

  const totalHeight = s3SectionY + bottomSliderHeight + padding + 32;

  // Aggregate counters
  const totalActive = appState.buckets.reduce(
    (sum, b) => sum + b.slots.filter((s) => s === "processing").length,
    0,
  );

  const bottomSliderWidth = contentWidth;
  const bottomSliderX = paddingX;

  // Worker Capacity Logic
  const workerActivePx = (totalActive / WORKER_SLOTS) * bottomSliderWidth;
  const workerAtCap = totalActive >= WORKER_SLOTS;

  // S3 Backlog draining logic
  const backlogPx =
    (appState.s3Backlog / INITIAL_S3_BACKLOG) * bottomSliderWidth;

  return (
    <div
      className="my-4 rounded-xl border p-3"
      style={{
        borderColor: "rgba(51, 146, 255, 0.2)",
        backgroundColor: "rgba(10, 16, 41, 0.04)",
      }}
    >
      <svg
        viewBox={`0 0 ${totalWidth} ${totalHeight}`}
        className="mx-auto w-full"
        style={{ maxWidth: totalWidth }}
      >
        <defs>
          <marker
            id="bucket-arrow"
            viewBox="0 0 10 10"
            refX="8"
            refY="5"
            markerWidth="6"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <path
              d="M2 1L8 5L2 9"
              fill="none"
              stroke={BUCKET_COLOR}
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </marker>
        </defs>

        {appState.buckets.map((bucketState, bIdx) => {
          const regionY =
            firstRegionY + bIdx * (regionBoxHeight + regionVerticalGap);
          const regionMidY = regionY + regionBoxHeight / 2;
          const sourceY = regionMidY - bucketHeight / 2;
          const sourceX = paddingX;

          const activeInBucket = bucketState.slots.filter(
            (s) => s === "processing",
          ).length;
          const isAtBucketCap = activeInBucket >= PER_KEY_CAP;

          return (
            <g
              key={bucketState.name}
              style={{ transition: "opacity 0.8s ease" }}
            >
              <rect
                x={sourceX}
                y={sourceY}
                width={bucketWidth}
                height={bucketHeight}
                rx={8}
                fill={`${BUCKET_COLOR}15`}
                stroke={BUCKET_COLOR}
                strokeWidth={1.5}
              />
              <text
                x={sourceX + bucketWidth / 2}
                y={sourceY + 28}
                textAnchor="middle"
                fontSize="11"
                fontWeight={600}
                fill={BUCKET_COLOR}
              >
                {bucketState.name}
              </text>

              <line
                x1={sourceX + bucketWidth}
                y1={regionMidY}
                x2={regionX - 2}
                y2={regionMidY}
                stroke={BUCKET_COLOR}
                strokeWidth={1.5}
                markerEnd="url(#bucket-arrow)"
              />

              <rect
                x={regionX}
                y={regionY}
                width={regionBoxWidth}
                height={regionBoxHeight}
                rx={8}
                fill={isAtBucketCap ? PROCESSING_COLOR : BUCKET_COLOR}
                fillOpacity={0.04}
                stroke={isAtBucketCap ? PROCESSING_COLOR : BUCKET_COLOR}
                strokeWidth={1.2}
                strokeDasharray={isAtBucketCap ? "none" : "4 3"}
                opacity={0.6}
                style={{ transition: "all 0.4s ease" }}
              />

              {bucketState.slots.map((state, slotIdx) => {
                const col = slotIdx % COLS;
                const row = Math.floor(slotIdx / COLS);
                const x =
                  regionX + regionPadding + col * (CELL_SIZE + CELL_GAP);
                const y =
                  regionY + regionPadding + row * (CELL_SIZE + CELL_GAP);

                if (state === "empty") {
                  return (
                    <g key={slotIdx}>
                      <rect
                        x={x}
                        y={y}
                        width={CELL_SIZE}
                        height={CELL_SIZE}
                        rx={4}
                        fill="#0A1029"
                        stroke={PENDING_COLOR}
                        strokeWidth={1}
                        strokeDasharray="2 2"
                        opacity={0.6}
                        style={{ transition: "all 0.4s ease" }}
                      />
                    </g>
                  );
                }

                const color = state === "done" ? DONE_COLOR : PROCESSING_COLOR;

                return (
                  <g key={slotIdx}>
                    {state === "processing" && (
                      <rect
                        x={x - 2}
                        y={y - 2}
                        width={CELL_SIZE + 4}
                        height={CELL_SIZE + 4}
                        rx={6}
                        fill={color}
                        opacity={0.18}
                      >
                        <animate
                          attributeName="opacity"
                          values="0.18;0.08;0.18"
                          dur="0.9s"
                          repeatCount="indefinite"
                        />
                      </rect>
                    )}
                    <rect
                      x={x}
                      y={y}
                      width={CELL_SIZE}
                      height={CELL_SIZE}
                      rx={4}
                      fill={`${color}20`}
                      stroke={color}
                      strokeWidth={2}
                      style={{ transition: "all 0.4s ease" }}
                    />
                    {state === "processing" ? (
                      <Spinner
                        x={x + CELL_SIZE / 2}
                        y={y + CELL_SIZE / 2}
                        color={color}
                      />
                    ) : (
                      <CheckIcon
                        x={x + CELL_SIZE / 2}
                        y={y + CELL_SIZE / 2}
                        color={color}
                      />
                    )}
                  </g>
                );
              })}
            </g>
          );
        })}

        {/* Global Worker Pool Capacity Slider */}
        <g transform={`translate(0, ${bottomAreaY})`}>
          <WorkerIcon x={bottomSliderX - 18} y={4} color={PROCESSING_COLOR} />
          <text
            x={bottomSliderX + 4}
            y={-2}
            fontSize="11"
            fontWeight={600}
            fill={TEXT_COLOR}
          >
            Hatchet Worker Pool Allocation
          </text>

          <rect
            x={bottomSliderX}
            y={12}
            width={bottomSliderWidth}
            height={bottomSliderHeight}
            rx={bottomSliderHeight / 2}
            fill="rgba(10, 16, 41, 0.6)"
            stroke={PENDING_COLOR}
            strokeWidth={0.5}
          />
          <rect
            x={bottomSliderX}
            y={12}
            width={workerActivePx}
            height={bottomSliderHeight}
            rx={bottomSliderHeight / 2}
            fill={PROCESSING_COLOR}
            style={{ transition: "all 0.4s ease" }}
          />

          <text
            x={bottomSliderX}
            y={bottomSliderHeight + 28}
            fontSize="11"
            fill={PENDING_COLOR}
          >
            {workerAtCap
              ? "Worker at capacity • Tasks queuing normally in broker"
              : ""}
          </text>

          <text
            x={bottomSliderX + bottomSliderWidth}
            y={bottomSliderHeight + 28}
            textAnchor="end"
            fontSize="11"
            fill={TEXT_COLOR}
          >
            <tspan fontWeight={700} fill={PROCESSING_COLOR}>
              {totalActive}
            </tspan>
            <tspan> / {WORKER_SLOTS} slots</tspan>
          </text>
        </g>

        {/* Horizontal Divider */}
        <line
          x1={paddingX}
          y1={s3SectionY - 20}
          x2={totalWidth - paddingX}
          y2={s3SectionY - 20}
          stroke={PENDING_COLOR}
          strokeOpacity={0.3}
          strokeWidth={1}
          strokeDasharray="4 4"
        />

        {/* Global S3 Backlog Progress Slider */}
        <g transform={`translate(0, ${s3SectionY})`}>
          <DatabaseIcon x={bottomSliderX - 18} y={4} color={BUCKET_COLOR} />
          <text
            x={bottomSliderX + 4}
            y={-2}
            fontSize="11"
            fontWeight={600}
            fill={TEXT_COLOR}
          >
            Task Backlog (Amazon S3 Objects)
          </text>

          {/* Background Track */}
          <rect
            x={bottomSliderX}
            y={12}
            width={bottomSliderWidth}
            height={bottomSliderHeight}
            rx={bottomSliderHeight / 2}
            fill="rgba(10, 16, 41, 0.6)"
            stroke={PENDING_COLOR}
            strokeWidth={0.5}
          />

          {/* Active Backlog Drain */}
          <rect
            x={bottomSliderX}
            y={12}
            width={backlogPx}
            height={bottomSliderHeight}
            rx={bottomSliderHeight / 2}
            fill={BUCKET_COLOR}
            fillOpacity={0.8}
            style={{ transition: "width 0.4s ease" }}
          />

          <text
            x={bottomSliderX}
            y={bottomSliderHeight + 28}
            fontSize="11"
            fill={appState.status === "finished" ? DONE_COLOR : PENDING_COLOR}
          >
            {appState.status === "finished"
              ? "Queue Depleted."
              : "Polling S3 paginator & queuing child workflows"}
          </text>

          <text
            x={bottomSliderX + bottomSliderWidth}
            y={bottomSliderHeight + 28}
            textAnchor="end"
            fontSize="11"
            fill={TEXT_COLOR}
          >
            <tspan fontWeight={700} fill={BUCKET_COLOR}>
              {appState.s3Backlog}
            </tspan>
            <tspan> / {INITIAL_S3_BACKLOG} remaining</tspan>
          </text>
        </g>
      </svg>
    </div>
  );
};

export default MultiBucketSlotPoolDiagram;
