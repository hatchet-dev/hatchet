package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/rand"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func (m *MonitoringService) MonitoringPostRunProbe(ctx echo.Context, request gen.MonitoringPostRunProbeRequestObject) (gen.MonitoringPostRunProbeResponseObject, error) {
	if !m.enabled {
		return nil, fmt.Errorf("monitoring is not enabled")
	}

	tenant := request.Tenant

	if tenant == uuid.Nil {
		return nil, fmt.Errorf("tenant is required")
	}

	if !slices.Contains[[]string](m.permittedTenants, tenant.String()) {

		err := fmt.Errorf("tenant is not a monitoring tenant for this instance")

		if len(m.permittedTenants) > 0 {
			m.l.Error().Err(err).Msgf("monitoring tenants are %v", m.permittedTenants)
		} else {
			m.l.Error().Err(err).Msg("no monitoring tenants are configured")
		}

		return gen.MonitoringPostRunProbe403JSONResponse{}, err
	}

	token, err := getBearerTokenFromRequest(ctx.Request())

	if err != nil {
		return nil, err
	}

	os.Setenv("HATCHET_CLIENT_TLS_ROOT_CA_FILE", os.Getenv("SERVER_TLS_ROOT_CA_FILE"))
	os.Setenv("HATCHET_CLIENT_TOKEN", token)
	os.Setenv("HATCHET_CLIENT_NAMESPACE", generateRandomNamespace(tenant))

	events := make(chan string, 50)
	errorChan := make(chan error, 50)
	interrupt := cmdutils.InterruptChan()

	cancellableContext, cancel := context.WithTimeout(ctx.Request().Context(), m.probeTimeout)

	defer cancel()
	var cleanup func() error

	go func() {
		cleanup, err = m.run(cancellableContext, events, errorChan)
		if err != nil {
			errorChan <- err
		}

		// check if the context was cancelled in the meantime

		if cancellableContext.Err() != nil {
			cleanupErr := cleanup()
			if cleanupErr != nil {
				m.l.Error().Msgf("error cleaning up probe worker: %s", cleanupErr)
			}
		}
	}()

	defer func() {
		var cleanupErr error
		if cleanup != nil {
			cleanupErr = cleanup()
		} else {
			m.l.Warn().Msg("cleanup function was never set for probe")
		}
		if cleanupErr != nil {
			m.l.Error().Msgf("error cleaning up probe worker: %s", err)
		}
	}()

	messages := []string{"This is a stream event for step-one", "This is a stream event for step-two"}
	messageIndex := 0

	for {

		select {
		case e := <-events:
			if e != messages[messageIndex] {
				return nil, fmt.Errorf("expected message %s, got %s", messages[messageIndex], e)
			}

			messageIndex++
			if messageIndex == len(messages) {
				m.l.Debug().Msg("probe completed successfully")
				return nil, nil
			}

		case <-cancellableContext.Done():

			if cancellableContext.Err() == context.DeadlineExceeded {
				m.l.Error().Msg("timed out waiting for probe to complete")
				return nil, fmt.Errorf("timed out waiting for probe to complete")
			} else {
				m.l.Debug().Msg("probe completed successfully")
				return nil, nil
			}

		case err := <-errorChan:
			m.l.Error().Msgf("error during probe: %s", err)
			return nil, err
		case <-interrupt:
			return nil, fmt.Errorf("interrupted during probe")
		}

	}

}

type probeEvent struct {
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func (m *MonitoringService) run(ctx context.Context, events chan<- string, errors chan<- error) (func() error, error) {
	c, err := client.New()

	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}
	streamKey := "streamKey"
	streamValue := fmt.Sprintf("stream-event-%d", rand.Intn(100)+1)

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events(m.eventName),
			Name:        "probe",
			Description: "This is part of the monitoring system for testing the readiness of this Hatchet installation.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &probeEvent{}

					err = ctx.WorkflowInput(input)

					if err != nil {
						return nil, err
					}
					ctx.StreamEvent([]byte("This is a stream event for step-one"))

					return &stepOneOutput{}, nil
				},
				).SetName("step-one"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &stepOneOutput{}
					err = ctx.StepOutput("step-one", input)

					if err != nil {
						return nil, err
					}
					ctx.StreamEvent([]byte("This is a stream event for step-two"))

					return &stepOneOutput{}, nil
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
		testEvent := probeEvent{}

		err := c.Event().Push(
			context.Background(),
			m.eventName,
			testEvent,
			client.WithEventMetadata(map[string]string{
				streamKey: streamValue,
			},
			),
		)

		if err != nil {
			errors <- fmt.Errorf("error pushing event: %w", err)
		}
	}()

	cleanup, err := w.Start()

	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
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

func generateRandomNamespace(tenant uuid.UUID) string {
	return fmt.Sprintf("monitoring-%s-%s", tenant.String(), uuid.New().String()[0:8])
}
