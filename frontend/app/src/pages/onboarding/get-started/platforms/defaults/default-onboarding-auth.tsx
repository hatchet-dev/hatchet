import { Button } from '@/components/ui/button';

export const DefaultOnboardingAuth: React.FC = () => {
  return (
    <div>
      <p>
        Before you can start your worker, you need to generate an Auth token.
      </p>
      <p>Click the button below to generate your Auth token.</p>
      <Button onClick={() => console.log('Generate Auth token')}>
        Generate Auth token
      </Button>
    </div>
  );
};
// TODO