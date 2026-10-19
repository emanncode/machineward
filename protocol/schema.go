package protocol

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
)

//go:embed v1/schema.json
var schemaFiles embed.FS

var (
	resolvedSchemaOnce sync.Once
	resolvedSchema     *jsonschema.Resolved
	resolvedSchemaErr  error
)

func validateSchema(data []byte) error {
	resolvedSchemaOnce.Do(func() {
		schemaJSON, err := schemaFiles.ReadFile("v1/schema.json")
		if err != nil {
			resolvedSchemaErr = fmt.Errorf("read protocol schema: %w", err)
			return
		}

		var schema jsonschema.Schema
		if err := json.Unmarshal(schemaJSON, &schema); err != nil {
			resolvedSchemaErr = fmt.Errorf("parse protocol schema: %w", err)
			return
		}

		resolvedSchema, resolvedSchemaErr = schema.Resolve(nil)
		if resolvedSchemaErr != nil {
			resolvedSchemaErr = fmt.Errorf("compile protocol schema: %w", resolvedSchemaErr)
		}
	})
	if resolvedSchemaErr != nil {
		return resolvedSchemaErr
	}

	var instance any
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("parse message JSON: %w", err)
	}
	if err := resolvedSchema.Validate(instance); err != nil {
		return err
	}
	return nil
}
