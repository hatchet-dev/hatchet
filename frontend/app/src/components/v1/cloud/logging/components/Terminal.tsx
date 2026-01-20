import { useTerminal, TerminalScrollCallbacks } from '../hooks/useTerminal';
import { useRef } from 'react';

interface TerminalProps {
  logs: string;
  autoScroll?: boolean;
  callbacks?: TerminalScrollCallbacks;
}

function Terminal({ logs, autoScroll, callbacks }: TerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  useTerminal(containerRef, logs, { autoScroll, callbacks });

  return (
    <div
      className="h-[500px] md:h-[600px] pl-6 pt-6 pb-6 bg-[#1e293b] rounded-md relative overflow-hidden font-mono text-xs [&_canvas]:block [&_.xterm-cursor]:!hidden [&_textarea]:!fixed [&_textarea]:!left-[-9999px] [&_textarea]:!top-[-9999px]"
      ref={containerRef}
      onFocus={(e) => e.currentTarget.blur()}
      tabIndex={-1}
    ></div>
  );
}

export default Terminal;
