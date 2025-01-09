import { Loading } from '@/components/ui/loading';
import { useTenant } from '@/lib/atoms';
import { UserContextType, MembershipsContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { PropsWithChildren, useState } from 'react';
import { Button } from '@/components/ui/button';
import { DefaultOnboardingAuth } from './platforms/defaults/default-onboarding-auth';
import { DefaultOnboardingWorkflow } from './platforms/defaults/default-onboarding-workflow';
import { OnboardingInterface } from './platforms/_onboarding.interface';
import {
  BiAlarm,
  BiBook,
  BiCalendar,
  BiLogoDiscordAlt,
  BiLogoGoLang,
  BiLogoPython,
  BiLogoTypescript,
} from 'react-icons/bi';
import { IconType } from 'react-icons/lib';
import { typescriptOnboarding } from './platforms/typescript';
import { WorkerListener } from './platforms/defaults/default-worker-listener';
import { Badge } from '@/components/ui/badge';
import { pythonOnboarding } from './platforms/python';
import { goOnboarding } from './platforms/go';

const DEFAULT_OPEN = ['platform'];

const PLATFORMS: {
  name: string;
  icon: IconType;
  onboarding: OnboardingInterface;
}[] = [
  {
    name: 'Python',
    icon: BiLogoPython,
    onboarding: pythonOnboarding,
  },
  {
    name: 'Typescript',
    icon: BiLogoTypescript,
    onboarding: typescriptOnboarding,
  },
  {
    name: 'Go',
    icon: BiLogoGoLang,
    onboarding: goOnboarding,
  },
];

export default function GetStarted() {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;
  const { tenant: currTenant } = useTenant();

  const [steps, setSteps] = useState(DEFAULT_OPEN);
  const [platform, setPlatform] = useState<(typeof PLATFORMS)[0] | undefined>();
  const [existingProject, setExistingProject] = useState<boolean>();

  const [authComplete, setAuthComplete] = useState(false);
  const [workerConnected, setWorkerConnected] = useState(false);
  const [workflowTriggered, setWorkflowTriggered] = useState<string>();

  const progressToStep = (step: string) => {
    setSteps((steps) => [...steps, step]);
  };

  if (!user || !memberships || !currTenant) {
    return <Loading />;
  }

  const Trigger = ({
    children,
    stepComplete,
    i,
  }: PropsWithChildren & { stepComplete: boolean; i: number }) => (
    <AccordionTrigger
      className={`flex items-center justify-start py-4 text-xl font-semibold hover:no-underline	 ${
        stepComplete ? '' : 'opacity-50'
      }`}
      hideChevron={true}
      disabled
    >
      <span className="flex items-center justify-center w-10 h-10 bg-purple-500 text-white rounded-full mr-4">
        {i}
      </span>
      {children}
    </AccordionTrigger>
  );

  const Next = ({ step, disabled }: { step: string; disabled?: boolean }) => (
    <div className="flex justify-end mt-4">
      {!steps.includes(step) && (
        <Button
          onClick={() => progressToStep(step)}
          className="bg-purple-500 hover:bg-purple-600 text-white px-6 py-2 rounded-lg"
          disabled={disabled}
        >
          Continue
        </Button>
      )}
    </div>
  );

  const Skip = () => {
    return (
      <a href="/" className="block">
        <Button variant="ghost">
          <span className="text-xs font-semibold">Skip Tutorial</span>
        </Button>
      </a>
    );
  };

  const PlatformPicker = () => (
    <>
      <div className="flex flex-row gap-4">
        {PLATFORMS.map((item) => (
          <Button
            key={item.name}
            onClick={() => {
              setPlatform(item);
            }}
            className={`flex flex-col items-center justify-center space-y-2 bg-white text-gray-800 w-24 h-24 rounded-lg shadow-md hover:bg-gray-100 ${!platform || platform?.name === item.name ? 'opacity-100' : 'opacity-50'}`}
          >
            <div className="flex items-center justify-center rounded-md w-16 h-16">
              {item.icon && item.icon({ size: 48 })}
            </div>
            <span className="text-xs font-semibold">{item.name}</span>
          </Button>
        ))}
      </div>
      {!platform && (
        <div className="mt-4">
          <Skip />
        </div>
      )}
      {platform && (
        <div className="flex flex-row gap-4 mt-4">
          <Button
            onClick={() => {
              setExistingProject(true);
              progressToStep('setup');
            }}
            className={`flex flex-col items-center justify-center space-y-2 bg-white text-gray-800 rounded-lg shadow-md hover:bg-gray-100 ${existingProject === undefined || existingProject ? 'opacity-100' : 'opacity-50'}`}
          >
            <span className="text-xs font-semibold">
              Add Hatchet to an Existing Project
            </span>
          </Button>
          <Button
            onClick={() => {
              setExistingProject(false);
              progressToStep('setup');
            }}
            className={`flex flex-col items-center justify-center space-y-2 bg-white text-gray-800  rounded-lg shadow-md hover:bg-gray-100 ${existingProject === undefined || !existingProject ? 'opacity-100' : 'opacity-50'}`}
          >
            <span className="text-xs font-semibold">
              Start a New Tutorial Project from Scratch
            </span>
          </Button>
          <Skip />
        </div>
      )}
    </>
  );

  return (
    <div className="flex flex-col items-center w-full h-full overflow-auto">
      <div className="container mx-auto px-4 py-8 lg:px-8 lg:py-12 max-w-4xl">
        <div className="flex flex-col justify-center space-y-4">
          <div className="flex flex-row justify-between mt-10">
            <h1 className="text-3xl font-bold">
              Run your First Workflow in Hatchet
            </h1>

            <a href="/">
              <Button variant="outline">Skip Tutorial</Button>
            </a>
          </div>
          <p>
            <Badge>
              <BiAlarm className="mr-2" /> Estimated 6 min
            </Badge>
          </p>

          <p className="text-gray-600 dark:text-gray-300">
            Get started with Hatchet Cloud by creating a new project, connecting
            your worker, and triggering your first workflow run! At the end of
            this tutorial, you'll have the skills needed to deploy distributed
            workflows with hatchet.{' '}
            <a
              href="https://docs.hatchet.run"
              className="text-purple-300 hover:underline"
            >
              Read the docs.
            </a>
          </p>

          <Accordion type="multiple" value={steps} className="w-full">
            <AccordionItem value="platform">
              <Trigger stepComplete={steps.includes('platform')} i={1}>
                Choose your platform
              </Trigger>
              <AccordionContent className="py-4 px-6 mb-4">
                <PlatformPicker />
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="setup">
              <Trigger stepComplete={steps.includes('setup')} i={2}>
                Setup your application
              </Trigger>
              <AccordionContent className="py-4 px-6">
                {platform &&
                  platform.onboarding.setup({
                    existingProject: !!existingProject,
                  })}
                <Next step="auth" />
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="auth">
              <Trigger stepComplete={steps.includes('auth')} i={3}>
                Generate your Hatchet auth token
              </Trigger>
              <AccordionContent className="py-4 px-6">
                <DefaultOnboardingAuth
                  tenantId={currTenant.metadata.id}
                  onAuthComplete={() => {
                    setAuthComplete(true);
                  }}
                  skip={() => {
                    progressToStep('worker');
                  }}
                />
                <Next step="worker" disabled={!authComplete} />
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="worker">
              <Trigger stepComplete={steps.includes('worker')} i={4}>
                Start your worker
              </Trigger>
              <AccordionContent className="py-4 px-6">
                {platform && platform.onboarding.worker({})}
                <div className="mt-5 text-xl">
                  <WorkerListener
                    tenantId={currTenant.metadata.id}
                    setWorkerConnected={setWorkerConnected}
                  />
                </div>
                <Next step="workflow" disabled={!workerConnected} />
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="workflow">
              <Trigger stepComplete={steps.includes('workflow')} i={5}>
                Trigger your first workflow run
              </Trigger>
              <AccordionContent className="py-4 px-6">
                <DefaultOnboardingWorkflow
                  tenantId={currTenant.metadata.id}
                  workerConnected={workerConnected}
                  setWorkflowTriggered={setWorkflowTriggered}
                />

                <Button
                  disabled={!workflowTriggered}
                  onClick={() => {
                    // Open the latest workflow run in the Hatchet dashboard
                    window.open(`/workflow-runs/${workflowTriggered}`);
                  }}
                  className="bg-purple-500 hover:bg-purple-600 text-white px-6 py-3 rounded-lg mt-5 "
                >
                  Open Latest Run in the Dashboard
                </Button>
              </AccordionContent>
            </AccordionItem>
          </Accordion>

          <div className={`mt-8 pb-10 ${!workflowTriggered ?? 'hidden'}`}>
            {workflowTriggered ? (
              <h2 className="text-2xl font-bold mb-4">What's Next?</h2>
            ) : (
              <h2 className="text-2xl font-bold mb-4">Get Stuck?</h2>
            )}{' '}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Button
                onClick={() => {
                  // Explore advanced topics in the docs
                  window.open(
                    'https://docs.hatchet.run/home/basics/steps',
                    '_blank',
                  );
                }}
                className=" px-6 py-3 rounded-lg"
                variant={'outline'}
              >
                <BiBook className="mr-2" />
                Explore Docs
              </Button>
              <Button
                onClick={() => {
                  // Schedule a meeting with the Hatchet team
                  window.open(
                    'https://discord.com/invite/ZMeUafwH89',
                    '_blank',
                  );
                }}
                className=" px-6 py-3 rounded-lg"
                variant={'outline'}
              >
                <BiLogoDiscordAlt className="mr-2" />
                Join the Hatchet Discord
              </Button>
              <Button
                onClick={() => {
                  // Schedule a meeting with the Hatchet team
                  window.open('https://hatchet.run/office-hours', '_blank');
                }}
                className=" px-6 py-3 rounded-lg"
                variant={'outline'}
              >
                <BiCalendar className="mr-2" />
                Meet with the Hatchet Team
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
