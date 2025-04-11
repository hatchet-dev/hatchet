import {
  AlertTriangle,
  Bug,
  Calendar,
  Clock,
  Code,
  Cpu,
  Database,
  Github,
  Key,
  Play,
  Scale,
  Settings,
  Users,
  UsersIcon,
} from 'lucide-react';
import { FaDiscord, FaPlay } from 'react-icons/fa';

export type NavItem = {
  title: string;
  url: string;
  icon: React.ElementType;
  isActive?: boolean;
  items?: NavItem[];
};

export type SupportItem = {
  title: string;
  url: string;
  icon: React.ElementType;
};

export type NavSection = {
  label: string;
  items: NavItem[];
};

export type NavStructure = {
  sections: {
    [key: string]: NavSection;
  };
  support: SupportItem[];
  navSecondary: NavItem[];
};

export const getMainNavLinks = (currentPath: string): NavStructure => {
  const isActive = (path: string) => currentPath.startsWith(path);

  return {
    sections: {
      activity: {
        label: 'Activity',
        items: [
          {
            title: 'Runs',
            url: '/runs',
            icon: Play,
            isActive: isActive('/runs'),
          },
        ],
      },
      triggers: {
        label: 'Triggers',
        items: [
          {
            title: 'Scheduled Runs',
            url: '/scheduled',
            icon: Calendar,
            isActive: isActive('/scheduled'),
          },
          {
            title: 'Cron Jobs',
            url: '/crons',
            icon: Clock,
            isActive: isActive('/crons'),
          },
        ],
      },
      resources: {
        label: 'Resources',
        items: [
          {
            title: 'Tasks & Workflows',
            url: '/tasks',
            icon: Code,
            isActive: isActive('/tasks'),
          },
          {
            title: 'Worker Services',
            url: '/services',
            icon: Cpu,
            isActive: isActive('/services'),
          },
          {
            title: 'Rate Limits',
            url: '/rate-limits',
            icon: Scale,
            isActive: isActive('/rate-limits'),
          },
        ],
      },
      settings: {
        label: 'Settings',
        items: [
          {
            title: 'API Tokens',
            url: '/settings/api-tokens',
            icon: Key,
            isActive: isActive('/settings/api-tokens'),
          },
          {
            title: 'Team',
            url: '/settings/team',
            icon: Users,
            isActive: isActive('/settings/team'),
          },
          {
            title: 'More Settings',
            url: '/settings',
            icon: Settings,
            isActive: isActive('/settings'),
            items: [
              {
                title: 'Tenant Settings',
                url: '/settings/overview',
                icon: UsersIcon,
                isActive: isActive('/settings/overview'),
              },
              {
                title: 'Github',
                url: '/settings/github',
                icon: Github,
                isActive: isActive('/settings/github'),
              },
              {
                title: 'Resource Limits',
                url: '/settings/resource-limits',
                icon: Scale,
                isActive: isActive('/settings/resource-limits'),
              },
              {
                title: 'Alerting',
                url: '/settings/alerting',
                icon: AlertTriangle,
                isActive: isActive('/settings/alerting'),
              },
              {
                title: 'Ingestors',
                url: '/settings/ingestors',
                icon: Database,
                isActive: isActive('/settings/ingestors'),
              },
            ],
          },
        ],
      },
    },
    support: [
      {
        title: 'Join Our Community',
        url: 'https://hatchet.run/discord',
        icon: FaDiscord,
      },
      {
        title: 'Restart Tutorial',
        url: '/tutorial',
        icon: FaPlay,
      },
    ],
    navSecondary: [
      {
        title: 'Feedback',
        url: 'https://github.com/hatchet-dev/hatchet/issues',
        icon: Bug,
      },
    ],
  };
};
