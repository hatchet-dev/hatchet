import makeEmitter from 'better-emitter';

export const globalEmitter = makeEmitter<{
  'new-tenant': { defaultOrganizationId?: string };
}>();
