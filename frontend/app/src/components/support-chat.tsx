import { useTheme } from '@/components/hooks/use-theme';
import { useTenantDetails } from '@/hooks/use-tenant';
import { User } from '@/lib/api';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import React, {
  PropsWithChildren,
  useCallback,
  useEffect,
  useMemo,
} from 'react';

interface SupportChatProps {
  user?: User;
}

type DisplayableTheme = 'dark' | 'light';

type PylonWindow = Window & {
  Pylon?: (command: string, ...args: unknown[]) => void;
  pylon?: {
    chat_settings?: Record<string, unknown>;
  };
};

const getPylonWindow = () => window as PylonWindow;

const syncPylonTheme = (theme: DisplayableTheme) => {
  const pylonWindow = getPylonWindow();

  pylonWindow.pylon = {
    ...pylonWindow.pylon,
    chat_settings: {
      ...pylonWindow.pylon?.chat_settings,
      theme,
    },
  };

  pylonWindow.Pylon?.('setTheme', theme);
};

const loadPylonScript = (appId: string) => {
  const existingScript = document.querySelector(
    `script[data-hatchet-pylon-app-id="${appId}"]`,
  );

  if (existingScript) {
    return;
  }

  const pylonScript = `(function(){var e=window;var t=document;var n=function(){n.e(arguments)};n.q=[];n.e=function(e){n.q.push(e)};e.Pylon=n;var r=function(){var e=t.createElement("script");e.setAttribute("type","text/javascript");e.setAttribute("async","true");e.setAttribute("src","https://widget.usepylon.com/widget/${appId}");var n=t.getElementsByTagName("script")[0];n.parentNode.insertBefore(e,n)};if(t.readyState==="complete"){r()}else if(e.addEventListener){e.addEventListener("load",r,false)}})();`;
  const script = document.createElement('script');

  script.dataset.hatchetPylonAppId = appId;
  script.innerHTML = pylonScript;
  document.body.appendChild(script);
};

export const usePylon = () => {
  const { meta } = useApiMeta();
  const { currentlyVisibleTheme } = useTheme();

  const show = useCallback(() => {
    syncPylonTheme(currentlyVisibleTheme);
    getPylonWindow().Pylon?.('show');
  }, [currentlyVisibleTheme]);

  if (!meta?.pylonAppId) {
    return {
      enabled: false,
      show: () => {},
    };
  }

  return {
    enabled: true,
    show,
  };
};

const SupportChat: React.FC<PropsWithChildren & SupportChatProps> = ({
  user,
  children,
}) => {
  const { meta } = useApiMeta();
  const { currentlyVisibleTheme } = useTheme();
  const { tenant } = useTenantDetails();

  const APP_ID = useMemo(() => {
    if (!meta?.pylonAppId) {
      return null;
    }

    return meta?.pylonAppId;
  }, [meta?.pylonAppId]);

  useEffect(() => {
    if (!APP_ID || !user) {
      return;
    }

    const pylonWindow = getPylonWindow();

    pylonWindow.pylon = {
      ...pylonWindow.pylon,
      chat_settings: {
        ...pylonWindow.pylon?.chat_settings,
        app_id: APP_ID,
        email: user.email,
        name: user.name,
        email_hash: user.emailHash,
        theme: currentlyVisibleTheme,
      },
    };
  }, [user, APP_ID, currentlyVisibleTheme]);

  useEffect(() => {
    if (!APP_ID) {
      return;
    }

    loadPylonScript(APP_ID);
  }, [APP_ID]);

  useEffect(() => {
    if (!APP_ID || !user) {
      return;
    }

    const pylonWindow = getPylonWindow();

    pylonWindow.Pylon?.('hideChatBubble');

    pylonWindow.Pylon?.('setNewIssueCustomFields', {
      user_id: user.metadata.id,
      tenant_name: tenant?.name,
      tenant_slug: tenant?.slug,
      tenant_id: tenant?.metadata?.id,
    });
  }, [user, APP_ID, tenant]);

  useEffect(() => {
    if (!APP_ID) {
      return;
    }

    syncPylonTheme(currentlyVisibleTheme);
  }, [APP_ID, currentlyVisibleTheme]);

  return children;
};

export default SupportChat;
