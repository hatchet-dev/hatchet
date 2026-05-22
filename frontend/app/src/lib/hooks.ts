import api, { APIError, APIErrors } from './api';
import { controlPlaneApi } from './api/api';
import { getFieldErrors } from './utils';
import { useToast } from '@/components/v1/hooks/use-toast';
import useControlPlane from '@/hooks/use-control-plane';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { Dispatch, SetStateAction } from 'react';

function isAPIErrors(data: unknown): data is APIErrors {
  return (
    !!data &&
    typeof data === 'object' &&
    Array.isArray((data as Partial<APIErrors>).errors)
  );
}

function isAPIError(data: unknown): data is APIError {
  return (
    !!data &&
    typeof data === 'object' &&
    typeof (data as Partial<APIError>).description === 'string'
  );
}

const GENERIC_MESSAGE = 'An internal error occurred.';

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

    const status = error.response?.status;
    const data: unknown = error.response?.data;

    if (status !== undefined && status >= 500) {
      handler([GENERIC_MESSAGE]);
      return;
    }

    if (status === 400 && isAPIErrors(data) && data.errors.length > 0) {
      const fieldErrors = getFieldErrors(data);

      if (Object.keys(fieldErrors).length !== 0) {
        if (props.setFieldErrors) {
          props.setFieldErrors(fieldErrors);
        }
        if (props.setErrors) {
          props.setErrors(Object.values(fieldErrors));
        }
        return;
      }
    }

    if (isAPIErrors(data) && data.errors.length > 0) {
      handler(data.errors.map((e) => e.description || GENERIC_MESSAGE));
      return;
    }

    if (isAPIError(data)) {
      handler([data.description || GENERIC_MESSAGE]);
      return;
    }

    handler([GENERIC_MESSAGE]);
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
