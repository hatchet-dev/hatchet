import React, { useState, useEffect } from "react";
import { brand, state, fill, inactive, gradient } from "./diagram-colors";

const TaskEvictionDiagram: React.FC = () => {
  // Phases:
  // 0=running, 1=waiting, 2=evicted (slot freed), 3=worker picks up other work,
  // 4=wait satisfied -> resumed (event log replayed), 5=complete
  const [phase, setPhase] = useState(0);

  useEffect(() => {
    const durations = [1500, 1500, 1500, 1800, 1800, 1500];
    const timer = setTimeout(() => {
      setPhase((prev) => (prev + 1) % 6);
    }, durations[phase]);
    return () => clearTimeout(timer);
  }, [phase]);

  const nodeW = 120;
  const nodeH = 44;
  const rx = 8;
  const y = 40;

  // The top flow has 5 nodes; phase 3 (other work) keeps "Evicted" active.
  const activeNode = phase <= 2 ? phase : phase - 1;

  const steps = [
    { x: 30, label: "Task Runs", sub: "holds slot" },
    { x: 175, label: "Hits Wait", sub: "sleep / event / child" },
    { x: 320, label: "Evicted", sub: "slot freed" },
    { x: 465, label: "Resumed", sub: "replays event log" },
    { x: 610, label: "Complete", sub: "return result" },
  ];

  const phaseColors = [
    {
      fill: fill.activeNode,
      stroke: brand.blue,
      text: brand.cyan,
      sub: brand.blue,
    },
    {
      fill: fill.running,
      stroke: state.running,
      text: state.runningLight,
      sub: state.runningDark,
    },
    {
      fill: fill.magenta,
      stroke: brand.magenta,
      text: brand.magentaLight,
      sub: brand.magenta,
    },
    {
      fill: fill.success,
      stroke: state.success,
      text: state.successLighter,
      sub: state.successLight,
    },
    {
      fill: fill.success,
      stroke: state.success,
      text: state.successLighter,
      sub: state.successLight,
    },
  ];

  const statusLabels = [
    "running — task holds a worker slot",
    "waiting — sleep / event / child run",
    "evicted — slot freed",
    "worker picks up other work",
    "wait satisfied — replaying event log",
    "complete!",
  ];
  const statusColors = [
    brand.blue,
    state.runningLight,
    brand.magentaLight,
    brand.cyan,
    state.successLight,
    state.successLight,
  ];

  // Worker box + slots
  const workerX = 250;
  const workerY = 150;
  const workerW = 260;
  const workerH = 96;
  const slotW = 110;
  const slotH = 38;
  const slotY = workerY + 46;
  const slot1X = workerX + 14;
  const slot2X = workerX + workerW - slotW - 14;

  const isEvicting = phase === 2;
  const isResuming = phase === 4;

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      {/* Status bar */}
      <div className="flex items-center gap-3 rounded-lg border border-neutral-700/40 bg-neutral-900/50 px-4 py-2">
        <span className="text-xs font-medium text-gray-400">Durable task:</span>
        {statusLabels.map((_, i) => (
          <span
            key={i}
            className="flex h-2 w-2 rounded-full transition-all duration-300"
            style={{
              backgroundColor:
                i < phase
                  ? state.success
                  : i === phase
                    ? statusColors[i]
                    : inactive.dot,
              boxShadow: i === phase ? `0 0 6px ${statusColors[i]}` : "none",
            }}
          />
        ))}
        <span
          className="ml-1 text-xs font-medium transition-colors duration-300"
          style={{ color: statusColors[phase] }}
        >
          {statusLabels[phase]}
        </span>
      </div>

      {/* Diagram */}
      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <svg
          viewBox="0 0 760 270"
          className="w-full h-auto"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient id="te-blue" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop
                offset="0%"
                stopColor={gradient.blue[0]}
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor={gradient.blue[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="te-yellow" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop
                offset="0%"
                stopColor={gradient.yellow[0]}
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor={gradient.yellow[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="te-magenta" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop
                offset="0%"
                stopColor={gradient.magenta[0]}
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor={gradient.magenta[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="te-green" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop
                offset="0%"
                stopColor={gradient.green[0]}
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor={gradient.green[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
          </defs>

          <style>{`
            @keyframes te-dash {
              to { stroke-dashoffset: -20; }
            }
            .te-flow {
              stroke-dasharray: 8 6;
              animation: te-dash 0.8s linear infinite;
            }
            @keyframes te-pulse {
              0%, 100% { opacity: 0.35; }
              50% { opacity: 1; }
            }
            .te-pulse {
              animation: te-pulse 1.4s ease-in-out infinite;
            }
          `}</style>

          {/* Edges between top nodes */}
          {steps.slice(0, -1).map((s, i) => {
            const nextX = steps[i + 1].x;
            const isCurrent = i === activeNode && phase !== 3;

            let edgeColor: string = inactive.edge;
            if (i < activeNode) {
              edgeColor = phaseColors[i].stroke;
            }
            if (isCurrent) {
              edgeColor = phaseColors[i].stroke;
            }

            return (
              <path
                key={`edge-${i}`}
                d={`M ${s.x + nodeW + 2} ${y + nodeH / 2} L ${nextX - 2} ${y + nodeH / 2}`}
                fill="none"
                stroke={edgeColor}
                strokeWidth="2"
                className={isCurrent ? "te-flow" : ""}
                style={{
                  opacity: i <= activeNode ? 1 : 0.15,
                  transition: "opacity 0.4s ease, stroke 0.4s ease",
                }}
              />
            );
          })}

          {/* Top flow nodes */}
          {steps.map((s, i) => {
            const isActive = i === activeNode;
            const isPast = i < activeNode;

            let nodeFill: string = fill.inactiveNode;
            let stroke: string = inactive.edge;
            let textColor: string = inactive.text;
            let subColor: string = inactive.stroke;

            if (isActive || isPast) {
              nodeFill = phaseColors[i].fill;
              stroke = phaseColors[i].stroke;
              textColor = phaseColors[i].text;
              subColor = phaseColors[i].sub;
            }

            return (
              <g key={`node-${i}`}>
                <rect
                  x={s.x}
                  y={y}
                  width={nodeW}
                  height={nodeH}
                  rx={rx}
                  fill={nodeFill}
                  stroke={stroke}
                  strokeWidth={isActive ? "2" : "1.5"}
                  strokeDasharray={
                    (i === 1 || i === 2) && (isActive || isPast)
                      ? "4 3"
                      : "none"
                  }
                  style={{
                    transition: "all 0.4s ease",
                    opacity:
                      isPast && !isActive ? 0.7 : i > activeNode ? 0.3 : 1,
                  }}
                />
                <text
                  x={s.x + nodeW / 2}
                  y={y + nodeH / 2 - 3}
                  textAnchor="middle"
                  fill={textColor}
                  fontSize="12"
                  fontWeight="500"
                  style={{ transition: "fill 0.4s ease" }}
                >
                  {s.label}
                </text>
                <text
                  x={s.x + nodeW / 2}
                  y={y + nodeH / 2 + 12}
                  textAnchor="middle"
                  fill={subColor}
                  fontSize="9"
                  style={{ transition: "fill 0.4s ease" }}
                >
                  {s.sub}
                </text>

                {/* Pause icon above Hits Wait */}
                {i === 1 && phase === 1 && (
                  <g className="te-pulse">
                    <rect
                      x={s.x + nodeW / 2 - 8}
                      y={y - 22}
                      width="5"
                      height="14"
                      rx="1"
                      fill={state.runningLight}
                    />
                    <rect
                      x={s.x + nodeW / 2 + 3}
                      y={y - 22}
                      width="5"
                      height="14"
                      rx="1"
                      fill={state.runningLight}
                    />
                  </g>
                )}
              </g>
            );
          })}

          {/* Replay annotation: Hits Wait -> Resumed (checkpoint replay) */}
          {phase >= 4 && (
            <g style={{ transition: "opacity 0.5s ease" }}>
              <path
                d={`M ${steps[1].x + nodeW / 2} ${y - 8} C ${steps[1].x + nodeW / 2} ${y - 34}, ${steps[3].x + nodeW / 2} ${y - 34}, ${steps[3].x + nodeW / 2} ${y - 8}`}
                fill="none"
                stroke={state.success}
                strokeWidth="1.5"
                strokeDasharray="4 3"
              />
              <text
                x={(steps[1].x + steps[3].x) / 2 + nodeW / 2}
                y={y - 32}
                textAnchor="middle"
                fill={state.successLight}
                fontSize="9"
              >
                replay up to checkpoint, then continue
              </text>
            </g>
          )}

          {/* Connector: Evicted node <-> slot 1 (task leaves the worker) */}
          <path
            d={`M ${slot1X + slotW / 2} ${slotY - 4} C ${slot1X + slotW / 2} ${workerY - 20}, ${steps[2].x + nodeW / 2} ${y + nodeH + 30}, ${steps[2].x + nodeW / 2} ${y + nodeH + 4}`}
            fill="none"
            stroke={isEvicting ? brand.magenta : inactive.edge}
            strokeWidth="1.5"
            strokeDasharray="4 4"
            className={isEvicting ? "te-flow" : ""}
            style={{
              opacity: isEvicting ? 1 : 0.2,
              transition: "all 0.4s ease",
            }}
          />

          {/* Connector: Resumed node <-> slot 2 (task re-triggered on worker) */}
          <path
            d={`M ${steps[3].x + nodeW / 2} ${y + nodeH + 4} C ${steps[3].x + nodeW / 2} ${y + nodeH + 30}, ${slot2X + slotW / 2} ${workerY - 20}, ${slot2X + slotW / 2} ${slotY - 4}`}
            fill="none"
            stroke={isResuming ? state.success : inactive.edge}
            strokeWidth="1.5"
            strokeDasharray="4 4"
            className={isResuming ? "te-flow" : ""}
            style={{
              opacity: isResuming ? 1 : 0.2,
              transition: "all 0.4s ease",
            }}
          />

          {/* Worker box */}
          <rect
            x={workerX}
            y={workerY}
            width={workerW}
            height={workerH}
            rx={10}
            fill="rgba(10, 16, 41, 0.2)"
            stroke={inactive.stroke}
            strokeWidth="1.5"
          />
          <text
            x={workerX + 14}
            y={workerY + 22}
            fill={brand.cyanDark}
            fontSize="11"
            fontWeight="500"
          >
            Worker
          </text>
          <text
            x={workerX + workerW - 14}
            y={workerY + 22}
            textAnchor="end"
            fill={inactive.text}
            fontSize="9"
          >
            2 slots
          </text>

          {/* Slot 1: durable task -> freed -> other task */}
          <rect
            x={slot1X}
            y={slotY}
            width={slotW}
            height={slotH}
            rx={6}
            fill={
              phase <= 0
                ? fill.activeNode
                : phase === 1
                  ? fill.running
                  : phase === 2
                    ? "rgba(10, 16, 41, 0.1)"
                    : fill.activeNode
            }
            stroke={
              phase <= 0
                ? brand.blue
                : phase === 1
                  ? state.running
                  : phase === 2
                    ? inactive.stroke
                    : brand.blue
            }
            strokeWidth="1.5"
            strokeDasharray={phase === 1 || phase === 2 ? "4 3" : "none"}
            style={{ transition: "all 0.4s ease" }}
          />
          {phase === 2 ? (
            <text
              x={slot1X + slotW / 2}
              y={slotY + slotH / 2 + 4}
              textAnchor="middle"
              fill={brand.magentaLight}
              fontSize="10"
              className="te-pulse"
            >
              slot freed
            </text>
          ) : (
            <>
              <text
                x={slot1X + slotW / 2}
                y={slotY + slotH / 2 - 2}
                textAnchor="middle"
                fill={phase === 1 ? state.runningLight : brand.cyan}
                fontSize="10"
                fontWeight="500"
                style={{ transition: "fill 0.4s ease" }}
              >
                {phase <= 1 ? "durable task" : "other task"}
              </text>
              <text
                x={slot1X + slotW / 2}
                y={slotY + slotH / 2 + 12}
                textAnchor="middle"
                fill={phase === 1 ? state.runningDark : brand.blue}
                fontSize="8"
                style={{ transition: "fill 0.4s ease" }}
              >
                {phase <= 0 ? "running" : phase === 1 ? "waiting" : "running"}
              </text>
            </>
          )}

          {/* Slot 2: empty -> durable task resumed */}
          <rect
            x={slot2X}
            y={slotY}
            width={slotW}
            height={slotH}
            rx={6}
            fill={phase >= 4 ? fill.success : "rgba(10, 16, 41, 0.1)"}
            stroke={phase >= 4 ? state.success : inactive.stroke}
            strokeWidth="1.5"
            strokeDasharray={phase >= 4 ? "none" : "4 3"}
            style={{ transition: "all 0.4s ease" }}
          />
          {phase >= 4 ? (
            <>
              <text
                x={slot2X + slotW / 2}
                y={slotY + slotH / 2 - 2}
                textAnchor="middle"
                fill={state.successLighter}
                fontSize="10"
                fontWeight="500"
              >
                durable task
              </text>
              <text
                x={slot2X + slotW / 2}
                y={slotY + slotH / 2 + 12}
                textAnchor="middle"
                fill={state.successLight}
                fontSize="8"
              >
                {phase === 4 ? "replaying" : "complete"}
              </text>
            </>
          ) : (
            <text
              x={slot2X + slotW / 2}
              y={slotY + slotH / 2 + 4}
              textAnchor="middle"
              fill={inactive.text}
              fontSize="10"
            >
              empty slot
            </text>
          )}
        </svg>
      </div>
    </div>
  );
};

export default TaskEvictionDiagram;
