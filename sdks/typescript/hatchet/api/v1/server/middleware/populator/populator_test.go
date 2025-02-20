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

func TestPopulatorMiddleware(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Mock RouteInfo
	routeInfo := &middleware.RouteInfo{
		Resources: []string{"resource1", "resource2"},
	}

	resource2Id := uuid.New().String()

	// Setting params for the context
	c.SetParamNames("resource2")
	c.SetParamValues(resource2Id)

	// Creating Populator with mock getter function
	populator := NewPopulator(&server.ServerConfig{})

	populator.RegisterGetter("resource1", topLevelResourceGetter)
	populator.RegisterGetter("resource2", oneToManyResourceGetter)

	// Using the Populator middleware
	middlewareFunc := populator.Middleware(routeInfo)
	err := middlewareFunc(c)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, c.Get("resource1"))
	assert.NotNil(t, c.Get("resource2"))
}
