import { useState, useEffect } from 'react';
import { Time } from './time';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from './card';

export function TimeExamples() {
  // Create dates for examples
  const now = new Date();
  const oneMinuteAgo = new Date(now.getTime() - 60 * 1000);
  const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
  const oneDayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);
  const oneWeekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
  const oneMonthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);

  // State for current time to demonstrate live updates
  const [currentTime, setCurrentTime] = useState(new Date());

  // Update current time every second for demo purposes
  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Time Component Examples</CardTitle>
          <CardDescription>
            Live demonstration of all time variants. The timeSince variant uses
            timeago-react and auto-updates.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="grid grid-cols-4 gap-4">
              <div className="font-medium">Event</div>
              <div className="font-medium">Time Since (timeago-react)</div>
              <div className="font-medium">Timestamp</div>
              <div className="font-medium">Short</div>
            </div>

            <div className="grid grid-cols-4 gap-4 items-center">
              <div>Current time</div>
              <div>
                <Time date={currentTime} variant="timeSince" />
              </div>
              <div>
                <Time date={currentTime} variant="timestamp" />
              </div>
              <div>
                <Time date={currentTime} variant="short" />
              </div>
            </div>

            <div className="grid grid-cols-4 gap-4 items-center">
              <div>1 minute ago</div>
              <div>
                <Time date={oneMinuteAgo} variant="timeSince" />
              </div>
              <div>
                <Time date={oneMinuteAgo} variant="timestamp" />
              </div>
              <div>
                <Time date={oneMinuteAgo} variant="short" />
              </div>
            </div>

            <div className="grid grid-cols-4 gap-4 items-center">
              <div>1 hour ago</div>
              <div>
                <Time date={oneHourAgo} variant="timeSince" />
              </div>
              <div>
                <Time date={oneHourAgo} variant="timestamp" />
              </div>
              <div>
                <Time date={oneHourAgo} variant="short" />
              </div>
            </div>

            <div className="grid grid-cols-4 gap-4 items-center">
              <div>1 day ago</div>
              <div>
                <Time date={oneDayAgo} variant="timeSince" />
              </div>
              <div>
                <Time date={oneDayAgo} variant="timestamp" />
              </div>
              <div>
                <Time date={oneDayAgo} variant="short" />
              </div>
            </div>

            <div className="grid grid-cols-4 gap-4 items-center">
              <div>1 week ago</div>
              <div>
                <Time date={oneWeekAgo} variant="timeSince" />
              </div>
              <div>
                <Time date={oneWeekAgo} variant="timestamp" />
              </div>
              <div>
                <Time date={oneWeekAgo} variant="short" />
              </div>
            </div>

            <div className="grid grid-cols-4 gap-4 items-center">
              <div>1 month ago</div>
              <div>
                <Time date={oneMonthAgo} variant="timeSince" />
              </div>
              <div>
                <Time date={oneMonthAgo} variant="timestamp" />
              </div>
              <div>
                <Time date={oneMonthAgo} variant="short" />
              </div>
            </div>

            <div className="grid grid-cols-4 gap-4 items-center">
              <div>No date</div>
              <div>
                <Time variant="timeSince" />
              </div>
              <div>
                <Time variant="timestamp" />
              </div>
              <div>
                <Time variant="short" />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Usage</CardTitle>
          <CardDescription>
            How to use the Time component in your application
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <h3 className="text-lg font-medium">Basic Usage</h3>
            <pre className="bg-muted p-2 rounded mt-2 overflow-x-auto">
              {`<Time date={new Date()} variant="timeSince" />`}
            </pre>
          </div>

          <div>
            <h3 className="text-lg font-medium">Variants</h3>
            <ul className="list-disc list-inside mt-2 space-y-2">
              <li>
                <strong>timeSince</strong> - Live updating relative time using
                timeago-react
              </li>
              <li>
                <strong>timestamp</strong> - ISO-like timestamp in monospace
                font
              </li>
              <li>
                <strong>short</strong> - Concise date-time format
              </li>
            </ul>
          </div>

          <div>
            <h3 className="text-lg font-medium">Options</h3>
            <ul className="list-disc list-inside mt-2 space-y-2">
              <li>
                <strong>updateInterval</strong> - (Optional) Update interval in
                milliseconds for timeSince variant
              </li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
