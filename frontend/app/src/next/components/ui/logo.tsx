import { Avatar, AvatarFallback, AvatarImage } from '@/next/components/ui/avatar';
import logo from '@/next/assets/logo.svg';
import hatchet from '@/next/assets/hatchet.svg';
import { cn } from '@/next/lib/utils';
import { useTheme } from '@/next/components/theme-provider';

const variants = {
  icon: 'h-8 w-8 rounded-lg',
  md: 'h-8 w-8 rounded-lg',
  large: 'h-12 w-12 rounded-lg',
};

type LogoProps = {
  variant?: keyof typeof variants;
};

export function Logo({ variant = 'md' }: LogoProps) {
  const { theme } = useTheme();
  return (
    <>
      <Avatar className={variants[variant]}>
        <AvatarImage src={logo} />
        <AvatarFallback className="rounded-lg">🪓</AvatarFallback>
      </Avatar>
      {variant !== 'icon' && (
        <div className="grid flex-1 text-left text-sm leading-tight">
          <span className="truncate font-semibold">
            <img
              src={hatchet}
              alt="Hatchet"
              className={cn(
                'transition-all h-4 ml-1',
                theme === 'dark' ? 'invert' : '',
              )}
            />
          </span>
        </div>
      )}
    </>
  );
}
