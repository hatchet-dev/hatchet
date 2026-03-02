package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	ctx := context.Background()

	// Generate a unique suffix for webhook names
	suffix := uuid.New().String()[:8]

	// List existing webhooks
	fmt.Println("Listing existing webhooks...")
	webhooks, err := client.Webhooks().List(ctx, rest.V1WebhookListParams{})
	if err != nil {
		log.Fatalf("failed to list webhooks: %v", err)
	}
	if webhooks.Rows != nil {
		fmt.Printf("Found %d existing webhooks\n", len(*webhooks.Rows))
	}

	// Create a webhook with Basic Auth
	fmt.Println("\nCreating webhook with Basic Auth...")
	basicWebhook, err := client.Webhooks().Create(ctx, features.CreateWebhookOpts{
		Name:               fmt.Sprintf("test-basic-webhook-%s", suffix),
		SourceName:         rest.GENERIC,
		EventKeyExpression: "body.event_type",
		Auth: features.BasicAuth{
			Username: "testuser",
			Password: "testpass",
		},
	})
	if err != nil {
		log.Fatalf("failed to create basic auth webhook: %v", err)
	}
	fmt.Printf("Created webhook: %s\n", basicWebhook.Name)

	// Get the webhook
	fmt.Println("\nGetting webhook...")
	retrieved, err := client.Webhooks().Get(ctx, basicWebhook.Name)
	if err != nil {
		log.Fatalf("failed to get webhook: %v", err)
	}
	fmt.Printf("Retrieved webhook: %s, AuthType: %s\n", retrieved.Name, retrieved.AuthType)

	// Update the webhook
	fmt.Println("\nUpdating webhook...")
	eventKeyExpr := "body.type"
	updated, err := client.Webhooks().Update(ctx, basicWebhook.Name, features.UpdateWebhookOpts{
		EventKeyExpression: &eventKeyExpr,
	})
	if err != nil {
		log.Fatalf("failed to update webhook: %v", err)
	}
	fmt.Printf("Updated webhook expression to: %s\n", updated.EventKeyExpression)

	// Create a webhook with API Key Auth
	fmt.Println("\nCreating webhook with API Key Auth...")
	apiKeyWebhook, err := client.Webhooks().Create(ctx, features.CreateWebhookOpts{
		Name:               fmt.Sprintf("test-apikey-webhook-%s", suffix),
		SourceName:         rest.STRIPE,
		EventKeyExpression: "body.type",
		Auth: features.APIKeyAuth{
			HeaderName: "X-API-Key",
			APIKey:     "sk_test_123",
		},
	})
	if err != nil {
		log.Fatalf("failed to create api key webhook: %v", err)
	}
	fmt.Printf("Created webhook: %s\n", apiKeyWebhook.Name)

	// Create a webhook with HMAC Auth
	fmt.Println("\nCreating webhook with HMAC Auth...")
	hmacWebhook, err := client.Webhooks().Create(ctx, features.CreateWebhookOpts{
		Name:               fmt.Sprintf("test-hmac-webhook-%s", suffix),
		SourceName:         rest.GITHUB,
		EventKeyExpression: "headers['X-GitHub-Event']",
		Auth: features.HMACAuth{
			SigningSecret:       "whsec_test123",
			SignatureHeaderName: "X-Hub-Signature-256",
			Algorithm:           rest.SHA256,
			Encoding:            rest.HEX,
		},
	})
	if err != nil {
		log.Fatalf("failed to create hmac webhook: %v", err)
	}
	fmt.Printf("Created webhook: %s\n", hmacWebhook.Name)

	// List webhooks again to see our new ones
	fmt.Println("\nListing all webhooks...")
	webhooks, err = client.Webhooks().List(ctx, rest.V1WebhookListParams{})
	if err != nil {
		log.Fatalf("failed to list webhooks: %v", err)
	}
	if webhooks.Rows != nil {
		for _, w := range *webhooks.Rows {
			fmt.Printf("  - %s (source: %s, auth: %s)\n", w.Name, w.SourceName, w.AuthType)
		}
	}

	// Clean up - delete the webhooks we created
	fmt.Println("\nCleaning up - deleting test webhooks...")

	err = client.Webhooks().Delete(ctx, basicWebhook.Name)
	if err != nil {
		log.Printf("failed to delete basic webhook: %v", err)
	} else {
		fmt.Printf("Deleted webhook: %s\n", basicWebhook.Name)
	}

	err = client.Webhooks().Delete(ctx, apiKeyWebhook.Name)
	if err != nil {
		log.Printf("failed to delete apikey webhook: %v", err)
	} else {
		fmt.Printf("Deleted webhook: %s\n", apiKeyWebhook.Name)
	}

	err = client.Webhooks().Delete(ctx, hmacWebhook.Name)
	if err != nil {
		log.Printf("failed to delete hmac webhook: %v", err)
	} else {
		fmt.Printf("Deleted webhook: %s\n", hmacWebhook.Name)
	}

	fmt.Println("\nDone!")
}
