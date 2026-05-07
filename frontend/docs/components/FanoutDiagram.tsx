import React from "react";

// ── Palette ──────────────────────────────────────────────────────────────────

const BLU = "rgb(51, 146, 255)";
const BLU_BG = "rgba(51, 146, 255, 0.15)";
const BLU_TXT = "#B8D9FF";
const BLU_SUB = "#3392FF";

const MAG = "rgb(188, 70, 221)";
const MAG_BG = "rgba(188, 70, 221, 0.18)";
const MAG_TXT = "#D585EF";

const TEL = "rgb(20, 184, 166)";
const TEL_BG = "rgba(20, 184, 166, 0.15)";
const TEL_TXT = "#5EEAD4";

const GRN = "rgb(34, 197, 94)";
const GRN_BG = "rgba(34, 197, 94, 0.25)";
const GRN_TXT = "#86EFAC";
const GRN_SUB = "#4ADE80";

const MUTED = "#64748B";
const ELB = "#8AA3BF"; // edge label text

// ── Shared SVG primitives ─────────────────────────────────────────────────────

/** A rounded rectangle with centered label (and optional sub-label). */
function SvgBox({
  x,
  y,
  w,
  h,
  rx = 10,
  bg,
  stroke,
  label,
  sub,
  lColor,
  sColor,
  fontSize = 13,
}: {
  x: number;
  y: number;
  w: number;
  h: number;
  rx?: number;
  bg: string;
  stroke: string;
  label: string;
  sub?: string;
  lColor: string;
  sColor?: string;
  fontSize?: number;
}) {
  const lines = label.split("\n");
  const lh = fontSize + 4;
  const totalTH = lines.length * lh + (sub ? lh - 1 : 0);
  const startY = y + (h - totalTH) / 2 + lh - 2;
  return (
    <g>
      <rect
        x={x}
        y={y}
        width={w}
        height={h}
        rx={rx}
        fill={bg}
        stroke={stroke}
        strokeWidth="1.5"
      />
      {lines.map((ln, i) => (
        <text
          key={i}
          x={x + w / 2}
          y={startY + i * lh}
          textAnchor="middle"
          fill={lColor}
          fontSize={fontSize}
          fontWeight="600"
        >
          {ln}
        </text>
      ))}
      {sub && (
        <text
          x={x + w / 2}
          y={startY + lines.length * lh}
          textAnchor="middle"
          fill={sColor ?? MUTED}
          fontSize={10}
        >
          {sub}
        </text>
      )}
    </g>
  );
}

/** Animated bezier edge with an optional label placed at 40% along the arc. */
function SvgEdge({
  sx,
  sy,
  tx,
  ty,
  stroke,
  delay = 0,
  label,
}: {
  sx: number;
  sy: number;
  tx: number;
  ty: number;
  stroke: string;
  delay?: number;
  label?: string;
}) {
  const dx = tx - sx;
  const d = `M ${sx} ${sy} C ${sx + dx * 0.55} ${sy}, ${tx - dx * 0.55} ${ty}, ${tx} ${ty}`;
  // place label near the midpoint of the curve, slightly above
  const lx = sx + dx * 0.45;
  const ly = sy + (ty - sy) * 0.45 - 6;
  return (
    <g>
      <path
        d={d}
        fill="none"
        stroke={stroke}
        strokeWidth="1.5"
        strokeOpacity="0.75"
        strokeDasharray="8 6"
        className="fd-flow"
        style={{ animationDelay: `${delay}s` }}
      />
      {label && (
        <text x={lx} y={ly} textAnchor="middle" fill={ELB} fontSize="9">
          {label}
        </text>
      )}
    </g>
  );
}

// ── Types ─────────────────────────────────────────────────────────────────────

export interface LeafNode {
  /** Box label (use "\n" for line breaks). */
  label: string;
  subLabel?: string;
  /** Text drawn along the edge from the mid node to this leaf. */
  edgeLabel?: string;
}

export interface MidNode {
  /** Box label. */
  label: string;
  subLabel?: string;
  /** Text drawn along the edge from the parent to this mid node. */
  edgeLabel?: string;
  /**
   * Children of this mid node.  When omitted the mid node is a terminal
   * child (no second-level fan-out from it).
   */
  leaves?: LeafNode[];
}

export interface FanoutDiagramProps {
  // ── Both variants ─────────────────────────────────────────────────────────
  /** Label for the leftmost (trigger / parent) node. */
  parentLabel?: string;
  /** Small sub-label inside the parent box. */
  parentSubLabel?: string;

  // ── Simple (1-level) variant ──────────────────────────────────────────────
  /**
   * Labels for fanned-out child nodes.
   * The last entry is always rendered as the "…, Child N" slot.
   * Ignored when `midNodes` is provided.
   */
  childLabels?: string[];
  /**
   * Show the right-hand "Collect Results" box that fans back in.
   * Defaults to `true`.  Ignored when `midNodes` is provided.
   */
  showCollect?: boolean;
  collectLabel?: string;
  collectSubLabel?: string;

  // ── Nested (2-level) variant ──────────────────────────────────────────────
  /**
   * When provided the component renders a **two-level fan-out** instead of
   * the simple variant.
   *
   * Each entry is an intermediate node that may itself fan out to leaf nodes
   * via its `leaves` array.  Mid nodes without `leaves` are rendered as
   * terminal children (no second-level fan-out shown from them).
   *
   * @example
   * ```tsx
   * <FanoutDiagram
   *   parentLabel="Fetch Buckets"
   *   midNodes={[
   *     { label: "List objects", edgeLabel: "bucket-0", leaves: [
   *       { label: "Download +\nprocess", edgeLabel: "bucket-0/object-0.png" },
   *       { label: "Download +\nprocess", edgeLabel: "bucket-0/object-1.png" },
   *     ]},
   *     { label: "List objects", edgeLabel: "bucket-N" },
   *   ]}
   * />
   * ```
   */
  midNodes?: MidNode[];
}

// ── CSS shared by both SVGs ───────────────────────────────────────────────────

const ANIM_CSS = `
  @keyframes fd-dash { to { stroke-dashoffset: -20; } }
  .fd-flow { stroke-dasharray: 8 6; animation: fd-dash 0.8s linear infinite; }
`;

// ── Simple (1-level) layout ───────────────────────────────────────────────────

function SimpleFanout({
  parentLabel,
  parentSubLabel,
  childLabels,
  showCollect,
  collectLabel,
  collectSubLabel,
}: Required<
  Pick<
    FanoutDiagramProps,
    | "parentLabel"
    | "parentSubLabel"
    | "childLabels"
    | "showCollect"
    | "collectLabel"
    | "collectSubLabel"
  >
>) {
  const childYs = [40, 110, 180, 270];
  // parent vertical midpoint in the 330-high viewBox
  const pMidY = 165;

  const viewW = showCollect ? 800 : 510;

  return (
    <svg
      viewBox={`0 0 ${viewW} 330`}
      className="w-full h-auto"
      xmlns="http://www.w3.org/2000/svg"
    >
      <style>{ANIM_CSS}</style>

      {/* Parent */}
      <SvgBox
        x={20}
        y={125}
        w={170}
        h={80}
        bg={BLU_BG}
        stroke={BLU}
        label={parentLabel}
        sub={parentSubLabel}
        lColor={BLU_TXT}
        sColor={BLU_SUB}
      />

      {/* Fan-out edges parent → children */}
      {childYs.map((cy, i) => (
        <SvgEdge
          key={i}
          sx={190}
          sy={pMidY}
          tx={320}
          ty={cy + 25}
          stroke={BLU}
          delay={i * 0.15}
        />
      ))}

      {/* Child boxes (first 3 explicit + ellipsis + Child N) */}
      {childLabels.slice(0, 3).map((lbl, i) => (
        <SvgBox
          key={i}
          x={320}
          y={childYs[i]}
          w={150}
          h={50}
          bg={MAG_BG}
          stroke={MAG}
          label={lbl}
          lColor={MAG_TXT}
        />
      ))}
      <text
        x="395"
        y="255"
        textAnchor="middle"
        fill={MUTED}
        fontSize="18"
        fontWeight="bold"
      >
        ...
      </text>
      <SvgBox
        x={320}
        y={childYs[3]}
        w={150}
        h={50}
        bg={MAG_BG}
        stroke={MAG}
        label={childLabels[childLabels.length - 1] ?? "Child N"}
        lColor={MAG_TXT}
      />

      {/* Converge edges children → collect (only when showCollect) */}
      {showCollect &&
        childYs.map((cy, i) => (
          <SvgEdge
            key={i}
            sx={470}
            sy={cy + 25}
            tx={600}
            ty={pMidY}
            stroke={GRN}
            delay={i * 0.15 + 0.4}
          />
        ))}

      {/* Collect box */}
      {showCollect && (
        <SvgBox
          x={600}
          y={125}
          w={180}
          h={80}
          bg={GRN_BG}
          stroke={GRN}
          label={collectLabel}
          sub={collectSubLabel}
          lColor={GRN_TXT}
          sColor={GRN_SUB}
        />
      )}
    </svg>
  );
}

// ── Nested (2-level) layout ───────────────────────────────────────────────────

// Box dimensions
const P_W = 160,
  P_H = 72; // parent
const M_W = 130,
  M_H = 60; // mid
const L_W = 150,
  L_H = 55; // leaf

// Column left-edge X positions
const P_X = 28;
const M_X = 330;
const L_X = 620;

const SLOT_H = 85; // vertical space per slot (leaf-to-leaf centre spacing)
const TOP_PAD = 40;

function NestedFanout({
  parentLabel,
  parentSubLabel,
  midNodes,
}: {
  parentLabel: string;
  parentSubLabel?: string;
  midNodes: MidNode[];
}) {
  // ── Slot assignment ───────────────────────────────────────────────────────
  // Each leaf → one slot; each leaf-less mid → one slot.
  type Slot =
    | { kind: "leaf"; midIdx: number; leafIdx: number; leaf: LeafNode }
    | { kind: "mid"; midIdx: number };

  const slots: Slot[] = [];
  midNodes.forEach((mid, mi) => {
    if (mid.leaves && mid.leaves.length > 0) {
      mid.leaves.forEach((leaf, li) =>
        slots.push({ kind: "leaf", midIdx: mi, leafIdx: li, leaf }),
      );
    } else {
      slots.push({ kind: "mid", midIdx: mi });
    }
  });

  // ── Y-position helpers ────────────────────────────────────────────────────
  const slotTopY = (i: number) => TOP_PAD + i * SLOT_H;
  const leafCenterY = (i: number) => slotTopY(i) + L_H / 2;
  const midOnlyCY = (i: number) => slotTopY(i) + M_H / 2;

  // Build lookup: midIdx → slot indices
  const leafSlotIdx: number[][] = midNodes.map(() => []);
  const midOnlySlotIdx: number[] = midNodes.map(() => -1);
  slots.forEach((s, i) => {
    if (s.kind === "leaf") leafSlotIdx[s.midIdx].push(i);
    else midOnlySlotIdx[s.midIdx] = i;
  });

  // Mid node vertical centres
  const midCY = midNodes.map((_mid, mi) => {
    const ls = leafSlotIdx[mi];
    if (ls.length > 0) {
      return (leafCenterY(ls[0]) + leafCenterY(ls[ls.length - 1])) / 2;
    }
    return midOnlyCY(midOnlySlotIdx[mi]);
  });

  const parentCY =
    midCY.length > 0
      ? (midCY[0] + midCY[midCY.length - 1]) / 2
      : TOP_PAD + P_H / 2;

  const svgH = slotTopY(slots.length) + Math.max(L_H, M_H) / 2 + TOP_PAD;
  const svgW = 820;

  // ── Render ────────────────────────────────────────────────────────────────
  let delay = 0;
  const nd = (inc = 0.12) => {
    delay += inc;
    return delay;
  };

  return (
    <svg
      viewBox={`0 0 ${svgW} ${svgH}`}
      className="w-full h-auto"
      xmlns="http://www.w3.org/2000/svg"
    >
      <style>{ANIM_CSS}</style>

      {/* Parent box */}
      <SvgBox
        x={P_X}
        y={parentCY - P_H / 2}
        w={P_W}
        h={P_H}
        bg={BLU_BG}
        stroke={BLU}
        label={parentLabel}
        sub={parentSubLabel}
        lColor={BLU_TXT}
        sColor={BLU_SUB}
      />

      {midNodes.map((mid, mi) => {
        const mcy = midCY[mi];
        const ls = leafSlotIdx[mi];

        return (
          <g key={mi}>
            {/* Edge: parent → mid */}
            <SvgEdge
              sx={P_X + P_W}
              sy={parentCY}
              tx={M_X}
              ty={mcy}
              stroke={BLU}
              delay={nd()}
              label={mid.edgeLabel}
            />

            {/* Mid box */}
            <SvgBox
              x={M_X}
              y={mcy - M_H / 2}
              w={M_W}
              h={M_H}
              bg={MAG_BG}
              stroke={MAG}
              label={mid.label}
              sub={mid.subLabel}
              lColor={MAG_TXT}
            />

            {/* Edges mid → leaves + leaf boxes */}
            {ls.map((si, li) => {
              const lcy = leafCenterY(si);
              const leaf = (mid.leaves ?? [])[li];
              return (
                <g key={li}>
                  <SvgEdge
                    sx={M_X + M_W}
                    sy={mcy}
                    tx={L_X}
                    ty={lcy}
                    stroke={TEL}
                    delay={nd()}
                    label={leaf.edgeLabel}
                  />
                  <SvgBox
                    x={L_X}
                    y={lcy - L_H / 2}
                    w={L_W}
                    h={L_H}
                    bg={TEL_BG}
                    stroke={TEL}
                    label={leaf.label}
                    sub={leaf.subLabel}
                    lColor={TEL_TXT}
                  />
                </g>
              );
            })}
          </g>
        );
      })}
    </svg>
  );
}

// ── Public component ──────────────────────────────────────────────────────────

const FanoutDiagram: React.FC<FanoutDiagramProps> = ({
  parentLabel = "Parent Task",
  parentSubLabel = "spawn(input)",
  childLabels = ["Child 1", "Child 2", "Child 3", "Child N"],
  showCollect = true,
  collectLabel = "Collect Results",
  collectSubLabel = "await all children",
  midNodes,
}) => (
  <div className="my-8 flex justify-center">
    <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
      {midNodes ? (
        <NestedFanout
          parentLabel={parentLabel}
          parentSubLabel={parentSubLabel}
          midNodes={midNodes}
        />
      ) : (
        <SimpleFanout
          parentLabel={parentLabel}
          parentSubLabel={parentSubLabel}
          childLabels={childLabels}
          showCollect={showCollect}
          collectLabel={collectLabel}
          collectSubLabel={collectSubLabel}
        />
      )}
    </div>
  </div>
);

export default FanoutDiagram;
