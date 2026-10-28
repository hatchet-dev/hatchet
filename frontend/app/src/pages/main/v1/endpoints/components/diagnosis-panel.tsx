import { Diagnosis, DiagnosisSeverity } from '../lib/diagnose';
import { Copy, CopySentences, listOf, renderInline, t } from './copy';
import { cn } from '@/lib/utils';
import {
  AlertTriangle,
  CheckCircle2,
  Clock,
  Info,
  XCircle,
} from 'lucide-react';

const STYLE: Record<
  DiagnosisSeverity,
  { icon: typeof Info; className: string; iconClassName: string }
> = {
  healthy: {
    icon: CheckCircle2,
    className:
      'border-green-300 bg-green-50/50 dark:border-green-900 dark:bg-green-950/30',
    iconClassName: 'text-green-600 dark:text-green-400',
  },
  info: {
    icon: Info,
    className:
      'border-blue-200 bg-blue-50/50 dark:border-blue-900 dark:bg-blue-950/30',
    iconClassName: 'text-blue-600 dark:text-blue-400',
  },
  pending: {
    icon: Clock,
    className: 'border-border bg-muted/30',
    iconClassName: 'text-muted-foreground',
  },
  warning: {
    icon: AlertTriangle,
    className:
      'border-yellow-400 bg-yellow-50/50 dark:border-yellow-800 dark:bg-yellow-950/20',
    iconClassName: 'text-yellow-600 dark:text-yellow-400',
  },
  error: {
    icon: XCircle,
    className:
      'border-red-300 bg-red-50/50 dark:border-red-800 dark:bg-red-950/40',
    iconClassName: 'text-red-600 dark:text-red-400',
  },
};

export const DiagnosisPanel = ({
  diagnosis,
  rawError,
}: {
  diagnosis: Diagnosis;
  // the error text the diagnosis was derived from, shown verbatim
  rawError?: string;
}) => {
  const style = STYLE[diagnosis.severity];
  const Icon = style.icon;
  const steps = diagnosis.steps.flatMap(listOf);
  // a single step reads better as part of the description than as a list of one
  const inlineStep = steps.length === 1 ? steps[0] : undefined;

  return (
    <div className={cn('rounded-lg border p-4', style.className)}>
      <div className="flex flex-row gap-3">
        <Icon className={cn('mt-0.5 size-5 shrink-0', style.iconClassName)} />
        <div className="min-w-0 flex-1 space-y-2">
          <h3 className="font-semibold">
            <Copy copyRef={diagnosis.title} />
          </h3>
          {diagnosis.cause && (
            <p className="text-sm font-medium">
              <Copy copyRef={diagnosis.cause} />
            </p>
          )}
          {(diagnosis.detail.length > 0 || inlineStep) && (
            <p className="text-sm">
              <CopySentences refs={diagnosis.detail} />
              {inlineStep && (
                <>
                  {diagnosis.detail.length > 0 && ' '}
                  {renderInline(inlineStep)}
                </>
              )}
            </p>
          )}
          {rawError && diagnosis.severity !== 'healthy' && (
            <pre className="whitespace-pre-wrap break-all rounded bg-background/60 px-2 py-1 font-mono text-xs text-muted-foreground">
              {rawError}
            </pre>
          )}
          {steps.length > 1 && (
            <div>
              <div className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {t('detail.whatToCheck')}
              </div>
              <ol className="list-decimal space-y-1 pl-5 text-sm">
                {steps.map((step) => (
                  <li key={step}>{renderInline(step)}</li>
                ))}
              </ol>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
