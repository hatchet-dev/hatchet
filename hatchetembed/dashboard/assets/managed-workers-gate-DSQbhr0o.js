import{v as s,aF as u,b as y,ae as b,j as e,f0 as i,B as t,f2 as d}from"./index-wypz6hY5.js";(function(){try{var a=typeof window<"u"?window:typeof global<"u"?global:typeof globalThis<"u"?globalThis:typeof self<"u"?self:{},n=new a.Error().stack;n&&(a._sentryDebugIds=a._sentryDebugIds||{},a._sentryDebugIds[n]="c94ac337-b2ab-4002-ae1c-2d351b6290dc",a._sentryDebugIdIdentifier="sentry-dbid-c94ac337-b2ab-4002-ae1c-2d351b6290dc")}catch{}})();/**
 * @license lucide-react v1.8.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const f=[["path",{d:"M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z",key:"p7xjir"}]],h=s("cloud",f);/**
 * @license lucide-react v1.8.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const m=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["path",{d:"m4.93 4.93 4.24 4.24",key:"1ymg45"}],["path",{d:"m14.83 9.17 4.24-4.24",key:"1cb5xl"}],["path",{d:"m14.83 14.83 4.24 4.24",key:"q42g0n"}],["path",{d:"m9.17 14.83-4.24 4.24",key:"bqpfvv"}],["circle",{cx:"12",cy:"12",r:"4",key:"4exip2"}]],p=s("life-buoy",m);/**
 * @license lucide-react v1.8.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const k=[["path",{d:"m22 7-8.991 5.727a2 2 0 0 1-2.009 0L2 7",key:"132q7q"}],["rect",{x:"2",y:"4",width:"20",height:"16",rx:"2",key:"izxlao"}]],x=s("mail",k);function w({children:a}){const n=u({from:y.tenantRoute.to}),{cloud:r,featureFlags:o,isCloudEnabled:c}=b(n.tenant),l=(o==null?void 0:o["managed-worker"])==="true";return c&&!r?null:l?e.jsx(e.Fragment,{children:a}):c?e.jsx(i,{icon:e.jsx(p,{className:"h-5 w-5"}),title:"Managed Workers not enabled",description:"Managed Workers aren't enabled for your tenant. Contact support for more information.",actions:e.jsxs(e.Fragment,{children:[e.jsx(t,{leftIcon:e.jsx(x,{className:"h-4 w-4"}),onClick:()=>window.open("mailto:support@hatchet.run","_blank"),children:"Contact us"}),e.jsx(t,{leftIcon:e.jsx(d,{className:"h-4 w-4"}),variant:"outline",onClick:()=>window.history.back(),children:"Go back"})]})}):e.jsx(i,{icon:e.jsx(h,{className:"h-5 w-5"}),title:"Managed Workers are not available",description:"Managed Workers are only available in Hatchet Cloud.",actions:e.jsx(t,{leftIcon:e.jsx(d,{className:"h-4 w-4"}),variant:"outline",onClick:()=>window.history.back(),children:"Go back"})})}export{w as M};
