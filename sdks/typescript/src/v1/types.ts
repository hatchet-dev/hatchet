// Define proper JSON value types
export type JsonPrimitive = string | number | boolean | null;
export type JsonArray = JsonValue[];
export type JsonValue = JsonPrimitive | JsonObject | JsonArray;
export type JsonObject = { [Key in string]: JsonValue } & {
  [Key in string]?: JsonValue | undefined;
};

// Input and output types
export type InputType = JsonObject;
export type UnknownInputType = {};
export type OutputType = JsonObject | void;

// Helper type to check if a type is a valid workflow output structure
type IsValidWorkflowOutput<T> = T extends Record<string, JsonObject> ? true : false;

// Improved WorkflowOutputType with helpful error message
export type WorkflowOutputType<T = any> =
  IsValidWorkflowOutput<T> extends true
    ? T
    : (Record<string, JsonObject> | void) & {
        // This will only appear in error messages
        [ERROR_WORKFLOW_OUTPUT]?: 'Workflow output must be shaped as Record<"task-name", JsonObject>. Each property must be an object, not a primitive value.';
      };

// Symbol used for the error message
declare const ERROR_WORKFLOW_OUTPUT: unique symbol;
