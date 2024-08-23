import AlarmClock from "../atoms/icons/alarm-clock";
import CalendarDays from "../atoms/icons/calendar-days";
import ChartBarTrendUp from "../atoms/icons/chart-bar-trend-up";
import EyeOpen from "../atoms/icons/eye-open";
import HalfDottedCirclePlay from "../atoms/icons/half-dotted-circle-play";
import Satellite from "../atoms/icons/satellite";
import Particles from "../particles";

interface FeaturesExtraProps {}

const FeaturesExtra: React.FC<FeaturesExtraProps> = () => {
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
            {/* Feature */}
            <div>
              <div className="flex items-center space-x-2 mb-1">
                <EyeOpen />
                <h4 className="font-medium dark:text-slate-50">Observability</h4>
              </div>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                All of your runs are fully searchable, allowing you to quickly
                identify issues. We stream logs, track latency, error rates, or
                custom metrics in your run.
              </p>

              <a href="https://docs.hatchet.run/home/features/errors-and-logging">
                <span className="relative inline-flex items-center mt-2 text-sm text-slate-500 dark:text-slate-300">
                  Learn More{" "}
                  <span className="tracking-normal text-indigo-500 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">
                    -&gt;
                  </span>
                </span>
              </a>
            </div>
            {/* Feature */}
            <div>
              <div className="flex items-center space-x-2 mb-1">
                <HalfDottedCirclePlay />
                <h4 className="font-medium dark:text-slate-50">
                  (Practical) Durable Execution
                </h4>
              </div>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Replay events and manually pick up execution from specific steps
                in your workflow.
              </p>
              <a href="https://docs.hatchet.run/home/features/durable-execution">
                <span className="relative inline-flex items-center mt-2 text-sm text-slate-500 dark:text-slate-300">
                  Learn More{" "}
                  <span className="tracking-normal text-indigo-500 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">
                    -&gt;
                  </span>
                </span>
              </a>
            </div>
            {/* Feature */}
            <div>
              <div className="flex items-center space-x-2 mb-1">
                <AlarmClock />
                <h4 className="font-medium dark:text-slate-50">Cron</h4>
              </div>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Set recurring schedules for functions runs to execute.
              </p>
              <a href="https://docs.hatchet.run/home/features/triggering-runs/cron-trigger">
                <span className="relative inline-flex items-center mt-2 text-sm text-slate-500 dark:text-slate-300">
                  Learn More{" "}
                  <span className="tracking-normal text-indigo-500 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">
                    -&gt;
                  </span>
                </span>
              </a>
            </div>
            {/* Feature */}
            <div>
              <div className="flex items-center space-x-2 mb-1">
                <CalendarDays />
                <h4 className="font-medium dark:text-slate-50">
                  {" "}
                  One-Time Scheduling
                </h4>
              </div>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Schedule a function run to execute at a specific time and date
                in the future.
              </p>
              <a href="https://docs.hatchet.run/home/features/triggering-runs/schedule-trigger">
                <span className="relative inline-flex items-center mt-2 text-sm text-slate-500 dark:text-slate-300">
                  Learn More{" "}
                  <span className="tracking-normal text-indigo-500 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">
                    -&gt;
                  </span>
                </span>
              </a>
            </div>
            {/* Feature */}
            <div>
              <div className="flex items-center space-x-2 mb-1">
                <ChartBarTrendUp />
                <h4 className="font-medium dark:text-slate-50">Spike Protection</h4>
              </div>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Smooth out spikes in traffic and only execute what your system
                can handle.
              </p>
              <a href="https://docs.hatchet.run/home/features/concurrency/overview#setting-concurrency-on-workers">
                <span className="relative inline-flex items-center mt-2 text-sm text-slate-500 dark:text-slate-300">
                  Learn More{" "}
                  <span className="tracking-normal text-indigo-500 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">
                    -&gt;
                  </span>
                </span>
              </a>
            </div>
            {/* Feature */}
            <div>
              <div className="flex items-center space-x-2 mb-1">
                <Satellite />
                <h4 className="font-medium dark:text-slate-50">
                  Incremental Streaming
                </h4>
              </div>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Subscribe to updates as your functions progress in the
                background worker.
              </p>
              <a href="https://docs.hatchet.run/home/features/streaming">
                <span className="relative inline-flex items-center mt-2 text-sm text-slate-500 dark:text-slate-300">
                  Learn More{" "}
                  <span className="tracking-normal text-indigo-500 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">
                    -&gt;
                  </span>
                </span>
              </a>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default FeaturesExtra;
