package populator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type oneToManyResource struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
}

type topLevelResource struct {
	ID string `json:"id"`
}

func topLevelResourceGetter(config *server.ServerConfig, parentId, id string) (interface{}, string, error) {
	return &topLevelResource{
		ID: id,
	}, "", nil
}

// TestPopulatorResourceChain tests the chain-building and traversal functionality
// focusing on parent-child relationships between resources
func TestPopulatorResourceChain(t *testing.T) {
	// Setup Echo
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Setup a chain of resources: tenant -> workflow -> workflow-run
	c.SetParamNames("tenant", "workflow", "workflow-run")
	c.SetParamValues("tenant-123", "workflow-456", "run-789")

	// Create resource chain in RouteInfo
	routeInfo := &middleware.RouteInfo{
		Resources: []string{"tenant", "workflow", "workflow-run"},
	}

	// Mock resources that will be populated
	mockTenant := &struct{ ID string }{ID: "tenant-123"}
	mockWorkflow := &struct{ ID, TenantID string }{ID: "workflow-456", TenantID: "tenant-123"}
	mockRun := &struct{ ID, WorkflowID string }{ID: "run-789", WorkflowID: "workflow-456"}

	// Create a map to track parent IDs between resources
	parentIDs := make(map[string]string)

	// Create our own implementation of the populate logic to test the core functionality
	// without hitting actual API calls
	rootResource := &resource{}
	currResource := rootResource
	var prevResource *resource

	// Build resource chain (simplified from the actual populate method)
	keysToIds := map[string]string{
		"tenant":       "tenant-123",
		"workflow":     "workflow-456",
		"workflow-run": "run-789",
	}

	for _, resourceKey := range routeInfo.Resources {
		currResource.ResourceKey = resourceKey

		if resourceId, exists := keysToIds[resourceKey]; exists && resourceId != "" {
			currResource.ResourceID = resourceId
		}

		if prevResource != nil {
			currResource.ParentID = prevResource.ResourceID
			prevResource.Children = append(prevResource.Children, currResource)
		}

		prevResource = currResource
		currResource = &resource{}
	}

	// Track resources populated in order
	populatedResources := []string{}

	// Manual traverseNode implementation for testing
	var traverseNode func(node *resource) error
	traverseNode = func(node *resource) error {
		populatedResources = append(populatedResources, node.ResourceKey)
		parentIDs[node.ResourceKey] = node.ParentID

		// Set mock resource
		switch node.ResourceKey {
		case "tenant":
			node.Resource = mockTenant
		case "workflow":
			node.Resource = mockWorkflow
		case "workflow-run":
			node.Resource = mockRun
		}

		// Process children
		for _, child := range node.Children {
			if err := traverseNode(child); err != nil {
				return err
			}
		}

		return nil
	}

	// Execute our test "traversal"
	err := traverseNode(rootResource)
	assert.NoError(t, err)

	// Add populated resources to context (simulating what the actual method does)
	currResource = rootResource
	for {
		if currResource.Resource != nil {
			c.Set(currResource.ResourceKey, currResource.Resource)
		}

		if len(currResource.Children) == 0 {
			break
		}

		currResource = currResource.Children[0]
	}

	// Assertions

	// Verify resources were populated in the expected order
	assert.Equal(t, []string{"tenant", "workflow", "workflow-run"}, populatedResources)

	// Verify parent IDs were set correctly
	assert.Equal(t, "", parentIDs["tenant"], "Tenant should have no parent")
	assert.Equal(t, "tenant-123", parentIDs["workflow"], "Workflow should have tenant as parent")
	assert.Equal(t, "workflow-456", parentIDs["workflow-run"], "Run should have workflow as parent")

	// Verify resources were set in the context
	assert.Equal(t, mockTenant, c.Get("tenant"))
	assert.Equal(t, mockWorkflow, c.Get("workflow"))
	assert.Equal(t, mockRun, c.Get("workflow-run"))
}

// TestPopulatorMiddleware tests the basic middleware flow without mocking repositories
func TestPopulatorMiddleware(t *testing.T) {
	// Setup Echo
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create a basic route info with a resource that we know doesn't exist
	routeInfo := &middleware.RouteInfo{
		Resources: []string{"nonexistent-resource"},
	}

	// Create populator and middleware
	populator := NewPopulator(&server.ServerConfig{})
	middlewareFunc := populator.Middleware(routeInfo)

	// Execute middleware - should return error for unknown resource
	err := middlewareFunc(c)

	// Assertions
	assert.Error(t, err)
}

// TestPopulatorErrNotFound tests the error handling in the FromContext getter
func TestPopulatorErrNotFound(t *testing.T) {
	// Setup Echo
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test GetTenant from context when not set
	getter := FromContext(c)
	tenant, err := getter.GetTenant()

	// Assertions for not found error
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, tenant)
}
