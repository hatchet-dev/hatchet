# HatchetSdkRest::CronWorkflows

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** |  |  |
| **workflow_version_id** | **String** |  |  |
| **workflow_id** | **String** |  |  |
| **workflow_name** | **String** |  |  |
| **cron** | **String** |  |  |
| **enabled** | **Boolean** |  |  |
| **method** | [**CronWorkflowsMethod**](CronWorkflowsMethod.md) |  |  |
| **name** | **String** |  | [optional] |
| **input** | **Hash&lt;String, Object&gt;** |  | [optional] |
| **additional_metadata** | **Hash&lt;String, Object&gt;** |  | [optional] |
| **priority** | **Integer** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::CronWorkflows.new(
  metadata: null,
  tenant_id: null,
  workflow_version_id: null,
  workflow_id: null,
  workflow_name: null,
  cron: null,
  enabled: null,
  method: null,
  name: null,
  input: null,
  additional_metadata: null,
  priority: null
)
```

