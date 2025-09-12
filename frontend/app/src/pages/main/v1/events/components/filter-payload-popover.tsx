import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { DotsVerticalIcon } from '@radix-ui/react-icons';
import { EyeIcon } from 'lucide-react';

export const FilterPayloadPopover = ({
  isOpen,
  setIsOpen,
  content,
}: {
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
  content: string;
}) => {
  return (
    <Popover modal={true} open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          className="h-8 w-8 p-0 opacity-0 pointer-events-none absolute"
        >
          <DotsVerticalIcon className="h-4 w-4 text-muted-foreground cursor-pointer" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className="md:w-[500px] lg:w-[700px] max-w-[90vw] p-0 my-4 shadow-xl border-2 bg-background/95 backdrop-blur-sm rounded-lg"
        align="center"
        side="left"
      >
        <div className="bg-muted/50 px-4 py-3 border-b border-border/50 flex-shrink-0">
          <div className="flex items-center gap-2">
            <EyeIcon className="h-4 w-4 text-muted-foreground" />
            <span className="font-semibold text-sm text-foreground">
              Filter Payload
            </span>
          </div>
        </div>
        <div className="p-4">
          <div className="max-h-[60vh] overflow-auto rounded-lg border border-border/50 bg-muted/10">
            <div className="p-4">
              <CodeHighlighter
                language="json"
                className="whitespace-pre-wrap break-words text-sm leading-relaxed"
                code={content}
              />
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};
