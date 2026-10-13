//go:build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const workflowDurationMetric = "hatchet_tenant_workflow_duration_milliseconds"

type runCountKey struct {
	workflowName string
	status       string
}

// TestPrometheusRunCountsMatchAPI checks that the workflow duration histogram records exactly one
// observation per finished run, with the same final status the API reports, across completed,
// retried, failed, cancelled and DAG runs.
func TestPrometheusRunCountsMatchAPI(t *testing.T) {
	metricsURL := os.Getenv("HATCHET_E2E_PROMETHEUS_URL")
	if metricsURL == "" {
		t.Skip("HATCHET_E2E_PROMETHEUS_URL is not set")
	}

	ctx := newTestContext(t)
	suffix := uuid.New().String()[:8]
	testRunID := uniqueID()

	succeed := sharedClient.NewStandaloneTask("metrics-succeed-"+suffix,
		func(ctx hatchet.Context, input EmptyInput) (map[string]string, error) {
			return map[string]string{"result": "ok"}, nil
		},
	)

	retryThenSucceed := sharedClient.NewStandaloneTask("metrics-retry-succeed-"+suffix,
		func(ctx hatchet.Context, input EmptyInput) (map[string]string, error) {
			if ctx.RetryCount() == 0 {
				return nil, errors.New("first attempt fails")
			}
			return map[string]string{"result": "ok"}, nil
		},
		hatchet.WithRetries(1),
	)

	fail := sharedClient.NewStandaloneTask("metrics-fail-"+suffix,
		func(ctx hatchet.Context, input EmptyInput) (map[string]string, error) {
			return nil, errors.New("every attempt fails")
		},
		hatchet.WithRetries(1),
	)

	cancel := sharedClient.NewStandaloneTask("metrics-cancel-"+suffix,
		func(ctx hatchet.Context, input EmptyInput) (map[string]string, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Minute):
				return map[string]string{"result": "not cancelled"}, nil
			}
		},
	)

	dag := sharedClient.NewWorkflow("metrics-dag-" + suffix)
	dagParent := dag.NewTask("parent", func(ctx hatchet.Context, input EmptyInput) (map[string]string, error) {
		return map[string]string{"result": "parent"}, nil
	})
	dag.NewTask("child", func(ctx hatchet.Context, input EmptyInput) (map[string]string, error) {
		return map[string]string{"result": "child"}, nil
	}, hatchet.WithParents(dagParent))

	worker, err := sharedClient.NewWorker("e2e-metrics-worker-"+suffix,
		hatchet.WithWorkflows(succeed, retryThenSucceed, fail, cancel, dag),
	)
	require.NoError(t, err)

	stopWorker, err := worker.Start()
	require.NoError(t, err)
	t.Cleanup(func() { _ = stopWorker() })

	pollUntil(t, ctx, func() (bool, error) {
		for _, name := range []string{succeed.GetName(), retryThenSucceed.GetName(), fail.GetName(), cancel.GetName(), dag.GetName()} {
			if _, err := sharedClient.Workflows().Get(ctx, name); err != nil {
				return false, nil
			}
		}
		return true, nil
	})

	metadata := hatchet.WithRunMetadata(map[string]string{"test_run_id": testRunID})
	expected := map[runCountKey]int{}
	var runIDs []string
	var cancelRunIDs []string

	trigger := func(name string, n int, status rest.V1TaskStatus, run func() (*hatchet.WorkflowRunRef, error)) {
		for range n {
			ref, err := run()
			require.NoError(t, err)

			runIDs = append(runIDs, ref.RunId)

			if status == rest.V1TaskStatusCANCELLED {
				cancelRunIDs = append(cancelRunIDs, ref.RunId)
			}
		}

		expected[runCountKey{workflowName: name, status: string(status)}] += n
	}

	trigger(succeed.GetName(), 5, rest.V1TaskStatusCOMPLETED, func() (*hatchet.WorkflowRunRef, error) {
		return succeed.RunNoWait(ctx, EmptyInput{}, metadata)
	})
	trigger(retryThenSucceed.GetName(), 3, rest.V1TaskStatusCOMPLETED, func() (*hatchet.WorkflowRunRef, error) {
		return retryThenSucceed.RunNoWait(ctx, EmptyInput{}, metadata)
	})
	trigger(fail.GetName(), 3, rest.V1TaskStatusFAILED, func() (*hatchet.WorkflowRunRef, error) {
		return fail.RunNoWait(ctx, EmptyInput{}, metadata)
	})
	trigger(cancel.GetName(), 2, rest.V1TaskStatusCANCELLED, func() (*hatchet.WorkflowRunRef, error) {
		return cancel.RunNoWait(ctx, EmptyInput{}, metadata)
	})
	trigger(dag.GetName(), 2, rest.V1TaskStatusCOMPLETED, func() (*hatchet.WorkflowRunRef, error) {
		return dag.RunNoWait(ctx, EmptyInput{}, metadata)
	})

	for _, runID := range cancelRunIDs {
		pollUntil(t, ctx, func() (bool, error) {
			status, err := sharedClient.Runs().GetStatus(ctx, runID)
			if err != nil {
				return false, err
			}
			return *status == rest.V1TaskStatusRUNNING, nil
		})

		externalIDs := []uuid.UUID{uuid.MustParse(runID)}
		_, err := sharedClient.Runs().Cancel(ctx, rest.V1CancelTaskRequest{ExternalIds: &externalIDs})
		require.NoError(t, err)
	}

	for _, runID := range runIDs {
		pollUntil(t, ctx, func() (bool, error) {
			status, err := sharedClient.Runs().GetStatus(ctx, runID)
			if err != nil {
				return false, err
			}
			switch *status {
			case rest.V1TaskStatusCOMPLETED, rest.V1TaskStatusFAILED, rest.V1TaskStatusCANCELLED:
				return true, nil
			}
			return false, nil
		})
	}

	limit := int64(100)
	metadataFilter := []string{"test_run_id:" + testRunID}
	runs, err := sharedClient.Runs().List(ctx, rest.V1WorkflowRunListParams{
		Since:              time.Now().Add(-10 * time.Minute),
		Limit:              &limit,
		AdditionalMetadata: &metadataFilter,
	})
	require.NoError(t, err)

	apiCounts := map[runCountKey]int{}
	for _, run := range runs.Rows {
		require.NotNil(t, run.WorkflowName)
		apiCounts[runCountKey{workflowName: *run.WorkflowName, status: string(run.Status)}]++
	}
	require.Equal(t, expected, apiCounts, "API run counts")

	workflowNames := map[string]struct{}{}
	for k := range expected {
		workflowNames[k.workflowName] = struct{}{}
	}

	// observations are recorded asynchronously after the status update, so wait for them to land
	var metricCounts map[runCountKey]int
	deadline := time.Now().Add(30 * time.Second)
	for {
		metricCounts, err = scrapeWorkflowRunCounts(ctx, metricsURL, workflowNames)
		require.NoError(t, err)

		if maps.Equal(metricCounts, expected) || time.Now().After(deadline) {
			break
		}

		time.Sleep(pollInterval)
	}
	require.Equal(t, expected, metricCounts, "prometheus run counts")

	// a late duplicate observation would push a count past the API's
	time.Sleep(3 * time.Second)
	metricCounts, err = scrapeWorkflowRunCounts(ctx, metricsURL, workflowNames)
	require.NoError(t, err)
	require.Equal(t, expected, metricCounts, "prometheus run counts after settling")
}

func scrapeWorkflowRunCounts(ctx context.Context, metricsURL string, workflowNames map[string]struct{}) (map[runCountKey]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraping %s: status %d", metricsURL, resp.StatusCode)
	}

	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}

	counts := map[runCountKey]int{}

	family, ok := families[workflowDurationMetric]
	if !ok {
		return counts, nil
	}

	for _, m := range family.GetMetric() {
		var key runCountKey
		for _, label := range m.GetLabel() {
			switch label.GetName() {
			case "workflow_name":
				key.workflowName = label.GetValue()
			case "status":
				key.status = label.GetValue()
			}
		}

		if _, ok := workflowNames[key.workflowName]; !ok {
			continue
		}

		if c := int(m.GetHistogram().GetSampleCount()); c > 0 {
			counts[key] += c
		}
	}

	return counts, nil
}
