import { WorkersProvider } from '@/next/hooks/use-workers';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/docs-meta-data';
import { ROUTES } from '@/next/lib/routes';
import { WorkerType } from '@/lib/api';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Separator } from '@/next/components/ui/separator';

import { Step, Steps } from '@/components/v1/ui/steps';
import { Button } from '@/next/components/ui/button';
import { useState } from 'react';
function ServiceDetailPageContent() {
  const [activeStep, setActiveStep] = useState(0);
  const handleNext = () => {
    setActiveStep(activeStep + 1);
  };

  useBreadcrumbs(
    () => [
      {
        title: 'Worker Services',
        label: 'New Selfhosted Service',
        url: ROUTES.services.new(WorkerType.SELFHOSTED),
      },
    ],
    [],
  );

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage workers in a worker service">
          New Selfhost Worker Service
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.compute} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="mt-4" />
      <div className="flex flex-col gap-4">
        <Steps>
          <Step
            title="GitHub Repository"
            open={activeStep === 0}
            setOpen={(open: boolean) => setActiveStep(open ? 0 : -1)}
          >
            TODO
            <div className="flex justify-end mt-4">
              <Button onClick={handleNext}>Next</Button>
            </div>
          </Step>
        </Steps>
      </div>
    </BasicLayout>
  );
}

export default function ServiceDetailPage() {
  return (
    <WorkersProvider>
      <ServiceDetailPageContent />
    </WorkersProvider>
  );
}
