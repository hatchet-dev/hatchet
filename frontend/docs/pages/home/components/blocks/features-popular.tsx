import AlarmClock from "../atoms/icons/alarm-clock";
import CalendarDays from "../atoms/icons/calendar-days";
import ChartBarTrendUp from "../atoms/icons/chart-bar-trend-up";
import EyeOpen from "../atoms/icons/eye-open";
import HalfDottedCirclePlay from "../atoms/icons/half-dotted-circle-play";
import Satellite from "../atoms/icons/satellite";
import Particles from "../particles";
import { ReactNode } from "react";
import Link from "next/link";

const FeaturesExtra: React.FC = () => {
  return (
    <section className="relative my-6">
      {/* Particles animation */}
      <div className="absolute left-1/2 -translate-x-1/2 top-0 -z-10 w-80 h-80 -mt-24 -ml-32">
        <Particles
          className="absolute inset-0 -z-10"
          quantity={6}
          staticity={30}
        />
      </div>

      <div className="max-w-6xl">
        <div className="pt-0 md:pt-0">
          <div className="grid md:grid-cols-3 gap-8 md:gap-12">
            <FeatureItem
              icon={<EyeOpen />}
              title="Observability"
              description="All of your runs are fully searchable, allowing you to quickly identify issues. We stream logs, track latency, error rates, or custom metrics in your run."
              link="/features/errors-and-logging"
            />
            <FeatureItem
              icon={<HalfDottedCirclePlay />}
              title="(Practical) Durable Execution"
              description="Replay events and manually pick up execution from specific steps in your workflow."
              link="/features/durable-execution"
            />
            <FeatureItem
              icon={<AlarmClock />}
              title="Cron"
              description="Set recurring schedules for functions runs to execute."
              link="/features/triggering-runs/cron-trigger"
            />
            <FeatureItem
              icon={<CalendarDays />}
              title="One-Time Scheduling"
              description="Schedule a function run to execute at a specific time and date in the future."
              link="/features/triggering-runs/schedule-trigger"
            />
            <FeatureItem
              icon={<ChartBarTrendUp />}
              title="Spike Protection"
              description="Smooth out spikes in traffic and only execute what your system can handle."
              link="/features/concurrency/overview#setting-concurrency-on-workers"
            />
            <FeatureItem
              icon={<Satellite />}
              title="Incremental Streaming"
              description="Subscribe to updates as your functions progress in the background worker."
              link="/features/streaming"
            />
          </div>
        </div>
      </div>
    </section>
  );
};

export default FeaturesExtra;


interface FeatureItemProps {
  icon: ReactNode;
  title: string;
  description: string;
  link: string;
}

const FeatureItem: React.FC<FeatureItemProps> = ({
  icon,
  title,
  description,
  link,
}) => {
  return (
    <div>
      <div className="flex items-center space-x-2 mb-1">
        {icon}
        <h4 className="font-medium dark:text-slate-50">{title}</h4>
      </div>
      <p className="text-sm text-slate-600 dark:text-slate-400">
        {description}
      </p>
      <Link href={link} passHref>
        <span className="relative inline-flex items-center mt-2 text-sm text-slate-500 dark:text-slate-300 cursor-pointer">
          Learn More{" "}
          <span className="tracking-normal text-indigo-500 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">
            -&gt;
          </span>
        </span>
      </Link>
    </div>
  );
};
