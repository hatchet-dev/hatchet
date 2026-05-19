import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Separator } from '@/components/v1/ui/separator';
import { ArrowRightIcon } from '@radix-ui/react-icons';
import { BiBook, BiMessageSquareDetail } from 'react-icons/bi';
import { RiDiscordFill, RiGithubFill, RiSlackFill } from 'react-icons/ri';

export function SupportSection() {
  return (
    <div className="pb-6">
      <h2 className="text-md">Support</h2>
      <Separator className="my-4 bg-border/50" flush />
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <Card
          variant="light"
          className="bg-transparent ring-1 ring-border/50 border-none"
        >
          <CardHeader className="p-4 border-b border-border/50 ">
            <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground whitespace-nowrap">
              Community
            </CardTitle>
          </CardHeader>
          <CardContent className="p-4">
            <ul className="space-y-2">
              <li>
                <a
                  href="https://hatchet.run/discord"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                >
                  <RiDiscordFill className="mr-2" /> Join Discord
                </a>
              </li>
              <li>
                <a
                  href="https://github.com/hatchet-dev/hatchet/issues"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                >
                  <RiGithubFill className="mr-2" /> Github Issues
                </a>
              </li>
            </ul>
          </CardContent>
        </Card>

        <Card
          variant="light"
          className="bg-transparent ring-1 ring-border/50 border-none"
        >
          <CardHeader className="p-4 border-b border-border/50 ">
            <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground whitespace-nowrap">
              Resources
            </CardTitle>
          </CardHeader>
          <CardContent className="p-4">
            <ul className="space-y-2">
              <li>
                <a
                  href="https://hatchet.run/discord"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                >
                  <BiBook className="mr-2" /> Documentation
                </a>
              </li>
              <li>
                <a
                  href="mailto:support@hatchet.run?subject=Slack%20Channel%20Request"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                >
                  <RiSlackFill className="mr-2" />
                  Request Slack Channel
                </a>
              </li>
              <li>
                <a
                  href="mailto:support@hatchet.run"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                >
                  <BiMessageSquareDetail className="mr-2" />
                  Email Support
                </a>
              </li>
            </ul>
          </CardContent>
        </Card>

        <Card
          variant="light"
          className="bg-transparent ring-1 ring-border/50 border-none"
        >
          <CardHeader className="p-4 border-b border-border/50 ">
            <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground whitespace-nowrap">
              <a
                href="https://hatchet.run/office-hours"
                target="_blank"
                rel="noreferrer"
                className="group inline-flex items-center gap-1 hover:text-primary"
              >
                Book Office Hours
                <ArrowRightIcon className="size-3 transition-transform group-hover:translate-x-0.5" />
              </a>
            </CardTitle>
          </CardHeader>
          <CardContent className="p-4 space-y-2">
            <span className="text-sm text-muted-foreground whitespace-nowrap">
              GMT-5 Eastern Standard Time
            </span>
            <p className="text-sm text-muted-foreground whitespace-nowrap">
              Weekdays <span className="text-primary">09:00 - 17:00</span>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
