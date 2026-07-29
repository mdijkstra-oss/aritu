package lint

import (
	"encoding/json"
	"fmt"
)

const NamesSchema = `{"type":"object","properties":{"names":{"type":"array","items":{"type":"string"}}},"required":["names"],"additionalProperties":false}`

func VerdictSchemaFor(units []Unit) json.RawMessage {
	answers := make(map[string]schemaNode, len(units))
	keys := make([]string, 0, len(units))
	for _, unit := range units {
		answers[unit.Key] = answerSchema()
		keys = append(keys, unit.Key)
	}
	encoded, err := json.Marshal(closedObject(answers, keys))
	if err != nil {
		panic(fmt.Sprintf("the verdict schema failed to marshal, which its types make impossible: %v", err))
	}
	return encoded
}

// schemaNode is one node of a generated JSON Schema. AdditionalProperties is a
// pointer because the keyword only means anything on an object: emitted beside a
// string or a boolean the endpoint rejects the request outright, and a rejected
// request is not retried — every call carrying the schema fails the same way, for
// a reason that reads as the endpoint's rather than as ours.
type schemaNode struct {
	Type                 string                `json:"type"`
	Properties           map[string]schemaNode `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
}

func closedObject(properties map[string]schemaNode, required []string) schemaNode {
	isClosed := false
	return schemaNode{
		Type:                 "object",
		Properties:           properties,
		Required:             required,
		AdditionalProperties: &isClosed,
	}
}

func answerSchema() schemaNode {
	return closedObject(map[string]schemaNode{
		"satisfies": {Type: "boolean"},
		"reason":    {Type: "string"},
	}, []string{"satisfies", "reason"})
}
