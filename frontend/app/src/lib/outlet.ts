import { TenantMember, User } from './api';
import { useOutletContext } from 'react-router-dom';

export type UserContextType = { user: User };

export type MembershipsContextType = { memberships: Array<TenantMember> };

export const useContextFromParent = <T>(newcontext: T) => {
  const curr = useOutletContext<T>();

  return { ...curr, ...newcontext };
};
