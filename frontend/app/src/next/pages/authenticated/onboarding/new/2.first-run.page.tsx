import { Button } from '@/next/components/ui/button';
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';

import { useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import BasicLayout from '@/next/components/layouts/basic.layout';
import useMembers, { MembersProvider } from '@/next/hooks/use-members';

export default function OnboardingFirstRunPage() {
  return (
    <MembersProvider>
      <OnboardingFirstRunContent />
    </MembersProvider>
  );
}

function OnboardingFirstRunContent() {
  const { data: members } = useMembers();
  const navigate = useNavigate();

  return (
    <BasicLayout>
      <Card>
        <CardHeader>
          <CardTitle>First Run</CardTitle>
          <CardDescription>
            Tenants are isolated environments that are used to organize your
            workloads.
          </CardDescription>
        </CardHeader>
        <CardContent>{/* The onboarding flow */}</CardContent>
        <CardFooter className="flex justify-between gap-2">
          <DocsButton doc={docs.home.environments} size="icon" />
          <dl className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => {
                navigate(ROUTES.runs.list);
              }}
            >
              Skip
            </Button>
            {members?.length === 1 ? (
              <Button
                onClick={() => {
                  navigate(ROUTES.onboarding.inviteTeam);
                }}
              >
                Continue
              </Button>
            ) : (
              <Button
                onClick={() => {
                  navigate(ROUTES.runs.list);
                }}
              >
                Finish
              </Button>
            )}
          </dl>
        </CardFooter>
      </Card>
    </BasicLayout>
  );
}
