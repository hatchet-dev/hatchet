import { TenantPaymentMethod } from '@/lib/api';
import {
  FaCcAmex,
  FaCcDiscover,
  FaCcMastercard,
  FaCcVisa,
  FaCreditCard,
  FaCcDinersClub,
  FaCcJcb,
} from 'react-icons/fa';
import { IconType } from 'react-icons/lib';

const ccIcons: Record<string, IconType> = {
  visa: FaCcVisa,
  mastercard: FaCcMastercard,
  amex: FaCcAmex,
  discover: FaCcDiscover,
  dinersclub: FaCcDinersClub,
  jcb: FaCcJcb,
  generic: FaCreditCard,
};

export interface PaymentMethodsProps {
  methods?: TenantPaymentMethod[];
  manageLink?: string;
}

export default function PaymentMethods({
  methods = [],
  manageLink,
}: PaymentMethodsProps) {
  return (
    <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Payment Methods
        </h3>
      </div>
      <p className="text-gray-700 dark:text-gray-300 my-4"></p>
      {methods.map((method, i) => {
        const Icon =
          method.brand in ccIcons ? ccIcons[method.brand] : ccIcons.generic;
        return (
          <div key={i} className="flex flex-row items-center gap-4">
            <Icon size={24} /> {method.brand.toUpperCase()}
            <span>*** *** {method.last4}</span>
            <span>Expires {method.expiration}</span>
          </div>
        );
      })}
      <br />
      {manageLink && (
        <a href={manageLink} className="underline">
          Manage Payment Methods
        </a>
      )}
    </div>
  );
}
