import { Button } from '@/components/ui/button';
import { UserCircleIcon } from '@heroicons/react/24/outline';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

import { useNavigate } from 'react-router-dom';
import api, { User } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import hatchet from '@/assets/hatchet_logo.png';
import { useSidebar } from '@/components/sidebar-provider';

interface MainNavProps {
  user: User;
}

export default function MainNav({ user }: MainNavProps) {
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});
  const { toggleSidebarOpen } = useSidebar();

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate('/auth/login');
    },
    onError: handleApiError,
  });

  return (
    <div className="fixed top-0 w-screen h-16 border-b">
      <div className="flex h-16 items-center pr-4 pl-4">
        <button
          onClick={() => toggleSidebarOpen()}
          className="flex flex-row gap-4 items-center"
        >
          <img src={hatchet} alt="Hatchet" className="h-9 rounded" />
        </button>
        <div className="ml-auto flex items-center space-x-4">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                className="relative h-10 w-10 rounded-full p-1"
              >
                <UserCircleIcon className="h-6 w-6 text-foreground cursor-pointer" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-56" align="end" forceMount>
              <DropdownMenuLabel className="font-normal">
                <div className="flex flex-col space-y-1">
                  <p className="text-sm font-medium leading-none">
                    {user.name || user.email}
                  </p>
                  <p className="text-xs leading-none text-muted-foreground">
                    {user.email}
                  </p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => logoutMutation.mutate()}>
                Log out
                <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  );
}
