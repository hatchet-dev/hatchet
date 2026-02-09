import { SectionHeader } from './section-header';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { ChevronDownIcon } from '@radix-ui/react-icons';

export function CreateApiTokenSection({
  tokenName,
  onTokenNameChange,
  expiresIn,
  expiresInOptions,
  onExpiresInChange,
  onExpiresInSelected,
  fieldErrors,
  isGenerating,
  onGenerateToken,
}: {
  tokenName: string;
  onTokenNameChange: (value: string) => void;
  expiresIn: string;
  expiresInOptions: Record<string, string>;
  onExpiresInChange: (value: string) => void;
  onExpiresInSelected?: (label: string, value: string) => void;
  fieldErrors: Record<string, string>;
  isGenerating: boolean;
  onGenerateToken: () => void;
}) {
  const selectedLabel =
    Object.entries(expiresInOptions).find(
      ([, value]) => value === expiresIn,
    )?.[0] || 'Select an option';

  return (
    <div>
      <SectionHeader title="Create API token" showOnboardingBadge />
      <div className="grid gap-4 items-end grid-flow-row lg:[grid-template-columns:1fr_1fr_auto_1fr] lg:grid-flow-col">
        <div className="grid gap-2">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            type="text"
            required={true}
            autoCapitalize="none"
            autoCorrect="off"
            placeholder="My Token"
            value={tokenName}
            onChange={(e) => onTokenNameChange(e.target.value)}
            disabled={isGenerating}
          />
          {fieldErrors.name && (
            <div className="text-sm text-red-500">{fieldErrors.name}</div>
          )}
        </div>

        <div className="grid gap-2">
          <Label htmlFor="expiresIn">Expires In</Label>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                size="default"
                className="flex justify-between data-[state=open]:bg-muted"
                disabled={isGenerating}
              >
                {selectedLabel}
                <ChevronDownIcon className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-[160px]">
              {Object.entries(expiresInOptions).map(([label, value]) => (
                <DropdownMenuItem
                  key={value}
                  onClick={() => {
                    onExpiresInChange(value);
                    onExpiresInSelected?.(label, value);
                  }}
                >
                  {label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="grid gap-2 justify-self-start">
          <Button
            variant="default"
            size="default"
            onClick={onGenerateToken}
            disabled={isGenerating}
          >
            {isGenerating && <Spinner />}
            Generate Token
          </Button>
        </div>
      </div>
    </div>
  );
}
