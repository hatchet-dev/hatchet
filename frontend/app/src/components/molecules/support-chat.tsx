import { User } from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import React, { PropsWithChildren, useEffect, useMemo } from 'react';

interface SupportChatProps {
  user: User;
}

const SupportChat: React.FC<PropsWithChildren & SupportChatProps> = ({
  user,
  children,
}) => {
  const meta = useApiMeta();

  const { tenant } = useTenant();

  const APP_ID = useMemo(() => {
    if (!meta.data?.pylonAppId) {
      return null;
    }

    return meta.data.pylonAppId;
  }, [meta]);

  useEffect(() => {
    if (!APP_ID) {
      return;
    }

    const pylonScript = `(function(){var e=window;var t=document;var n=function(){n.e(arguments)};n.q=[];n.e=function(e){n.q.push(e)};e.Pylon=n;var r=function(){var e=t.createElement("script");e.setAttribute("type","text/javascript");e.setAttribute("async","true");e.setAttribute("src","https://widget.usepylon.com/widget/${APP_ID}");var n=t.getElementsByTagName("script")[0];n.parentNode.insertBefore(e,n)};if(t.readyState==="complete"){r()}else if(e.addEventListener){e.addEventListener("load",r,false)}})();`;
    document.body.appendChild(document.createElement('script')).innerHTML =
      pylonScript;
  }, [APP_ID]);

  useEffect(() => {
    if (!APP_ID || !user) {
      return;
    }

    (window as any).pylon = {
      chat_settings: {
        app_id: APP_ID,
        email: user.email,
        name: user.name,
        email_hash: user.emailHash,
      },
    };

    (window as any).Pylon('setNewIssueCustomFields', {
      user_id: user.metadata.id,
      tenant_name: tenant?.name,
      tenant_slug: tenant?.slug,
      tenant_id: tenant?.metadata?.id,
    });
  }, [user, APP_ID, tenant]);

  return children;
};

export default SupportChat;
