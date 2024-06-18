package lago

import (
	"fmt"

	"github.com/getlago/lago-go-client"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type LagoBilling struct {
	client *lago.Client
}

type LagoBillingOpts struct {
	ApiKey  string
	BaseUrl string
}

func NewLagoBilling(opts *LagoBillingOpts) (*LagoBilling, error) {
	if opts.ApiKey == "" || opts.BaseUrl == "" {
		return nil, fmt.Errorf("api key and base url are required if lago is enabled")
	}

	lagoClient := lago.New().SetBaseURL(opts.BaseUrl).SetApiKey(opts.ApiKey)

	return &LagoBilling{
		client: lagoClient,
	}, nil
}

func (l *LagoBilling) UpsertTenant(tenant db.TenantModel) error {
	// customerInput := &lago.CustomerInput{
	// 	ExternalID:              "5eb02857-a71e-4ea2-bcf9-57d3a41bc6ba",
	// 	Name:                    "Gavin Belson",
	// 	Email:                   "dinesh@piedpiper.test",
	// 	AddressLine1:            "5230 Penfield Ave",
	// 	AddressLine2:            "",
	// 	City:                    "Woodland Hills",
	// 	Country:                 "US",
	// 	Currency:                "USD",
	// 	State:                   "CA",
	// 	Zipcode:                 "75001",
	// 	LegalName:               "Coleman-Blair",
	// 	LegalNumber:             "49-008-2965",
	// 	TaxIdentificationNumber: "EU123456789",
	// 	Phone:                   "+330100000000",
	// 	Timezone:                "Europe/Paris",
	// 	URL:                     "http://hooli.com",
	// 	BillingConfiguration: &CustomerBillingConfigurationInput{
	// 		InvoiceGracePeriod: 3,
	// 		PaymentProvider:    lago.PaymentProviderStripe,
	// 		ProviderCustomerID: "cus_123456789",
	// 		SyncWithProvider:   true,
	// 		DocumentLocale:     "fr",
	// 	},
	// 	Metadata: []*lago.CustomerMetadataInput{
	// 		{
	// 			Key:              "Purchase Order",
	// 			Value:            "123456789",
	// 			DisplayInInvoice: true,
	// 		},
	// 	},
	// }

	// customer, err := l.client.Customer().Create(customerInput)

	// if err != nil {
	// 	return err
	// }

	return nil
}
