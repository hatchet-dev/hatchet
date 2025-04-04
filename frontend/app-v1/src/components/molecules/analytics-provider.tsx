import { User } from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import React, { PropsWithChildren, useEffect, useMemo } from 'react';

interface AnalyticsProviderProps {
  user: User;
}

const AnalyticsProvider: React.FC<
  PropsWithChildren & AnalyticsProviderProps
> = ({ user, children }) => {
  const meta = useApiMeta();

  const [loaded, setLoaded] = React.useState(false);

  const { tenant } = useTenant();

  const config = useMemo(() => {
    return meta.data?.posthog;
  }, [meta]);

  useEffect(() => {
    if (loaded) {
      return;
    }

    if (tenant && tenant.analyticsOptOut) {
      console.log(
        'Skipping Analytics initialization due to opt-out, we respect user privacy.',
      );
      return;
    }

    if (!config || !tenant) {
      return;
    }

    console.log('Initializing Analytics, opt out in settings.');
    setLoaded(true);
    const posthogScript = `
!function(t,e){var o,n,p,r;e.__SV||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.async=!0,p.src=s.api_host.replace(".i.posthog.com","-assets.i.posthog.com")+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="capture identify alias people.set people.set_once set_config register register_once unregister opt_out_capturing has_opted_out_capturing opt_in_capturing reset isFeatureEnabled onFeatureFlags getFeatureFlag getFeatureFlagPayload reloadFeatureFlags group updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures getActiveMatchingSurveys getSurveys onSessionId".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);
posthog.init('${config.apiKey}',{
  api_host:'${config.apiHost}',
  session_recording: {
      maskAllInputs: true,
      maskTextSelector: "*"
  }
})
`;
    document.head.appendChild(document.createElement('script')).innerHTML =
      posthogScript;
  }, [config, loaded, tenant]);

  useEffect(() => {
    if (!config || !user) {
      return;
    }

    setTimeout(() => {
      (window as any).posthog.identify(
        user.metadata.id, // Required. Replace 'distinct_id' with your user's unique identifier
        { email: user.email, name: user.name }, // $set, optional
        {}, // $set_once, optional
      );
    });
  }, [user, config, tenant]);

  return children;
};

export default AnalyticsProvider;
