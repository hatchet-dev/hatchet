import { Avatar, AvatarFallback } from '@/components/ui/avatar';

const variants = {
  md: 'h-8 w-8 rounded-lg',
  large: 'h-12 w-12 rounded-lg',
};

type LogoProps = {
  variant?: keyof typeof variants;
};

export function Logo({ variant = 'md' }: LogoProps) {
  return (
    <Avatar className={variants[variant]}>
      <AvatarFallback className="rounded-lg">🪓</AvatarFallback>
    </Avatar>
  );
}
