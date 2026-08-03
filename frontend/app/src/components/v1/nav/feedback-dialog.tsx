import { useToast } from '@/components/v1/hooks/use-toast';
import {
  setFeedbackDialogOpen,
  useFeedbackDialogOpen,
} from '@/components/v1/nav/feedback-dialog-store';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { Textarea } from '@/components/v1/ui/textarea';
import api from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';

export function FeedbackDialog() {
  const open = useFeedbackDialogOpen();
  const { toast } = useToast();
  const { handleApiError } = useApiError({});
  const [message, setMessage] = useState('');

  const submitMutation = useMutation({
    mutationKey: ['feedback:create'],
    mutationFn: async () => {
      await api.feedbackCreate({
        message: message.trim(),
      });
    },
    onSuccess: () => {
      toast({
        title: 'Feedback sent',
        description: 'Thanks for helping us improve Hatchet.',
      });
      setMessage('');
      setFeedbackDialogOpen(false);
    },
    onError: handleApiError,
  });

  return (
    <Dialog open={open} onOpenChange={setFeedbackDialogOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Send feedback</DialogTitle>
          <DialogDescription>
            Share a bug, idea, or anything else with the Hatchet team.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4 py-2">
          <div className="flex flex-col gap-2">
            <Label htmlFor="feedback-message">Feedback</Label>
            <Textarea
              id="feedback-message"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="What's on your mind?"
              rows={5}
              autoFocus
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setFeedbackDialogOpen(false)}
            disabled={submitMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={() => submitMutation.mutate()}
            disabled={!message.trim() || submitMutation.isPending}
          >
            {submitMutation.isPending ? <Spinner /> : 'Send feedback'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
