import React from "react";
import { parseDocComments } from "./codeParser";
import { Src } from "./codeData";
import CodeStyleRender from "./CodeStyleRender";

interface CodeRendererProps {
  source: Src;
  target: string;
}

export const CodeBlock = ({ source, target }: CodeRendererProps) => {
    const [collapsed, setCollapsed] = React.useState(true);
    const [plainText, setPlainText] = React.useState(false);


    const parsed = parseDocComments(source.raw, target, collapsed);

    return <>
        <div className="flex flex-row gap-4 justify-between items-center pl-2">
            <a href={source.githubUrl} target="_blank" rel="noopener noreferrer" className="text-xs text-gray-500 font-mono hover:underline">{source.props.path}</a>
            <div className="flex gap-2 justify-end">
                <button
                    onClick={() => setCollapsed(!collapsed)}
                    className="flex items-center gap-1 px-3 py-1 text-sm rounded border hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
                >
                    {collapsed ? (
                        <>
                            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
                            </svg>
                            <span>Expand</span>
                        </>
                    ) : (
                        <>
                            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M4 8a.5.5 0 0 1 .5-.5h7a.5.5 0 0 1 0 1h-7A.5.5 0 0 1 4 8Z"/>
                            </svg>
                            <span>Collapse</span>
                        </>
                    )}
                </button>
                <button
                    onClick={() => setPlainText(!plainText)}
                    className="flex items-center gap-1 px-3 py-1 text-sm rounded border hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
                >
                    {plainText ? (
                        <>
                            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M2.5 3.5v9h11v-9h-11zm0-1h11a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1v-9a1 1 0 0 1 1-1z"/>
                                <path d="M3.5 5.5h9v1h-9v-1zm0 2h9v1h-9v-1zm0 2h5v1h-5v-1z"/>
                            </svg>
                            <span>Show HTML</span>
                        </>
                    ) : (
                        <>
                            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M5.854 4.854a.5.5 0 1 0-.708-.708l-3.5 3.5a.5.5 0 0 0 0 .708l3.5 3.5a.5.5 0 0 0 .708-.708L2.707 8l3.147-3.146zm4.292 0a.5.5 0 0 1 .708-.708l3.5 3.5a.5.5 0 0 1 0 .708l-3.5 3.5a.5.5 0 0 1-.708-.708L13.293 8l-3.147-3.146z"/>
                            </svg>
                            <span>Show Plain Text</span>
                        </>
                    )}
                </button>
            </div>
        </div>
       {!plainText && <CodeStyleRender parsed={parsed} language={source.language} />}
        {/* plain text for SEO */}
        <pre style={{ display: plainText ? 'block' : 'none' }} aria-hidden="true">{parsed}</pre>

        <div className="flex gap-4 mb-4 p-2 justify-end">
            <a
                href={source.rawUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="no-underline text-gray-500 flex items-center gap-2"
            >
                View Raw
            </a>
            <a
                href={source.githubUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="no-underline text-[#0969da] flex items-center gap-2"
            >
                <svg height="16" width="16" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
                </svg>
                View Full Code on GitHub &rarr;
            </a>
        </div>
    </>
}
