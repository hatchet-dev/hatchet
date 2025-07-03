import {
  FaCcAmex,
  FaCcDiscover,
  FaCcMastercard,
  FaCcVisa,
  FaCreditCard,
  FaCcDinersClub,
  FaCcJcb,
} from 'react-icons/fa';

import { LuBanknote } from 'react-icons/lu';

import { IconType } from 'react-icons/lib';
import useBilling from '@/next/hooks/use-billing';
import { Button } from '@/next/components/ui/button';

const ccIcons: Record<string, IconType> = {
  visa: FaCcVisa,
  mastercard: FaCcMastercard,
  amex: FaCcAmex,
  discover: FaCcDiscover,
  dinersclub: FaCcDinersClub,
  jcb: FaCcJcb,
  generic: FaCreditCard,
  link: LuBanknote,
};

export function PaymentMethods() {
  const { billing } = useBilling();

  const manageClicked = async () => {
    const link = await billing.getManagedUrl;
    if (link.data) {
      window.location.href = link.data;
    }
  };

  return (
    <>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Payment Methods
        </h3>
      </div>
      {billing.hasPaymentMethods ? (
        <>
          {billing.state?.paymentMethods?.map((method, i) => {
            const Icon =
              method.brand in ccIcons ? ccIcons[method.brand] : ccIcons.generic;
            return (
              <div key={i} className="flex flex-row items-center gap-4 mb-4">
                <div className="flex flex-col mt-4 text-sm">
                  <div className="flex flex-row gap-2 items-center">
                    <Icon size={24} />
                    {method.brand.toUpperCase()}
                    {method.last4 ? ` *** *** ${method.last4} ` : null}
                    {method.expiration ? `(Expires {method.expiration})` : null}
                  </div>
                </div>
              </div>
            );
          })}
          <div className="mt-4">
            <Button
              onClick={manageClicked}
              loading={billing.getManagedUrl.isLoading}
            >
              Manage Payment Methods
            </Button>
          </div>
        </>
      ) : (
        <div className="mt-4">
          <p className="">
            No payment methods added. Payment method is required to upgrade your
            subscription.
          </p>
          <div className="mt-4">
            <Button onClick={manageClicked} variant="default">
              Add a Payment Method
            </Button>
          </div>
        </div>
      )}
    </>
  );
}
