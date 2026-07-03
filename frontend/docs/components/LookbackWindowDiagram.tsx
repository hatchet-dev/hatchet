import React, { useState, useEffect } from "react";
import { brand, state, fill, inactive, gradient } from "./diagram-colors";

const LookbackWindowDiagram: React.FC = () => {
  const [phase, setPhase] = useState(0); // 0=event pushed, 1=wait registers, 2=without lookback misses, 3=lookback catches

  useEffect(() => {
    const durations = [1600, 1600, 2200, 2800];
    const timer = setTimeout(() => {
      setPhase((prev) => (prev + 1) % 4);
    }, durations[phase]);
    return () => clearTimeout(timer);
  }, [phase]);

  const waitRegistered = phase >= 1;
  const withoutResolved = phase >= 2;
  const lookbackResolved = phase >= 3;

  // Shared timeline geometry
  const lineStart = 170;
  const lineEnd = 715;
  const eventX = 320;
  const waitX = 520;
  const windowStart = 300; // lookback window reaches back past the event
  const laneOneY = 78;
  const laneTwoY = 196;
  const regionH = 40;

  const statusColor = lookbackResolved
    ? state.successLight
    : withoutResolved
      ? state.failedLight
      : waitRegistered
        ? brand.cyan
        : brand.blueLight;

  const statusText = lookbackResolved
    ? "lookback window catches the earlier event"
    : withoutResolved
      ? "without lookback: the earlier event is missed"
      : waitRegistered
        ? "durable wait registers (with the same scope)"
        : "event pushed with scope user_id:1234";

  const renderTimeline = (y: number) => (
    <g>
      <line
        x1={lineStart}
        y1={y}
        x2={lineEnd}
        y2={y}
        stroke={inactive.line}
        strokeWidth="1.5"
      />
      <path
        d={`M ${lineEnd} ${y} l -8 -4 v 8 z`}
        fill={inactive.line}
        stroke="none"
      />
    </g>
  );

  const renderEventDot = (y: number, matched: boolean, missed: boolean) => (
    <g>
      <circle
        cx={eventX}
        cy={y}
        r="6"
        fill={
          matched
            ? fill.success
            : missed
              ? fill.failed
              : "rgba(51, 146, 255, 0.25)"
        }
        stroke={
          matched ? state.successLight : missed ? state.failed : brand.blue
        }
        strokeWidth="1.5"
        className={phase === 0 ? "lb-pulse" : ""}
        style={{ transition: "all 0.4s ease" }}
      />
      {matched && (
        <path
          d={`M ${eventX - 3} ${y} l 2.5 2.5 l 4 -5`}
          fill="none"
          stroke={state.successLighter}
          strokeWidth="1.5"
          strokeLinecap="round"
        />
      )}
      {missed && (
        <g stroke={state.failedLight} strokeWidth="1.5" strokeLinecap="round">
          <line x1={eventX - 2.5} y1={y - 2.5} x2={eventX + 2.5} y2={y + 2.5} />
          <line x1={eventX + 2.5} y1={y - 2.5} x2={eventX - 2.5} y2={y + 2.5} />
        </g>
      )}
      <text
        x={eventX}
        y={y - 16}
        textAnchor="middle"
        fill={
          matched ? state.successLight : missed ? state.failedLight : brand.cyan
        }
        fontSize="10"
        fontWeight="500"
        style={{ transition: "fill 0.4s ease" }}
      >
        user:create
      </text>
      <text
        x={eventX}
        y={y + 24}
        textAnchor="middle"
        fill={inactive.text}
        fontSize="8"
      >
        scope user_id:1234
      </text>
    </g>
  );

  const renderWaitMarker = (y: number) => (
    <g
      style={{
        opacity: waitRegistered ? 1 : 0,
        transition: "opacity 0.4s ease",
      }}
    >
      <line
        x1={waitX}
        y1={y - 24}
        x2={waitX}
        y2={y + 14}
        stroke={brand.cyanDark}
        strokeWidth="1.5"
      />
      <text
        x={waitX}
        y={y - 32}
        textAnchor="middle"
        fill={brand.cyan}
        fontSize="10"
        fontWeight="500"
      >
        wait registers
      </text>
      <text
        x={waitX}
        y={y + 26}
        textAnchor="middle"
        fill={inactive.text}
        fontSize="8"
      >
        scope user_id:1234
      </text>
    </g>
  );

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      {/* Status */}
      <div className="flex items-center gap-2 rounded-lg border border-neutral-700/40 bg-neutral-900/50 px-4 py-2">
        <span
          className="text-xs transition-colors duration-300"
          style={{ color: statusColor }}
        >
          {statusText}
        </span>
      </div>

      {/* Diagram */}
      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <svg
          viewBox="0 0 760 260"
          className="w-full h-auto"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient id="lb-yellow" x1="0%" y1="0%" x2="100%" y2="100%">
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
          </defs>

          <style>{`
            @keyframes lb-pulse {
              0%, 100% { opacity: 0.4; }
              50% { opacity: 1; }
            }
            .lb-pulse {
              animation: lb-pulse 1.2s ease-in-out infinite;
            }
          `}</style>

          {/* ── Lane 1: without lookback ─────────────────────────── */}
          <text
            x={20}
            y={laneOneY - 4}
            fill={withoutResolved ? "#94A3B8" : inactive.textLight}
            fontSize="11"
            fontWeight="500"
            style={{ transition: "fill 0.4s ease" }}
          >
            Without
          </text>
          <text
            x={20}
            y={laneOneY + 10}
            fill={withoutResolved ? "#94A3B8" : inactive.textLight}
            fontSize="11"
            fontWeight="500"
            style={{ transition: "fill 0.4s ease" }}
          >
            lookback
          </text>

          {renderTimeline(laneOneY)}

          {/* Visibility region: only events after registration */}
          <g
            style={{
              opacity: withoutResolved ? 1 : 0,
              transition: "opacity 0.4s ease",
            }}
          >
            <rect
              x={waitX}
              y={laneOneY - regionH / 2}
              width={lineEnd - 15 - waitX}
              height={regionH}
              rx="6"
              fill={fill.blue}
              stroke={brand.blue}
              strokeWidth="1"
              strokeDasharray="4 4"
              opacity="0.8"
            />
            <text
              x={waitX + (lineEnd - 15 - waitX) / 2}
              y={laneOneY + 3}
              textAnchor="middle"
              fill={brand.blueLight}
              fontSize="9"
            >
              only sees new events
            </text>
          </g>

          {renderEventDot(laneOneY, false, withoutResolved)}
          {renderWaitMarker(laneOneY)}

          <text
            x={lineEnd}
            y={laneOneY - 32}
            textAnchor="end"
            fill={state.failedLight}
            fontSize="10"
            fontWeight="500"
            style={{
              opacity: withoutResolved ? 1 : 0,
              transition: "opacity 0.4s ease",
            }}
          >
            event missed — wait keeps blocking
          </text>

          {/* ── Lane 2: with lookback ────────────────────────────── */}
          <text
            x={20}
            y={laneTwoY - 4}
            fill={lookbackResolved ? state.successLight : inactive.textLight}
            fontSize="11"
            fontWeight="500"
            style={{ transition: "fill 0.4s ease" }}
          >
            With 1m
          </text>
          <text
            x={20}
            y={laneTwoY + 10}
            fill={lookbackResolved ? state.successLight : inactive.textLight}
            fontSize="11"
            fontWeight="500"
            style={{ transition: "fill 0.4s ease" }}
          >
            lookback
          </text>

          {renderTimeline(laneTwoY)}

          {/* Forward visibility region (same as lane 1) */}
          <g
            style={{
              opacity: withoutResolved ? 1 : 0,
              transition: "opacity 0.4s ease",
            }}
          >
            <rect
              x={waitX}
              y={laneTwoY - regionH / 2}
              width={lineEnd - 15 - waitX}
              height={regionH}
              rx="6"
              fill={fill.blue}
              stroke={brand.blue}
              strokeWidth="1"
              strokeDasharray="4 4"
              opacity="0.8"
            />
            <text
              x={waitX + (lineEnd - 15 - waitX) / 2}
              y={laneTwoY + 3}
              textAnchor="middle"
              fill={brand.blueLight}
              fontSize="9"
            >
              new events
            </text>
          </g>

          {/* Lookback window reaching back in time */}
          <g
            style={{
              opacity: lookbackResolved ? 1 : 0,
              transition: "opacity 0.4s ease",
            }}
          >
            <rect
              x={windowStart}
              y={laneTwoY - regionH / 2}
              width={waitX - windowStart}
              height={regionH}
              rx="6"
              fill={fill.runningLight}
              stroke="url(#lb-yellow)"
              strokeWidth="1.5"
              strokeDasharray="6 4"
            />
            <text
              x={windowStart + (waitX - windowStart) / 2 + 24}
              y={laneTwoY + 3}
              textAnchor="middle"
              fill={state.runningLight}
              fontSize="9"
            >
              1m lookback
            </text>
            {/* Arrow pointing back in time from registration */}
            <line
              x1={waitX - 8}
              y1={laneTwoY - regionH / 2 - 8}
              x2={windowStart + 8}
              y2={laneTwoY - regionH / 2 - 8}
              stroke={state.running}
              strokeWidth="1.5"
            />
            <path
              d={`M ${windowStart + 8} ${laneTwoY - regionH / 2 - 8} l 8 -4 v 8 z`}
              fill={state.running}
              stroke="none"
            />
          </g>

          {renderEventDot(laneTwoY, lookbackResolved, false)}
          {renderWaitMarker(laneTwoY)}

          <text
            x={lineEnd}
            y={laneTwoY - 32}
            textAnchor="end"
            fill={state.successLight}
            fontSize="10"
            fontWeight="500"
            style={{
              opacity: lookbackResolved ? 1 : 0,
              transition: "opacity 0.4s ease",
            }}
          >
            event matched — task resumes
          </text>

          {/* Time axis label */}
          <text
            x={lineEnd}
            y={laneTwoY + 44}
            textAnchor="end"
            fill={inactive.text}
            fontSize="9"
          >
            time →
          </text>
        </svg>
      </div>
    </div>
  );
};

export default LookbackWindowDiagram;
