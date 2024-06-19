import { Button } from '@/components/ui/button';
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
  hasMethods?: boolean;
  methods?: TenantPaymentMethod[];
  manageLink?: string;
}

export default function PaymentMethods({
  methods = [],
  manageLink,
  hasMethods,
}: PaymentMethodsProps) {
  return (
    <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Payment Methods
        </h3>
      </div>
      {hasMethods ? (
        <>
          {methods.map((method, i) => {
            const Icon =
              method.brand in ccIcons ? ccIcons[method.brand] : ccIcons.generic;
            return (
              <div key={i} className="flex flex-row items-center gap-4 mb-4">
                <Icon size={24} />
                <div className="flex flex-col">
                  <span>
                    {method.brand.toUpperCase()} *** *** {method.last4}
                  </span>
                  <span>Expires {method.expiration}</span>
                </div>
              </div>
            );
          })}
          {manageLink && (
            <div className="mt-4">
              <a href={manageLink} className="btn btn-primary">
                Manage Payment Methods
              </a>
            </div>
          )}
        </>
      ) : (
        <div className="mt-4">
          <p className="">
            No payment methods added. Payment method is required to upgrade your
            subscription.
          </p>
          {manageLink && (
            <div className="mt-4">
              <Button
                onClick={() => (window.location.href = manageLink)}
                variant="default"
              >
                Add a Payment Method
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
