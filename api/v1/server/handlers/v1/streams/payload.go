package streams

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var errNotEntitled = errors.New("durable streams are not enabled for this tenant")

func (s *V1StreamsService) V1StreamPayloadUpload(ctx echo.Context, request gen.V1StreamPayloadUploadRequestObject) (gen.V1StreamPayloadUploadResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	reqCtx := ctx.Request().Context()

	entitled, err := s.config.V1.TenantEntitlement().HasEntitlement(reqCtx, tenant.ID, repository.EntitlementDurableStreams)

	if err != nil {
		return nil, err
	}

	if !entitled {
		return gen.V1StreamPayloadUpload403JSONResponse(apierrors.NewAPIErrors(errNotEntitled.Error())), nil
	}

	payload, err := readPayloadPart(request.Body)

	if err != nil {
		return gen.V1StreamPayloadUpload400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	if len(payload) > repository.MaxStreamUploadedPayloadBytes {
		return gen.V1StreamPayloadUpload413JSONResponse(apierrors.NewAPIErrors(fmt.Sprintf("payload exceeds maximum size of %d bytes", repository.MaxStreamUploadedPayloadBytes))), nil
	}

	if len(payload) == 0 {
		return gen.V1StreamPayloadUpload400JSONResponse(apierrors.NewAPIErrors("payload is required")), nil
	}

	pre, post := s.config.V1.TenantLimit().Meter(reqCtx, nil, sqlcv1.LimitResourceSTREAMMESSAGE, tenant.ID, repository.StreamPayloadMessageUnits(len(payload)))

	if err := pre(); errors.Is(err, repository.ErrResourceExhausted) {
		return gen.V1StreamPayloadUpload429JSONResponse(apierrors.NewAPIErrors("resource exhausted: stream message limit exceeded for tenant")), nil
	} else if err != nil {
		return nil, err
	}

	ref, err := s.config.V1.Streams().InsertStreamPayload(reqCtx, tenant.ID, payload)

	if err != nil {
		return nil, err
	}

	post()

	encoded, err := repository.EncodeStreamPayloadRef(ref)

	if err != nil {
		return nil, err
	}

	return gen.V1StreamPayloadUpload200JSONResponse{Ref: encoded}, nil
}

func (s *V1StreamsService) V1StreamPayloadGet(ctx echo.Context, request gen.V1StreamPayloadGetRequestObject) (gen.V1StreamPayloadGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	if !authz.CanViewPayloads(ctx) {
		return gen.V1StreamPayloadGet403JSONResponse(apierrors.NewAPIErrors("not permitted to view payloads")), nil
	}

	entitled, err := s.config.V1.TenantEntitlement().HasEntitlement(ctx.Request().Context(), tenant.ID, repository.EntitlementDurableStreams)

	if err != nil {
		return nil, err
	}

	if !entitled {
		return gen.V1StreamPayloadGet403JSONResponse(apierrors.NewAPIErrors(errNotEntitled.Error())), nil
	}

	ref, err := repository.DecodeStreamPayloadRef(request.Params.Ref)

	if err != nil {
		return gen.V1StreamPayloadGet400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	payload, err := s.config.V1.Streams().GetStreamPayload(ctx.Request().Context(), tenant.ID, ref)

	if errors.Is(err, repository.ErrStreamPayloadNotFound) {
		return gen.V1StreamPayloadGet404JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	if err != nil {
		return nil, err
	}

	return gen.V1StreamPayloadGet200ApplicationoctetStreamResponse{
		Body:          bytes.NewReader(payload),
		ContentLength: int64(len(payload)),
	}, nil
}

func readPayloadPart(reader *multipart.Reader) ([]byte, error) {
	for {
		part, err := reader.NextPart()

		if errors.Is(err, io.EOF) {
			return nil, errors.New(`multipart body has no "payload" part`)
		}

		if err != nil {
			return nil, err
		}

		if part.FormName() != "payload" {
			continue
		}

		// one byte over the limit is enough to reject it
		return io.ReadAll(io.LimitReader(part, repository.MaxStreamUploadedPayloadBytes+1))
	}
}
