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
