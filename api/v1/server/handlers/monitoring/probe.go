package monitoring

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/client"
	clientconfig "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func (m *MonitoringService) MonitoringPostRunProbe(ctx echo.Context, request gen.MonitoringPostRunProbeRequestObject) (gen.MonitoringPostRunProbeResponseObject, error) {
	if !m.enabled {
		m.l.Error().Msg("monitoring is not enabled")
		return gen.MonitoringPostRunProbe403JSONResponse{}, nil
	}

	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	if !slices.Contains[[]string](m.permittedTenants, tenantId) {

		err := fmt.Errorf("tenant is not a monitoring tenant for this instance")

		if len(m.permittedTenants) > 0 {
			m.l.Error().Err(err).Msgf("monitoring tenants are %v", m.permittedTenants)
		} else {
			m.l.Error().Err(err).Msg("no monitoring tenants are configured")
		}

		return gen.MonitoringPostRunProbe403JSONResponse{}, nil
	}

	token, err := getBearerTokenFromRequest(ctx.Request())

	if err != nil {
		return nil, err
	}

	events := make(chan string, 50)
	stepChan := make(chan string, 50)
	errorChan := make(chan error, 50)

	cancellableContext, cancel := context.WithTimeout(ctx.Request().Context(), m.probeTimeout)

	defer cancel()

	cf := clientconfig.ClientConfigFile{
		Token:     token,
		TenantId:  tenantId,
		Namespace: randomNamespace(),
		TLS: clientconfig.ClientTLSConfigFile{
			Base: shared.TLSConfigFile{
				TLSRootCAFile: m.tlsRootCAFile,
			},
		},
	}

	cleanup, err := m.run(cancellableContext, cf, m.workflowName, events, stepChan, errorChan)
	if err != nil {
		m.l.Error().Msgf("error running probe: %s", err)
		return nil, err
	}

	defer cleanup()
	// Stream events are not necessarily received in order so we don't distinguish between them
	messages := []string{"This is a stream event", "This is a stream event"}
	messageIndex := 0
	stepMessages := []string{"step-one", "step-two"}
	for {

		select {
		case <-cancellableContext.Done():

			if cancellableContext.Err() == context.DeadlineExceeded {
				m.l.Error().Msg("timed out waiting for probe to complete")
				return nil, fmt.Errorf("timed out waiting for probe to complete")
			}

		case err := <-errorChan:
			m.l.Error().Msgf("error during probe: %s", err)
			return nil, err

		case e := <-events:
			if !strings.HasPrefix(e, messages[messageIndex]) {
				return nil, fmt.Errorf("expected message %s, to start with %s", messages[messageIndex], e)
			}
			messageIndex++

			if messageIndex == len(messages) {
				for i := range stepMessages {
					stepMessage := <-stepChan
					if stepMessage != stepMessages[i] {

						return nil, fmt.Errorf("probe did not complete successfully - step messages failed")
					}
				}
				return nil, nil
			}
		}
	}

}

type probeEvent struct {
	UniqueStreamId string
}

type stepOneOutput struct {
	Message        string `json:"message"`
	UniqueStreamId string
}

func (m *MonitoringService) run(ctx context.Context, cf clientconfig.ClientConfigFile, workflowName string, events chan<- string, stepChan chan<- string, errors chan<- error) (func(), error) {

	c, err := client.NewFromConfigFile(
		&cf, client.WithLogLevel(m.l.GetLevel().String()),
	)

	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(

		worker.WithClient(
			c,
		), worker.WithLogLevel(m.l.GetLevel().String()),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}
	streamKey := "streamKey"
	streamValue := fmt.Sprintf("stream-event-%d", rand.IntN(100)+1)
	var wfrId string

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events(m.eventName),
			Name:        workflowName,
			Description: "This is part of the monitoring system for testing the readiness of this Hatchet installation.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {

					input := &probeEvent{}

					err = ctx.WorkflowInput(input)
					if err != nil {
						return nil, err
					}

					wfrId = ctx.WorkflowRunId()

					if input.UniqueStreamId == "" {
						return nil, fmt.Errorf("uniqueStreamId is required")
					}

					if input.UniqueStreamId != streamValue {
						return nil, fmt.Errorf("uniqueStreamId does not match stream value")
					}

					ctx.StreamEvent([]byte("This is a stream event for step-one"))

					stepChan <- "step-one"

					return &stepOneOutput{
						Message:        "This is a message for step-one",
						UniqueStreamId: streamValue,
					}, nil
				},
				).SetName("step-one"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &probeEvent{}
					err = ctx.StepOutput("step-one", input)

					if err != nil {
						return nil, err
					}

					if input.UniqueStreamId == "" {
						return nil, fmt.Errorf("uniqueStreamId is required")
					}

					if input.UniqueStreamId != streamValue {
						return nil, fmt.Errorf("uniqueStreamId does not match stream value")
					}

					ctx.StreamEvent([]byte("This is a stream event for step-two"))
					stepChan <- "step-two"

					return &stepOneOutput{
						Message:        "This is a message for step-two",
						UniqueStreamId: streamValue,
					}, nil
				}).SetName("step-two").AddParents("step-one"),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		err = c.Subscribe().StreamByAdditionalMetadata(ctx, streamKey, streamValue, func(event client.StreamEvent) error {
			events <- string(event.Message)

			return nil
		})
		if err != nil {
			errors <- fmt.Errorf("error subscribing to stream: %w", err)
		}
	}()

	go func() {
		testEvent := probeEvent{
			UniqueStreamId: streamValue,
		}

		err := c.Event().Push(
			ctx,
			m.eventName,
			testEvent,
			nil,
			nil,
			client.WithEventMetadata(map[string]string{
				streamKey: streamValue,
			},
			),
		)

		if err != nil {
			errors <- fmt.Errorf("error pushing event: %w", err)
		}
	}()

	cleanupWorker, err := w.Start()

	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
	}

	cleanup := func() {
		err := cleanupWorker()
		if err != nil {
			m.l.Error().Msgf("error cleaning up worker: %s", err)
		}
		defer func() {
			if wfrId == "" {
				m.l.Warn().Msg("workflow run id was never set for probe")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			defer cancel()

			for i := 0; i < 10; i++ {
				wrfRow, err := m.config.APIRepository.WorkflowRun().GetWorkflowRunById(ctx, cf.TenantId, wfrId)

				if err != nil {
					m.l.Error().Msgf("error getting workflow run: %s", err)

				}

				if wrfRow.Status != dbsqlc.WorkflowRunStatusRUNNING {

					workflowId := sqlchelpers.UUIDToStr(wrfRow.Workflow.ID)

					_, err = m.config.APIRepository.Workflow().DeleteWorkflow(ctx, cf.TenantId, workflowId)

					if err != nil {
						m.l.Error().Msgf("error deleting workflow: %s", err)
					} else {
						m.l.Info().Msgf("deleted workflow %s", workflowId)
						return
					}

				}

				time.Sleep(200 * time.Millisecond)

			}

			m.l.Error().Msg("could not clean up workflow after 10 attempts")

		}()
	}

	return cleanup, nil
}

func getBearerTokenFromRequest(r *http.Request) (string, error) {
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer")

	if len(splitToken) != 2 {
		return "", fmt.Errorf("invalid token")
	}

	reqToken = strings.TrimSpace(splitToken[1])

	return reqToken, nil
}

func randomNamespace() string {
	return "ns_" + uuid.New().String()[0:8]
}
