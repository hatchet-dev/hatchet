import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from './accordion';

const stepsVariants = cva(
  'ml-4 mb-12 border-l border-border pl-6 dark:border-border [counter-reset:step] flex flex-col gap-12',
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
      className="w-full relative"
      value={open ? 'open' : 'closed'}
      disabled={disabled}
      onValueChange={(value) => setOpen(value === 'open')}
    >
      <AccordionItem value="open" className="border-none">
        <AccordionTrigger
          hideChevron={disabled}
          className={disabled ? 'hover:no-underline cursor-default' : ''}
        >
          <div className="absolute w-[33px] h-[33px] border-4 border-muted bg-muted rounded-full text-foreground text-base font-normal text-center mt-[3px] ml-[-41px]">
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
