import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import logo from '@/assets/logo.svg';

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
      <AvatarImage src={logo} />
      <AvatarFallback className="rounded-lg">🪓</AvatarFallback>
    </Avatar>
  );
}
