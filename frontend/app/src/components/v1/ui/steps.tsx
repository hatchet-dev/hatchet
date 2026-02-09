import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from './accordion';
import { cn } from '@/lib/utils';
import { cva, type VariantProps } from 'class-variance-authority';
import * as React from 'react';

const stepsVariants = cva(
  'ml-4 mb-12 border-l border-border pl-6 dark:border-border [counter-reset:step] flex flex-col gap-4',
);

interface StepProps {
  title?: string;
  children: React.ReactNode;
  stepNumber?: number; // Add the stepNumber prop
  open?: boolean;
  setOpen?: (collapsed: boolean) => void;
  disabled?: boolean;
}

const Step = ({
  title,
  children,
  stepNumber,
  open = true,
  setOpen = () => {},
  disabled = false,
}: StepProps) => {
  return (
    <Accordion
      type="single"
      collapsible={true}
      className="relative w-full"
      value={open ? 'open' : 'closed'}
      disabled={disabled}
      onValueChange={(value) => setOpen(value === 'open')}
    >
      <AccordionItem value="open" className="border-none">
        <AccordionTrigger
          hideChevron={disabled}
          className={disabled ? 'cursor-default hover:no-underline' : ''}
        >
          <div className="absolute ml-[-41px] mt-[3px] h-[33px] w-[33px] rounded-full border-4 border-muted bg-muted text-center text-base font-normal text-foreground">
            {stepNumber}
          </div>
          {title && (
            <h3 className="text-lg font-semibold hover:no-underline">
              {title}
            </h3>
          )}
        </AccordionTrigger>
        <AccordionContent>{children}</AccordionContent>
      </AccordionItem>
    </Accordion>
  );
};

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
