import { Link, useNavigate } from "react-router-dom";
import { UserRegisterForm } from "./components/user-register-form";
import { cn } from "@/lib/utils";
import { buttonVariants } from "@/components/ui/button";
import { useMutation } from "@tanstack/react-query";
import api, { UserRegisterRequest } from "@/lib/api";
import { useState } from "react";
import { useApiError } from "@/lib/hooks";

export default function Login() {
  const navigate = useNavigate();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createMutation = useMutation({
    mutationKey: ["user:create"],
    mutationFn: async (data: UserRegisterRequest) => {
      await api.userCreate(data);
    },
    onSuccess: () => {
      navigate("/");
    },
    onError: handleApiError,
  });

  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <Link
          to="/auth/login"
          className={cn(
            buttonVariants({ variant: "ghost" }),
            "absolute right-4 top-4 md:right-8 md:top-8"
          )}
        >
          Login
        </Link>
        <div className="lg:p-8 mx-auto w-screen">
          <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                Create an account
              </h1>
              <p className="text-sm text-muted-foreground">
                Create an account to get started.
              </p>
            </div>
            <UserRegisterForm
              isLoading={createMutation.isPending}
              onSubmit={createMutation.mutate}
              fieldErrors={fieldErrors}
            />
            <p className="px-8 text-center text-sm text-muted-foreground">
              By clicking continue, you agree to our{" "}
              <Link
                to="/terms"
                className="underline underline-offset-4 hover:text-primary"
              >
                Terms of Service
              </Link>{" "}
              and{" "}
              <Link
                to="/privacy"
                className="underline underline-offset-4 hover:text-primary"
              >
                Privacy Policy
              </Link>
              .
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
