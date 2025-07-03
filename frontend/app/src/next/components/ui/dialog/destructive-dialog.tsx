import * as React from 'react';
import { useState, useEffect } from 'react';
import { Input } from '@/next/components/ui/input';
import { Button } from '@/next/components/ui/button';
import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog/dialog';
import { FaBomb } from 'react-icons/fa';
import { Code } from '@/next/components/ui/code';

interface DestructiveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: React.ReactNode;
  confirmationText: string;
  confirmButtonText: string;
  cancelButtonText?: string;
  isLoading?: boolean;
  requireTextConfirmation?: boolean;
  onConfirm: () => void;
  onCancel?: () => void;
  children?: React.ReactNode;
  alertTitle?: string;
  alertDescription?: React.ReactNode;
  hideAlert?: boolean;
  submitVariant?: 'default' | 'destructive';
}

export function DestructiveDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmationText,
  confirmButtonText,
  cancelButtonText = 'Cancel',
  isLoading = false,
  requireTextConfirmation = true,
  onConfirm,
  onCancel,
  children,
  alertTitle = 'Destructive Action',
  alertDescription = 'This action cannot be undone. Please review carefully before proceeding.',
  hideAlert = false,
  submitVariant = 'destructive',
}: DestructiveDialogProps) {
  const [inputValue, setInputValue] = useState('');
  const isConfirmationValid = inputValue === confirmationText;

  // Reset input value when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setInputValue('');
    }
  }, [open]);

  const handleCancel = () => {
    onOpenChange(false);
    if (onCancel) {
      onCancel();
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description ? (
            <DialogDescription>{description}</DialogDescription>
          ) : null}
        </DialogHeader>

        {children}

        {requireTextConfirmation ? (
          <div className="mt-2">
            <label
              htmlFor="confirmation"
              className="block text-sm font-medium mb-2"
            >
              Type{' '}
              <Code language="text" value={confirmationText} variant="inline" />
              to confirm
            </label>
            <Input
              id="confirmation"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              placeholder={`Type "${confirmationText}" to confirm`}
              className={
                !isConfirmationValid && inputValue ? 'border-red-500' : ''
              }
            />
          </div>
        ) : null}

        {!hideAlert && (
          <Alert variant="destructive" className="my-2">
            <FaBomb className="h-4 w-4" />
            <AlertTitle>{alertTitle}</AlertTitle>
            <AlertDescription>{alertDescription}</AlertDescription>
          </Alert>
        )}

        <DialogFooter className="mt-2">
          <Button variant="outline" onClick={handleCancel} disabled={isLoading}>
            {cancelButtonText}
          </Button>
          <Button
            variant={submitVariant}
            onClick={onConfirm}
            loading={isLoading}
            disabled={
              (requireTextConfirmation && !isConfirmationValid) || isLoading
            }
            className="ml-2"
          >
            {confirmButtonText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
