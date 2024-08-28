"use client";

import { MdRateReview, MdDataUsage, MdAssignment } from "react-icons/md";
import { AiOutlineCodeSandbox, AiOutlineFieldTime } from "react-icons/ai";
import { FaChild, FaRegCircleXmark } from "react-icons/fa6";
import { FiAlertCircle } from "react-icons/fi";
import { GoClockFill } from "react-icons/go";
import { HiLightningBolt } from "react-icons/hi";
import { IoHandRight } from "react-icons/io5";
import Satellite from "../atoms/icons/satellite";
import AlarmClock from "../atoms/icons/alarm-clock";
import CalendarDays from "../atoms/icons/calendar-days";
import ChartBarTrendUp from "../atoms/icons/chart-bar-trend-up";
import HalfDottedCirclePlay from "../atoms/icons/half-dotted-circle-play";
import EyeOpen from "../atoms/icons/eye-open";
import CardBlock from "./card-block";


export default function FeaturesNavigator() {
  return (
    <section>
      <div className="relative max-w-full mx-auto">
        <CardBlock
          title="Explore Hatchet's Features"
          description="Discover the powerful features Hatchet offers to enhance your workflow management."
          items={[
            {
              icon: <ChartBarTrendUp />,
              title: "Concurrency Strategies",
              description:
                "Manage and distribute tasks across multiple workers to prevent bottlenecks and ensure fair resource allocation.",
              link: "/features/concurrency/overview",
            },
            {
              icon: <HalfDottedCirclePlay />,
              title: "Durable Execution",
              description:
                "Replay events and manually pick up execution from specific steps in your workflow.",
              link: "/features/durable-execution",
            },
            {
              icon: <AiOutlineCodeSandbox size={18} />,
              title: "Retries",
              description:
                "Automatically retry tasks that fail to ensure that your workflows complete successfully.",
              link: "/features/retries/overview",
            },
            {
              icon: <GoClockFill size={18} />,
              title: "Timeouts",
              description:
                "Set precise time limits for tasks to avoid long-running processes from stalling your workflows.",
              link: "/features/timeouts",
            },
            {
              icon: <EyeOpen />,
              title: "Errors and Logging",
              description:
                "Track and log errors to quickly identify issues with comprehensive observability into your task executions.",
              link: "/features/errors-and-logging",
            },
            {
              icon: <FiAlertCircle size={18} />,
              title: "On Failure Step",
              description:
                "Define specific actions to take when a step fails, ensuring resilient workflows.",
              link: "/features/on-failure-step",
            },
            {
              icon: <Satellite />,
              title: "Incremental Streaming",
              description:
                "Subscribe to updates as your functions progress in the background worker.",
              link: "/features/streaming",
            },
            {
              icon: <HiLightningBolt size={18} />,
              title: "Event Trigger",
              description:
                "Automatically trigger workflows in response to specific events, ensuring timely task execution.",
              link: "/features/triggering-runs/event-trigger",
            },
            {
              icon: <AlarmClock />,
              title: "Cron Scheduling",
              description:
                "Set up recurring tasks with cron schedules, allowing you to automate routine workflows.",
              link: "/features/triggering-runs/cron-trigger",
            },
            {
              icon: <CalendarDays />,
              title: "Schedule Trigger",
              description:
                "Schedule one-time tasks to run at a specific time and date, perfect for time-sensitive operations.",
              link: "/features/triggering-runs/schedule-trigger",
            },
            {
              icon: <MdRateReview size={18} />,
              title: "Global Rate Limits",
              description:
                "Implement rate limiting across your entire infrastructure to protect your systems from overload.",
              link: "/features/rate-limits",
            },
            {
              icon: <MdAssignment size={18} />,
              title: "Worker Assignment",
              description:
                "Automatically assign tasks to the most appropriate workers based on current workloads.",
              link: "/features/worker-assignment/overview",
            },
            {
              icon: <MdDataUsage size={18} />,
              title: "Additional Metadata",
              description:
                "Attach and manage custom metadata with your tasks for more insightful workflow tracking.",
              link: "/features/additional-metadata",
            },
            {
              icon: <IoHandRight size={18} />,
              title: "Manual Slot Release",
              description:
                "Manually release reserved slots for tasks to give you greater control over workflow execution.",
              link: "/features/advanced/manual-slot-release",
            },
            {
              icon: <FaRegCircleXmark size={18} />,
              title: "Cancellation",
              description:
                "Gracefully cancel workflows or tasks when they are no longer needed.",
              link: "/features/cancellation",
            },
            {
              icon: <FaChild size={18} />,
              title: "Child Workflows",
              description:
                "Easily manage complex workflows by creating and executing nested workflows.",
              link: "/features/child-workflows",
            },
            {
              icon: <AiOutlineFieldTime size={22} />,
              title: "Webhooks",
              description:
                "Integrate with external systems and services by triggering webhooks from your workflows.",
              link: "/features/webhooks",
            },
          ]}
        />
      </div>
    </section>
  );
}
