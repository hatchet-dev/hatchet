import { Button } from '@/components/ui/button';
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
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { useApiError } from '@/lib/hooks';
import { useState } from 'react';
import { Spinner } from '@/components/ui/loading';
import { TenantPaymentMethod } from '@/lib/api/generated/cloud/data-contracts';
import { cloudApi } from '@/lib/api/api';

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
}

export default function PaymentMethods({
  methods = [],
  hasMethods,
}: PaymentMethodsProps) {
  const { tenant } = useOutletContext<TenantContextType>();
  const { handleApiError } = useApiError({});
  const [loading, setLoading] = useState(false);

  const manageClicked = async () => {
    try {
      setLoading(true);
      const link = await cloudApi.billingPortalLinkGet(tenant.metadata.id);
      window.open(link.data.url, '_blank');
    } catch (e) {
      handleApiError(e as any);
    } finally {
      setLoading(false);
    }
  };

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
          <div className="mt-4">
            <Button onClick={manageClicked} variant="default">
              {loading ? <Spinner /> : 'Manage Payment Methods'}
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
              {loading ? <Spinner /> : 'Add a Payment Method'}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
