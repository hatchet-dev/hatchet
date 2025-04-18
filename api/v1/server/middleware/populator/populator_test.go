package populator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type oneToManyResource struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
}

func oneToManyResourceGetter(config *server.ServerConfig, parentId, id string) (interface{}, string, error) {
	if parentId == "" {
		parentId := uuid.NewString()

		return &oneToManyResource{
			ID:       id,
			ParentID: parentId,
		}, parentId, nil
	}

	return &oneToManyResource{
		ID:       id,
		ParentID: parentId,
	}, parentId, nil
}

type topLevelResource struct {
	ID string `json:"id"`
}

func topLevelResourceGetter(config *server.ServerConfig, parentId, id string) (interface{}, string, error) {
	return &topLevelResource{
		ID: id,
	}, "", nil
}

// CustomResourceHandler is a mock implementation that tracks the sequence of calls
// and populates resources with mock objects
type CustomResourceHandler struct {
	populatedResources []string
	resources          map[string]interface{}
	parentIDs          map[string]string
}

// NewCustomResourceHandler creates a new handler for testing resource chains
func NewCustomResourceHandler() *CustomResourceHandler {
	return &CustomResourceHandler{
		populatedResources: make([]string, 0),
		resources:          make(map[string]interface{}),
		parentIDs:          make(map[string]string),
	}
}

// Handler returns a function to use for populating resources during testing
func (h *CustomResourceHandler) Handler() func(echo.Context, *resource) error {
	return func(ctx echo.Context, node *resource) error {
		h.populatedResources = append(h.populatedResources, node.ResourceKey)
		h.parentIDs[node.ResourceKey] = node.ParentID

		// Set the node resource if we have a mock for it
		if res, ok := h.resources[node.ResourceKey]; ok {
			node.Resource = res
		}

		return nil
	}
}

// SetResource adds a mock resource to be used during testing
func (h *CustomResourceHandler) SetResource(key string, resource interface{}) {
	h.resources[key] = resource
}

// TestPopulatorResourceChain tests the chain-building and traversal functionality
// of the populator, focusing on the parent-child relationship between resources
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

	// Create server config
	config := &server.ServerConfig{}

	// Create mock resources
	mockTenant := &struct{ ID string }{ID: "tenant-123"}
	mockWorkflow := &struct{ ID, TenantID string }{ID: "workflow-456", TenantID: "tenant-123"}
	mockRun := &struct{ ID, WorkflowID string }{ID: "run-789", WorkflowID: "workflow-456"}

	// Create handler and set up mock resources
	handler := NewCustomResourceHandler()
	handler.SetResource("tenant", mockTenant)
	handler.SetResource("workflow", mockWorkflow)
	handler.SetResource("workflow-run", mockRun)

	// Create a test populator with a customized populateResource function
	testPopulator := &testPopulator{
		Populator: Populator{
			config: config,
		},
		resourceHandler: handler.Handler(),
	}

	// Call the populate method to test the chain building
	err := testPopulator.populate(c, routeInfo)

	// Assertions
	assert.NoError(t, err)

	// Verify resources were populated in the correct order
	assert.Equal(t, []string{"tenant", "workflow", "workflow-run"}, handler.populatedResources)

	// Verify parent IDs were set correctly
	assert.Equal(t, "", handler.parentIDs["tenant"], "Tenant should have no parent")
	assert.Equal(t, "tenant-123", handler.parentIDs["workflow"], "Workflow should have tenant as parent")
	assert.Equal(t, "workflow-456", handler.parentIDs["workflow-run"], "Run should have workflow as parent")

	// Verify resources were set in the context
	assert.Equal(t, mockTenant, c.Get("tenant"))
	assert.Equal(t, mockWorkflow, c.Get("workflow"))
	assert.Equal(t, mockRun, c.Get("workflow-run"))
}

// testPopulator is a special version of Populator for testing
type testPopulator struct {
	Populator
	resourceHandler func(echo.Context, *resource) error
}

// Override the traverseNode and populateResource methods for testing
func (p *testPopulator) traverseNode(c echo.Context, node *resource) error {
	populated := false

	// Populate current node if ID is available
	if node.ResourceID != "" {
		err := p.populateResource(c, node)
		if err != nil {
			return err
		}
		populated = true
	}

	// Process child nodes
	if node.Children != nil {
		for _, child := range node.Children {
			if populated {
				child.ParentID = node.ResourceID
			}

			err := p.traverseNode(c, child)
			if err != nil {
				return err
			}

			if !populated && child.ParentID != "" {
				// Use parent info to populate current resource
				err = p.populateResource(c, node)
				if err != nil {
					return err
				}
				populated = true
			}
		}
	}

	// Ensure resource was populated
	if !populated {
		err := p.populateResource(c, node)
		if err != nil {
			return err
		}
	}

	return nil
}

// populateResource is the customized version for testing
func (p *testPopulator) populateResource(c echo.Context, node *resource) error {
	if p.resourceHandler != nil {
		return p.resourceHandler(c, node)
	}
	return nil
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
