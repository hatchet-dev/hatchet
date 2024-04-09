import { Loading } from '@/components/ui/loading';
import { useTenantContext } from '@/lib/atoms';
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
import { BiLogoGoLang, BiLogoPython, BiLogoTypescript } from 'react-icons/bi';
import { IconType } from 'react-icons/lib';

const DEFAULT_OPEN = ['platform'];

const ALL_OPEN = ['platform', 'auth', 'worker', 'workflow'];

const PLATFORMS: {
  name: string;
  icon: IconType;
  onboarding: OnboardingInterface;
}[] = [
  {
    name: 'Python',
    icon: BiLogoPython,
    onboarding: {
      platform: () => <div>Python</div>,
      worker: () => <div>Python</div>,
    },
  },
  {
    name: 'Typescript',
    icon: BiLogoTypescript,
    onboarding: {
      platform: () => <div>Typescript</div>,
      worker: () => <div>Typescript</div>,
    },
  },
  {
    name: 'Go',
    icon: BiLogoGoLang,
    onboarding: {
      platform: () => <div>GoLang</div>,
      worker: () => <div>GoLang</div>,
    },
  },
];

export default function GetStarted() {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;
  const [currTenant] = useTenantContext();

  const [steps, setSteps] = useState(DEFAULT_OPEN);
  const [platform, setPlatform] = useState<(typeof PLATFORMS)[0] | undefined>();

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
      className={`flex items-center justify-start py-4 text-xl font-semibold ${
        stepComplete ? '' : 'opacity-50'
      }`}
      hideChevron={true}
      disabled
    >
      <span className="flex items-center justify-center w-10 h-10 bg-blue-500 text-white rounded-full mr-4">
        {i}
      </span>
      {children}
    </AccordionTrigger>
  );

  const Next = ({ step }: { step: string }) => (
    <div className="flex justify-end mt-4">
      {!steps.includes(step) && (
        <Button
          onClick={() => progressToStep(step)}
          className="bg-blue-500 text-white px-6 py-2 rounded-lg"
        >
          Next
        </Button>
      )}
    </div>
  );

  const PlatformPicker = () => (
    <div className="flex flex-row gap-4">
      {PLATFORMS.map((item) => (
        <Button
          key={item.name}
          onClick={() => {
            setPlatform(item);
            progressToStep('setup');
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
  );

  return (
    <div className="flex flex-col items-center justify-center w-full h-full">
      <div className="container mx-auto px-4 py-8 lg:px-8 lg:py-12 max-w-4xl">
        <div className="flex flex-col justify-center space-y-8">
          <h1 className="text-3xl font-bold">Learn Hatchet in 5 steps</h1>
          <p className="text-lg">
            Set up a project and run your first workflow to understand the
            fundamentals of building your application.{' '}
            <a
              href="https://docs.hatchet.run"
              className="text-blue-500 hover:underline"
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
                {platform && platform.onboarding.platform({})}
                <Next step="auth" />
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="auth">
              <Trigger stepComplete={steps.includes('auth')} i={3}>
                Generate your Auth token
              </Trigger>
              <AccordionContent className="py-4 px-6">
                <DefaultOnboardingAuth />
                <Next step="worker" />
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="worker">
              <Trigger stepComplete={steps.includes('worker')} i={4}>
                Start your worker
              </Trigger>
              <AccordionContent className="py-4 px-6">
                {platform && platform.onboarding.worker({})}
                <Next step="workflow" />
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="workflow">
              <Trigger stepComplete={steps.includes('workflow')} i={5}>
                Trigger your first workflow run
              </Trigger>
              <AccordionContent className="py-4 px-6">
                <DefaultOnboardingWorkflow />
                {/* TODO continue to inspect workflow run */}
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      </div>
    </div>
  );
}
