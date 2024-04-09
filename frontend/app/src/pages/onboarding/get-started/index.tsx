import { TenantSwitcher } from "@/components/molecules/nav-bar/tenant-switcher";
import { Loading } from "@/components/ui/loading";
import { useTenantContext } from "@/lib/atoms";
import { UserContextType, MembershipsContextType } from "@/lib/outlet";
import { useOutletContext } from "react-router-dom";

export default function GetStarted() {
  
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;
  const [currTenant] = useTenantContext();

  if (!user || !memberships || !currTenant) {
    return <Loading />;
  }

  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <div className="lg:p-8 mx-auto w-screen">
          <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
              welcome
              <a href="/">skip onboarding</a>
              <TenantSwitcher memberships={memberships} currTenant={currTenant} />

              </h1>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
