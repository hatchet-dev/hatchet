import { useOutletContext } from 'react-router-dom';
import { Tenant, TenantMember, User } from './api';

export type TenantContextType = { tenant: Tenant };

export type UserContextType = { user: User };

export type MembershipsContextType = { memberships: Array<TenantMember> };

export const useContextFromParent = <T>(newcontext: T) => {
  const curr = useOutletContext<T>();

  return { ...curr, ...newcontext };
};
