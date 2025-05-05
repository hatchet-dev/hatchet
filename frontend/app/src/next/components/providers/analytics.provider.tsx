import * as Sentry from '@sentry/browser';
import useApiMeta from '@/next/hooks/use-api-meta';
import useUser from '@/next/hooks/use-user';
import useTenant from '@/next/hooks/use-tenant';
import React, { PropsWithChildren, useEffect } from 'react';

const AnalyticsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const { data: user } = useUser();
  const meta = useApiMeta();

  const [loaded, setLoaded] = React.useState(false);

  const { tenant } = useTenant();

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

    if (!tenant) {
      return;
    }

    if (meta.oss?.decipherDsn) {
      console.log('Initializing Sentry');
      Sentry.init({
        dsn: meta.oss?.decipherDsn,
        integrations: [
          Sentry.browserTracingIntegration(),
          Sentry.replayIntegration({
            maskAllText: true,
            unmask: ['a', 'button', 'h1', 'h2', 'h3', 'p', 'label'],
            blockAllMedia: false,
            maskAllInputs: true,
          }),
        ],
        replaysOnErrorSampleRate: 1.0,
        replaysSessionSampleRate: 1.0,
        tracesSampleRate: 1.0,
      });
      setLoaded(true);
    }

    if (meta.oss?.posthog) {
      const posthogScript = `
!function(t,e){var o,n,p,r;e.__SV||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.async=!0,p.src=s.api_host.replace(".i.posthog.com","-assets.i.posthog.com")+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="capture identify alias people.set people.set_once set_config register register_once unregister opt_out_capturing has_opted_out_capturing opt_in_capturing reset isFeatureEnabled onFeatureFlags getFeatureFlag getFeatureFlagPayload reloadFeatureFlags group updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures getActiveMatchingSurveys getSurveys onSessionId".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);
posthog.init('${meta.oss?.posthog.apiKey}',{
  api_host:'${meta.oss?.posthog.apiHost}',
  session_recording: {
      maskAllInputs: true,
      maskTextSelector: "*"
  }
})
`;
      document.head.appendChild(document.createElement('script')).innerHTML =
        posthogScript;
      setLoaded(true);
    }
  }, [loaded, tenant, meta.oss?.decipherDsn, meta.oss?.posthog]);

  useEffect(() => {
    if (!meta.oss?.posthog || !meta.oss?.decipherDsn || !user) {
      return;
    }

    setTimeout(() => {
      if (meta.oss?.posthog) {
        (window as any).posthog.identify(
          user.metadata.id, // Required. Replace 'distinct_id' with your user's unique identifier
          { email: user.email, name: user.name }, // $set, optional
          {}, // $set_once, optional
        );
      }

      if (meta.oss?.decipherDsn) {
        // Set user information in Decipher via the Sentry TypeScript SDK
        Sentry.setUser({
          email: user.email, // Recommended identifier to set
          id: user.metadata.id, // Optional: use if email not available
          username: user.name, // Optional: use if email not available
          account: tenant?.name, // Recommended: Which account/organization is this user a member of?
          created_at: user.metadata.createdAt, // Recommended: date this user signed up for your product.
          // Additional user information can be added here
        });
      }
    });
  }, [user, meta.oss?.posthog, meta.oss?.decipherDsn, tenant]);

  return children;
};

export default AnalyticsProvider;
