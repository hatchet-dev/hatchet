export type SpanCardConnectorType =
  | 'horizontal'
  | 'vertical'
  | 't-right'
  | 'corner-top-right'
  | 'empty';

interface SpanCardConnectorProps {
  type: SpanCardConnectorType;
}

export const SpanCardConnector = ({ type }: SpanCardConnectorProps) => {
  if (type === 'empty') {
    return <div className="w-5 shrink-0 grow" />;
  }

  return (
    <div className="relative w-5 shrink-0 grow">
      {(type === 'vertical' || type === 't-right') && (
        <div className="absolute inset-y-0 left-1/2 z-10 w-px -translate-x-1/2 bg-border" />
      )}

      {type === 't-right' && (
        <div className="absolute left-1/2 top-2.5 h-px w-2.5 translate-y-[-3px] bg-border" />
      )}

      {type === 'corner-top-right' && (
        <>
          <div className="absolute left-1/2 top-0 h-[9px] w-px -translate-x-px bg-border" />

          <div className="absolute left-1/2 top-2.5 h-px w-2.5 translate-y-[-3px] bg-border" />
        </>
      )}
    </div>
  );
};
