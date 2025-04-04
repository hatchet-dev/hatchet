import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const stepsVariants = cva(
  'ml-4 mb-12 border-l border-border pl-6 dark:border-border [counter-reset:step] flex flex-col gap-12',
);

interface StepProps {
  title?: string;
  children: React.ReactNode;
  stepNumber?: number; // Add the stepNumber prop
}

const Step = ({ title, children, stepNumber }: StepProps) => (
  <div className="relative">
    <div className="absolute w-[33px] h-[33px] border-4 border-muted bg-muted rounded-full text-foreground text-base font-normal text-center mt-[3px] ml-[-41px]">
      {stepNumber}
    </div>
    <div className="pl-12">
      {title && <h3 className="text-lg font-semibold mb-2">{title}</h3>}
      {children}
    </div>
  </div>
);

const Steps = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof stepsVariants>
>(({ className, children, ...props }, ref) => (
  <div ref={ref} className={cn(stepsVariants(), className)} {...props}>
    {React.Children.map(children, (child, index) =>
      React.isValidElement(child)
        ? React.cloneElement(child as React.ReactElement<any>, {
            stepNumber: index + 1,
          })
        : child,
    )}
  </div>
));

Steps.displayName = 'Steps';

export { Steps, Step };
