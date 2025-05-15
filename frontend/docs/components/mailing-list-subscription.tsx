import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useState } from "react";
import { CheckCircle2 } from "lucide-react";

type FormState = {
  success?: boolean;
  error?: string;
} | null;

export function MailingListSubscription() {
  const [state, setState] = useState<FormState>(null);
  const [isLoading, setIsLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setIsLoading(true);
    setState(null);

    const formData = new FormData(e.currentTarget);
    const email = formData.get('email') as string;

    try {
      const response = await fetch('/api/subscribe', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email }),
      });

      if (!response.ok) {
        throw new Error('Failed to subscribe');
      }

      setState({ success: true });
    } catch (error) {
      setState({ success: false, error: 'Failed to subscribe' });
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="w-full max-w-md mx-auto p-6 space-y-4">
      {state?.success ? (
        <div className="flex items-center gap-2 p-3 bg-green-50 dark:bg-green-950/50 rounded-md border border-green-200 dark:border-green-800">
          <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
          <p className="text-sm text-green-700 dark:text-green-300">Thank you for subscribing!</p>
        </div>
      ) : (
        <>
          <div className="text-center space-y-2">
            <h3 className="text-lg font-semibold">Subscribe for more technical deep dives</h3>
            <p className="text-sm text-muted-foreground">
              Stay updated with our latest work. We share insights about distributed systems, workflow engines, and developer tools.
            </p>
          </div>
          
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="flex gap-2 md:flex-row flex-col">
              <Input
                type="email"
                name="email"
                placeholder="Enter your email"
                required
                className="flex-1"
              />
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Subscribing...' : 'Subscribe'}
              </Button>
            </div>
            {state?.error && (
              <p className="text-sm text-red-600">{state.error}</p>
            )}
          </form>
        </>
      )}
    </div>
  );
} 