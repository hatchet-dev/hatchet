import DOMPurify from 'dompurify';
import AnsiToHtml from 'ansi-to-html';

const convert = new AnsiToHtml({
  newline: true,
  bg: 'transparent',
});

export default function StepRunError({ text }: { text: string }) {
  const sanitizedHtml = DOMPurify.sanitize(convert.toHtml(text || ''), {
    USE_PROFILES: { html: true },
  });

  return (
    <div
      className="rounded-md h-[400px] overflow-y-auto bg-muted p-4"
      style={{
        width: '100%',
        maxWidth: '100%',
        minWidth: 0,
        wordWrap: 'break-word',
        overflowWrap: 'break-word',
      }}
    >
      <div
        className="text-indigo-300 font-mono text-xs"
        style={{
          width: '100%',
          maxWidth: '100%',
          minWidth: 0,
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          overflowWrap: 'break-word',
        }}
      >
        <span
          dangerouslySetInnerHTML={{
            __html: sanitizedHtml,
          }}
        />
      </div>
    </div>
  );
}
