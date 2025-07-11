package webhooksv1

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/labstack/echo/v4"
)

func (w *V1WebhooksService) V1WebhookReceive(ctx echo.Context, request gen.V1WebhookReceiveRequestObject) (gen.V1WebhookReceiveResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	webhook := ctx.Get("v1-webhook").(*sqlcv1.V1IncomingWebhook)

	rawBody, err := io.ReadAll(ctx.Request().Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	ok, response := w.validateWebhook(rawBody, *webhook, *ctx.Request())

	if !ok {
		return response.ToResponse()
	}

	payloadMap := make(map[string]interface{})

	if rawBody != nil {
		err := json.Unmarshal(rawBody, &payloadMap)

		if err != nil {
			return gen.V1WebhookReceive400JSONResponse{
				Errors: []gen.APIError{
					{
						Description: fmt.Sprintf("failed to unmarshal request body: %v", err),
					},
				},
			}, nil
		}

		delete(payloadMap, "tenant")
		delete(payloadMap, "v1-webhook")
	}

	eventKey, err := w.celParser.EvaluateIncomingWebhookExpression(webhook.EventKeyExpression, cel.NewInput(
		cel.WithInput(payloadMap),
	),
	)

	if err != nil {
		// TODO: Store this error
		return gen.V1WebhookReceive400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: fmt.Sprintf("failed to evaluate event key expression: %v", err),
				},
			},
		}, nil
	}

	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return gen.V1WebhookReceive400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: fmt.Sprintf("failed to marshal request body: %v", err),
				},
			},
		}, nil
	}

	_, err = w.config.Ingestor.IngestEvent(
		ctx.Request().Context(),
		tenant,
		eventKey,
		payload,
		nil,
		nil,
		nil,
		&webhook.Name,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to ingest event")
	}

	msg := "ok"

	return gen.V1WebhookReceive200JSONResponse(
		gen.V1WebhookReceive200JSONResponse{
			Message: &msg,
		},
	), nil
}

func computeHMACSignature(payload []byte, secret []byte, algorithm sqlcv1.V1IncomingWebhookHmacAlgorithm, encoding sqlcv1.V1IncomingWebhookHmacEncoding) (string, error) {
	var hashFunc func() hash.Hash
	switch algorithm {
	case sqlcv1.V1IncomingWebhookHmacAlgorithmSHA1:
		hashFunc = sha1.New
	case sqlcv1.V1IncomingWebhookHmacAlgorithmSHA256:
		hashFunc = sha256.New
	case sqlcv1.V1IncomingWebhookHmacAlgorithmSHA512:
		hashFunc = sha512.New
	case sqlcv1.V1IncomingWebhookHmacAlgorithmMD5:
		hashFunc = md5.New
	default:
		return "", fmt.Errorf("unsupported HMAC algorithm: %s", algorithm)
	}

	h := hmac.New(hashFunc, secret)
	h.Write(payload)
	signature := h.Sum(nil)

	switch encoding {
	case sqlcv1.V1IncomingWebhookHmacEncodingHEX:
		return hex.EncodeToString(signature), nil
	case sqlcv1.V1IncomingWebhookHmacEncodingBASE64:
		return base64.StdEncoding.EncodeToString(signature), nil
	case sqlcv1.V1IncomingWebhookHmacEncodingBASE64URL:
		return base64.URLEncoding.EncodeToString(signature), nil
	default:
		return "", fmt.Errorf("unsupported HMAC encoding: %s", encoding)
	}
}

type ValidationError struct {
	Http400 *gen.V1WebhookReceive400JSONResponse
	Http403 *gen.V1WebhookReceive403JSONResponse
	Http500 error
}

func (vr ValidationError) ToResponse() (gen.V1WebhookReceiveResponseObject, error) {
	if vr.Http400 != nil {
		return vr.Http400, nil
	}

	if vr.Http403 != nil {
		return vr.Http403, nil
	}

	if vr.Http500 != nil {
		return nil, vr.Http500
	}

	return nil, fmt.Errorf("no validation error set")
}

func (w *V1WebhooksService) validateWebhook(webhookPayload []byte, webhook sqlcv1.V1IncomingWebhook, request http.Request) (
	bool,
	*ValidationError,
) {
	switch webhook.SourceName {
	case sqlcv1.V1IncomingWebhookSourceNameSTRIPE:
		signatureHeader := request.Header.Get(webhook.AuthHmacSignatureHeaderName.String)

		if signatureHeader == "" {
			err := gen.V1WebhookReceive400JSONResponse{
				Errors: []gen.APIError{
					{
						Description: fmt.Sprintf("missing or invalid signature header: %s", webhook.AuthHmacSignatureHeaderName.String),
					},
				},
			}
			return false, &ValidationError{
				Http400: &err,
			}
		}

		splitHeader := strings.Split(signatureHeader, ",")

		timestampHeader := splitHeader[0]
		v1SignatureHeader := splitHeader[1]

		if timestampHeader == "" || v1SignatureHeader == "" {
			err := gen.V1WebhookReceive400JSONResponse{
				Errors: []gen.APIError{
					{
						Description: fmt.Sprintf("missing or invalid signature header: %s", webhook.AuthHmacSignatureHeaderName.String),
					},
				},
			}

			return false, &ValidationError{
				Http400: &err,
			}
		}

		timestamp := strings.TrimPrefix(timestampHeader, "t=")
		signature := strings.TrimPrefix(v1SignatureHeader, "v1=")

		if timestamp == "" || signature == "" {
			err := gen.V1WebhookReceive400JSONResponse{
				Errors: []gen.APIError{
					{
						Description: fmt.Sprintf("missing or invalid signature header: %s", webhook.AuthHmacSignatureHeaderName.String),
					},
				},
			}

			return false, &ValidationError{
				Http400: &err,
			}
		}

		decryptedSigningSecret, err := w.config.Encryption.Decrypt(webhook.AuthHmacWebhookSigningSecret, "v1_webhook_hmac_signing_secret")
		if err != nil {
			return false, &ValidationError{
				Http500: fmt.Errorf("failed to decrypt HMAC signing secret: %w", err),
			}
		}

		algorithm := webhook.AuthHmacAlgorithm.V1IncomingWebhookHmacAlgorithm
		encoding := webhook.AuthHmacEncoding.V1IncomingWebhookHmacEncoding

		signedPayload := fmt.Sprintf("%s.%s", timestamp, webhookPayload)

		expectedSignature, err := computeHMACSignature([]byte(signedPayload), decryptedSigningSecret, algorithm, encoding)

		if err != nil {
			return false, &ValidationError{Http500: fmt.Errorf("failed to compute HMAC signature: %w", err)}
		}

		if signature != expectedSignature {
			err := gen.V1WebhookReceive403JSONResponse{
				Errors: []gen.APIError{
					{
						Description: "invalid HMAC signature",
					},
				},
			}

			return false, &ValidationError{
				Http403: &err,
			}
		}
	case sqlcv1.V1IncomingWebhookSourceNameGENERIC:
	case sqlcv1.V1IncomingWebhookSourceNameGITHUB:
		switch webhook.AuthMethod {
		case sqlcv1.V1IncomingWebhookAuthTypeBASIC:
			username, password, ok := request.BasicAuth()

			if !ok {
				err := gen.V1WebhookReceive403JSONResponse{
					Errors: []gen.APIError{
						{
							Description: "missing or invalid authorization header",
						},
					},
				}

				return false, &ValidationError{
					Http403: &err,
				}
			}

			decryptedPassword, err := w.config.Encryption.Decrypt(webhook.AuthBasicPassword, "v1_webhook_basic_auth_password")

			if err != nil {
				return false, &ValidationError{Http500: fmt.Errorf("failed to decrypt basic auth password: %w", err)}
			}

			if username != webhook.AuthBasicUsername.String || password != string(decryptedPassword) {
				err := gen.V1WebhookReceive403JSONResponse{
					Errors: []gen.APIError{
						{
							Description: "invalid basic auth credentials",
						},
					},
				}

				return false, &ValidationError{
					Http403: &err,
				}
			}
		case sqlcv1.V1IncomingWebhookAuthTypeAPIKEY:
			apiKey := request.Header.Get(webhook.AuthApiKeyHeaderName.String)

			if apiKey == "" {
				err := gen.V1WebhookReceive403JSONResponse{
					Errors: []gen.APIError{
						{
							Description: fmt.Sprintf("missing or invalid api key header: %s", webhook.AuthApiKeyHeaderName.String),
						},
					},
				}

				return false, &ValidationError{
					Http403: &err,
				}
			}

			decryptedApiKey, err := w.config.Encryption.Decrypt(webhook.AuthApiKeyKey, "v1_webhook_api_key")

			if err != nil {
				return false, &ValidationError{Http500: fmt.Errorf("failed to decrypt api key: %w", err)}
			}

			if apiKey != string(decryptedApiKey) {
				err := gen.V1WebhookReceive403JSONResponse{
					Errors: []gen.APIError{
						{
							Description: fmt.Sprintf("invalid api key: %s", webhook.AuthApiKeyHeaderName.String),
						},
					},
				}

				return false, &ValidationError{
					Http403: &err,
				}
			}
		case sqlcv1.V1IncomingWebhookAuthTypeHMAC:
			// TODO: Potentially remove this replace?
			signature := strings.Replace(request.Header.Get(webhook.AuthHmacSignatureHeaderName.String), "sha256=", "", 1)

			if signature == "" {
				err := gen.V1WebhookReceive403JSONResponse{
					Errors: []gen.APIError{
						{
							Description: fmt.Sprintf("missing or invalid signature header: %s", webhook.AuthHmacSignatureHeaderName.String),
						},
					},
				}

				return false, &ValidationError{
					Http403: &err,
				}
			}

			decryptedSigningSecret, err := w.config.Encryption.Decrypt(webhook.AuthHmacWebhookSigningSecret, "v1_webhook_hmac_signing_secret")
			if err != nil {
				return false, &ValidationError{Http500: fmt.Errorf("failed to decrypt HMAC signing secret: %w", err)}
			}

			algorithm := webhook.AuthHmacAlgorithm.V1IncomingWebhookHmacAlgorithm
			encoding := webhook.AuthHmacEncoding.V1IncomingWebhookHmacEncoding

			expectedSignature, err := computeHMACSignature(webhookPayload, decryptedSigningSecret, algorithm, encoding)

			if err != nil {
				return false, &ValidationError{
					Http500: fmt.Errorf("failed to compute HMAC signature: %w", err),
				}
			}

			if signature != expectedSignature {
				err := gen.V1WebhookReceive403JSONResponse{
					Errors: []gen.APIError{
						{
							Description: "invalid HMAC signature",
						},
					},
				}

				return false, &ValidationError{
					Http403: &err,
				}
			}
		default:
			err := gen.V1WebhookReceive400JSONResponse{
				Errors: []gen.APIError{
					{
						Description: fmt.Sprintf("unsupported auth type: %s", webhook.AuthMethod),
					},
				},
			}

			return false, &ValidationError{
				Http400: &err,
			}
		}
	}

	return true, nil
}
