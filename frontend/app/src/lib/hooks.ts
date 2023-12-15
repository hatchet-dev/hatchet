import { useToast } from "@/components/ui/use-toast";
import { AxiosError } from "axios";
import { Dispatch, SetStateAction } from "react";
import { APIErrors } from "./api";
import { getFieldErrors } from "./utils";

export function useApiError(props: {
  setFieldErrors?: Dispatch<SetStateAction<Record<string, string>>>;
}): {
  handleApiError: (error: AxiosError) => void;
} {
  const { toast } = useToast();

  const handleError = (error: AxiosError) => {
    if (error.response?.status) {
      if (error.response?.status >= 500) {
        toast({
          title: "Error",
          description: "An internal error occurred.",
          duration: 5000,
        });

        return;
      }
    }

    const apiErrors = error.response?.data as APIErrors;

    if (error.response?.status === 400) {
      if (apiErrors && apiErrors.errors && apiErrors.errors.length > 0) {
        const fieldErrors = getFieldErrors(apiErrors);

        if (Object.keys(fieldErrors).length != 0) {
          props.setFieldErrors && props.setFieldErrors(fieldErrors);

          return;
        }
      }
    }

    if (!apiErrors || !apiErrors.errors || apiErrors.errors.length === 0) {
      toast({
        title: "Error",
        description: "An internal error occurred.",
        duration: 5000,
      });

      return;
    }

    for (const error of apiErrors.errors) {
      if (error.description) {
        toast({
          title: "Error",
          description: error.description,
          duration: 5000,
        });
      }
    }
  };

  return {
    handleApiError: handleError,
  };
}
