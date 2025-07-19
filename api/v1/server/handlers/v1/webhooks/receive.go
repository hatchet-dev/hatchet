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
	"errors"
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
	maybeTenant := ctx.Get("tenant")

	if maybeTenant == nil {
		return gen.V1WebhookReceive400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: "tenant not found",
				},
			},
		}, nil
	}

	tenant := maybeTenant.(*dbsqlc.Tenant)
	webhook := ctx.Get("v1-webhook").(*sqlcv1.V1IncomingWebhook)

	rawBody, err := io.ReadAll(ctx.Request().Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	ok, validationError := w.validateWebhook(rawBody, *webhook, *ctx.Request())

	if !ok {
		err := w.config.Ingestor.IngestWebhookValidationFailure(
			ctx.Request().Context(),
			tenant,
			webhook.Name,
			validationError.ErrorText,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to ingest webhook validation failure: %w", err)
		}

		return validationError.ToResponse()
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

		// This could cause unexpected behavior if the payload contains a key named "tenant" or "v1-webhook"
		delete(payloadMap, "tenant")
		delete(payloadMap, "v1-webhook")
	}

	eventKey, err := w.celParser.EvaluateIncomingWebhookExpression(webhook.EventKeyExpression, cel.NewInput(
		cel.WithInput(payloadMap),
	),
	)

	if err != nil {
		if eventKey == "" {
			err = fmt.Errorf("event key evaluted to an empty string")
		}

		ingestionErr := w.config.Ingestor.IngestWebhookValidationFailure(
			ctx.Request().Context(),
			tenant,
			webhook.Name,
			err.Error(),
		)

		if ingestionErr != nil {
			return nil, fmt.Errorf("failed to ingest webhook validation failure: %w", ingestionErr)
		}

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

type HttpResponseCode int

const (
	Http400 HttpResponseCode = iota
	Http403
	Http500
)

type ValidationError struct {
	Code      HttpResponseCode
	ErrorText string
}

func (vr ValidationError) ToResponse() (gen.V1WebhookReceiveResponseObject, error) {
	switch vr.Code {
	case Http400:
		return gen.V1WebhookReceive400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: vr.ErrorText,
				},
			},
		}, nil
	case Http403:
		return gen.V1WebhookReceive403JSONResponse{
			Errors: []gen.APIError{
				{
					Description: vr.ErrorText,
				},
			},
		}, nil
	case Http500:
		return nil, errors.New(vr.ErrorText)
	default:
		return nil, fmt.Errorf("no validation error set")
	}
}

func (w *V1WebhooksService) validateWebhook(webhookPayload []byte, webhook sqlcv1.V1IncomingWebhook, request http.Request) (
	bool,
	*ValidationError,
) {
	switch webhook.SourceName {
	case sqlcv1.V1IncomingWebhookSourceNameSTRIPE:
		signatureHeader := request.Header.Get(webhook.AuthHmacSignatureHeaderName.String)

		if signatureHeader == "" {
			return false, &ValidationError{
				Code:      Http400,
				ErrorText: fmt.Sprintf("missing or invalid signature header: %s", webhook.AuthHmacSignatureHeaderName.String),
			}
		}

		splitHeader := strings.Split(signatureHeader, ",")
		headersMap := make(map[string]string)

		for _, header := range splitHeader {
			parts := strings.Split(header, "=")
			if len(parts) != 2 {
				return false, &ValidationError{
					Code:      Http400,
					ErrorText: fmt.Sprintf("invalid signature header format: %s", webhook.AuthHmacSignatureHeaderName.String),
				}
			}
			headersMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}

		timestampHeader, hasTimestampHeader := headersMap["t"]
		v1SignatureHeader, hasV1SignatureHeader := headersMap["v1"]

		if timestampHeader == "" || v1SignatureHeader == "" || !hasTimestampHeader || !hasV1SignatureHeader {
			return false, &ValidationError{
				Code:      Http400,
				ErrorText: fmt.Sprintf("missing or invalid signature header: %s", webhook.AuthHmacSignatureHeaderName.String),
			}
		}

		timestamp := strings.TrimPrefix(timestampHeader, "t=")
		signature := strings.TrimPrefix(v1SignatureHeader, "v1=")

		if timestamp == "" || signature == "" {
			return false, &ValidationError{
				Code:      Http400,
				ErrorText: fmt.Sprintf("missing or invalid signature header: %s", webhook.AuthHmacSignatureHeaderName.String),
			}
		}

		decryptedSigningSecret, err := w.config.Encryption.Decrypt(webhook.AuthHmacWebhookSigningSecret, "v1_webhook_hmac_signing_secret")
		if err != nil {
			return false, &ValidationError{
				Code:      Http500,
				ErrorText: fmt.Sprintf("failed to decrypt HMAC signing secret: %s", err),
			}
		}

		algorithm := webhook.AuthHmacAlgorithm.V1IncomingWebhookHmacAlgorithm
		encoding := webhook.AuthHmacEncoding.V1IncomingWebhookHmacEncoding

		signedPayload := fmt.Sprintf("%s.%s", timestamp, webhookPayload)

		expectedSignature, err := computeHMACSignature([]byte(signedPayload), decryptedSigningSecret, algorithm, encoding)

		if err != nil {
			return false, &ValidationError{
				Code:      Http500,
				ErrorText: fmt.Sprintf("failed to compute HMAC signature: %s", err),
			}
		}

		if !signaturesMatch(signature, expectedSignature) {
			return false, &ValidationError{
				Code:      Http403,
				ErrorText: "invalid HMAC signature",
			}
		}
	case sqlcv1.V1IncomingWebhookSourceNameGITHUB:
		fallthrough
	case sqlcv1.V1IncomingWebhookSourceNameGENERIC:
		switch webhook.AuthMethod {
		case sqlcv1.V1IncomingWebhookAuthTypeBASIC:
			username, password, ok := request.BasicAuth()

			if !ok {
				return false, &ValidationError{
					Code:      Http403,
					ErrorText: "missing or invalid authorization header",
				}
			}

			decryptedPassword, err := w.config.Encryption.Decrypt(webhook.AuthBasicPassword, "v1_webhook_basic_auth_password")

			if err != nil {
				return false, &ValidationError{
					Code:      Http500,
					ErrorText: fmt.Sprintf("failed to decrypt basic auth password: %s", err),
				}
			}

			if username != webhook.AuthBasicUsername.String || password != string(decryptedPassword) {
				return false, &ValidationError{
					Code:      Http403,
					ErrorText: "invalid basic auth credentials",
				}
			}
		case sqlcv1.V1IncomingWebhookAuthTypeAPIKEY:
			apiKey := request.Header.Get(webhook.AuthApiKeyHeaderName.String)

			if apiKey == "" {
				return false, &ValidationError{
					Code:      Http403,
					ErrorText: fmt.Sprintf("missing or invalid api key header: %s", webhook.AuthApiKeyHeaderName.String),
				}
			}

			decryptedApiKey, err := w.config.Encryption.Decrypt(webhook.AuthApiKeyKey, "v1_webhook_api_key")

			if err != nil {
				return false, &ValidationError{
					Code:      Http500,
					ErrorText: fmt.Sprintf("failed to decrypt api key: %s", err),
				}
			}

			if apiKey != string(decryptedApiKey) {
				return false, &ValidationError{
					Code:      Http403,
					ErrorText: fmt.Sprintf("invalid api key: %s", webhook.AuthApiKeyHeaderName.String),
				}
			}
		case sqlcv1.V1IncomingWebhookAuthTypeHMAC:
			signature := request.Header.Get(webhook.AuthHmacSignatureHeaderName.String)

			if signature == "" {
				return false, &ValidationError{
					Code:      Http403,
					ErrorText: fmt.Sprintf("missing or invalid signature header: %s", webhook.AuthHmacSignatureHeaderName.String),
				}
			}

			decryptedSigningSecret, err := w.config.Encryption.Decrypt(webhook.AuthHmacWebhookSigningSecret, "v1_webhook_hmac_signing_secret")

			if err != nil {
				return false, &ValidationError{
					Code:      Http500,
					ErrorText: fmt.Sprintf("failed to decrypt HMAC signing secret: %s", err),
				}
			}

			algorithm := webhook.AuthHmacAlgorithm.V1IncomingWebhookHmacAlgorithm
			encoding := webhook.AuthHmacEncoding.V1IncomingWebhookHmacEncoding

			expectedSignature, err := computeHMACSignature(webhookPayload, decryptedSigningSecret, algorithm, encoding)

			if err != nil {
				return false, &ValidationError{
					Code:      Http500,
					ErrorText: fmt.Sprintf("failed to compute HMAC signature: %s", err),
				}
			}

			if !signaturesMatch(signature, expectedSignature) {
				return false, &ValidationError{
					Code:      Http403,
					ErrorText: "invalid HMAC signature",
				}
			}
		default:
			return false, &ValidationError{
				Code:      Http400,
				ErrorText: fmt.Sprintf("unsupported auth type: %s", webhook.AuthMethod),
			}
		}
	default:
		return false, &ValidationError{
			Code:      Http400,
			ErrorText: fmt.Sprintf("unsupported source name: %+v", webhook.SourceName),
		}
	}

	return true, nil
}

func signaturesMatch(providedSignature, expectedSignature string) bool {
	providedSignature = strings.TrimSpace(providedSignature)
	expectedSignature = strings.TrimSpace(expectedSignature)

	return strings.EqualFold(removePrefixesFromSignature(providedSignature), removePrefixesFromSignature(expectedSignature))
}

func removePrefixesFromSignature(signature string) string {
	signature = strings.TrimPrefix(signature, "sha1=")
	signature = strings.TrimPrefix(signature, "sha256=")
	signature = strings.TrimPrefix(signature, "sha512=")
	signature = strings.TrimPrefix(signature, "md5=")

	return signature
}
