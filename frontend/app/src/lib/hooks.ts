import api, { APIErrors } from './api';
import { controlPlaneApi } from './api/api';
import { getFieldErrors } from './utils';
import useControlPlane from '@/hooks/use-control-plane';
import { useToast } from '@/components/hooks/use-toast';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { Dispatch, SetStateAction } from 'react';

export function useApiError(
  props: {
    setFieldErrors?: Dispatch<SetStateAction<Record<string, string>>>;
    // if setErrors is passed, it will be used to pass the errors. otherwise,
    // it will use the global toast.
    setErrors?: (errors: string[]) => void;
  } = {},
): {
  handleApiError: (error: AxiosError) => void;
} {
  const { toast } = useToast();

  const handler = props.setErrors
    ? props.setErrors
    : (errors: string[]) => {
        for (const error of errors) {
          toast({
            title: 'Error',
            description: error,
            duration: 5000,
          });
        }
      };

  const handleError = (error: AxiosError) => {
    console.log(error);
    if (error.response?.status) {
      if (error.response?.status >= 500) {
        handler(['An internal error occurred.']);

        return;
      }
    }

    const apiErrors = error.response?.data as APIErrors;

    if (error.response?.status === 400) {
      if (apiErrors && apiErrors.errors && apiErrors.errors.length > 0) {
        const fieldErrors = getFieldErrors(apiErrors);

        if (Object.keys(fieldErrors).length != 0) {
          if (props.setFieldErrors) {
            props.setFieldErrors(fieldErrors);
          }

          if (props.setErrors) {
            const errors = Object.values(fieldErrors);
            props.setErrors(errors);
          }

          return;
        }
      }
    }

    if (!apiErrors || !apiErrors.errors || apiErrors.errors.length === 0) {
      handler(['An internal error occurred.']);

      return;
    }

    handler(
      apiErrors.errors.map(
        (error) => error.description || 'An internal error occurred.',
      ),
    );
  };

  return {
    handleApiError: handleError,
  };
}

export function useApiMetaIntegrations() {
  const { isControlPlaneEnabled } = useControlPlane();

  const metaQuery = useQuery({
    queryKey: ['metadata:get:integrations', isControlPlaneEnabled],
    queryFn: async () => {
      try {
        return isControlPlaneEnabled
          ? await controlPlaneApi.metadataListIntegrations()
          : await api.metadataListIntegrations();
      } catch (e) {
        console.error('Failed to get API meta integrations', e);
        return null;
      }
    },
    retry: false,
  });

  return metaQuery.data?.data;
}
