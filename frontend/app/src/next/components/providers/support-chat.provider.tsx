import useApiMeta from '@/next/hooks/use-api-meta';
import useUser from '@/next/hooks/use-user';
import { useTenant } from '@/next/hooks/use-tenant';
import React, { PropsWithChildren, useEffect, useMemo } from 'react';

interface PylonWindow extends Window {
  config?: {
    chat_settings: {
      app_id: string;
      email: string;
      name: string;
      email_hash: string;
    };
  };
  Pylon?: (method: string, ...args: any[]) => void;
}

const SupportChat: React.FC<PropsWithChildren> = ({ children }) => {
  const { data: user } = useUser();
  const meta = useApiMeta();
  const { tenant } = useTenant();

  const APP_ID = useMemo(
    () => meta.oss?.pylonAppId ?? null,
    [meta.oss?.pylonAppId],
  );

  useEffect(() => {
    if (!APP_ID || !user) {
      return;
    }

    // Initialize Pylon script
    const script = document.createElement('script');
    script.innerHTML = `(function(){var e=window;var t=document;var n=function(){n.e(arguments)};n.q=[];n.e=function(e){n.q.push(e)};e.Pylon=n;var r=function(){var e=t.createElement("script");e.setAttribute("type","text/javascript");e.setAttribute("async","true");e.setAttribute("src","https://widget.usepylon.com/widget/${APP_ID}");var n=t.getElementsByTagName("script")[0];n.parentNode.insertBefore(e,n)};if(t.readyState==="complete"){r()}else if(e.addEventListener){e.addEventListener("load",r,false)}})();`;
    document.body.appendChild(script);

    // Configure Pylon settings
    (window as PylonWindow).config = {
      chat_settings: {
        app_id: APP_ID,
        email: user.email || '',
        name: user.name || '',
        email_hash: user.emailHash || '',
      },
    };

    // Set custom fields
    (window as PylonWindow).Pylon?.('setNewIssueCustomFields', {
      user_id: user.metadata.id,
      tenant_name: tenant?.name,
      tenant_slug: tenant?.slug,
      tenant_id: tenant?.metadata?.id,
    });

    // Cleanup
    return () => {
      document.body.removeChild(script);
    };
  }, [APP_ID, user, tenant]);

  return children;
};

export default SupportChat;
