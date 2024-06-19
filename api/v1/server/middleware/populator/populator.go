package populator

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type ResourceGetterFunc func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error)

type Populator struct {
	// getters is a map of resource keys to getter methods
	getters map[string]ResourceGetterFunc

	config *server.ServerConfig
}

func NewPopulator(config *server.ServerConfig) *Populator {
	return &Populator{
		getters: make(map[string]ResourceGetterFunc),
		config:  config,
	}
}

func (p *Populator) RegisterGetter(resourceKey string, getter ResourceGetterFunc) {
	p.getters[resourceKey] = getter
}

func (p *Populator) Middleware(r *middleware.RouteInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := p.populate(c, r)
		if err != nil {
			return err
		}

		return nil
	}
}

func (p *Populator) populate(c echo.Context, r *middleware.RouteInfo) error {
	if len(r.Resources) == 0 {
		return nil
	}

	// Create a map of keys to identifiers in the form of strings
	keysToIds := make(map[string]string)

	// Add params to the map
	for _, paramName := range c.ParamNames() {
		keysToIds[paramName] = c.Param(paramName)
	}

	rootResource := &resource{}
	currResource := rootResource
	var prevResource *resource

	// loop through the requested resources, adding params as ids
	for _, resourceKey := range r.Resources {
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

	err := p.traverseNode(c, rootResource)

	if err != nil {
		return err
	}

	// loop through the resources again and add them to the context
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

	return nil
}

type resource struct {
	ResourceKey string
	ResourceID  string
	Resource    interface{}
	ParentID    string
	Children    []*resource
}

func (p *Populator) traverseNode(c echo.Context, node *resource) error {
	populated := false

	// determine if we have a resource locator to populate the node
	if node.ResourceID != "" {
		err := p.callGetter(node, node.ParentID, node.ResourceID)

		if err != nil {
			return fmt.Errorf("could not populate resource %s: %w", node.ResourceKey, err)
		}

		populated = true
	}

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
				// use the parent locator to populate the resource
				err = p.callGetter(node, node.ParentID, child.ParentID)

				if err != nil {
					return err
				}

				populated = true
			} else if populated && child.ParentID != node.ResourceID {
				// if the parent ID is not the same as the resource ID, throw an error
				return fmt.Errorf("resource %s could not be populated", node.ResourceKey)
			}
		}
	}

	// if parent is not populated at this point, throw an error
	if !populated {
		return fmt.Errorf("resource %s could not be populated", node.ResourceKey)
	}

	// if the resource is not nil, add to the context
	if node.Resource != nil {
		c.Set(node.ResourceKey, node.Resource)
	}

	return nil
}

func (p *Populator) callGetter(node *resource, parentId, id string) error {
	if _, exists := p.getters[node.ResourceKey]; !exists {
		return nil
	}

	res, uniqueParentId, err := p.getters[node.ResourceKey](p.config, parentId, id)
	if err != nil {
		return err
	}

	node.Resource = res
	node.ParentID = uniqueParentId

	return nil
}
