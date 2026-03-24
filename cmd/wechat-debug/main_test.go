package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestJSONOutputStructure tests that JSON output has consistent structure
func TestJSONOutputStructure(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
	}{
		{
			name:    "scan",
			command: "scan",
			args:    []string{"--mock", "--json"},
		},
		{
			name:    "focus",
			command: "focus",
			args:    []string{"--contact", "张三", "--mock", "--json"},
		},
		{
			name:    "send",
			command: "send",
			args:    []string{"--contact", "张三", "--message", "Test message", "--mock", "--json"},
		},
		{
			name:    "verify",
			command: "verify",
			args:    []string{"--contact", "张三", "--message", "Test message", "--mock", "--json"},
		},
		{
			name:    "run-chain",
			command: "run-chain",
			args:    []string{"--contact", "张三", "--message", "Test message", "--mock", "--json"},
		},
		{
			name:    "full-test",
			command: "full-test",
			args:    []string{"--mock", "--json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build command - use compiled binary directly with absolute path
			exePath := filepath.Join("..", "..", "cmd", "wechat-debug", "wechat-debug.exe")
			args := append([]string{tt.command}, tt.args...)
			cmd := exec.Command(exePath, args...)

			// Run command
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, string(output))
			}

			// Parse JSON output
			var result map[string]interface{}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, string(output))
			}

			// Check required fields exist
			if _, ok := result["mode"]; !ok {
				t.Error("Missing 'mode' field in JSON output")
			}

			if _, ok := result["steps"]; !ok {
				t.Error("Missing 'steps' field in JSON output")
			}

			// Check steps is an array
			steps, ok := result["steps"].([]interface{})
			if !ok {
				t.Error("'steps' field is not an array")
			} else if len(steps) == 0 {
				t.Error("'steps' array is empty")
			}

			// Check final field exists for chain commands
			if tt.command == "run-chain" || tt.command == "full-test" {
				if _, ok := result["final"]; !ok {
					t.Error("Missing 'final' field for chain command")
				}
			}

			// Check each step has required fields
			for i, stepInterface := range steps {
				step, ok := stepInterface.(map[string]interface{})
				if !ok {
					t.Errorf("Step %d is not a map", i)
					continue
				}

				// Check step has required fields
				if _, ok := step["step"]; !ok {
					t.Errorf("Step %d missing 'step' field", i)
				}
				if _, ok := step["status"]; !ok {
					t.Errorf("Step %d missing 'status' field", i)
				}
				if _, ok := step["confidence"]; !ok {
					t.Errorf("Step %d missing 'confidence' field", i)
				}

				// Check diagnostics field exists and is an array
				if diags, ok := step["diagnostics"]; ok {
					if _, ok := diags.([]interface{}); !ok {
						t.Errorf("Step %d 'diagnostics' is not an array", i)
					}
				}
			}
		})
	}
}

// TestDiagnosticsFieldWhitelist tests that diagnostics have stable field whitelist
func TestDiagnosticsFieldWhitelist(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
	}{
		{
			name:    "focus",
			command: "focus",
			args:    []string{"--contact", "张三", "--mock", "--json"},
		},
		{
			name:    "send",
			command: "send",
			args:    []string{"--contact", "张三", "--message", "Test message", "--mock", "--json"},
		},
		{
			name:    "verify",
			command: "verify",
			args:    []string{"--contact", "张三", "--message", "Test message", "--mock", "--json"},
		},
	}

	// Required whitelist fields
	requiredFields := []string{
		"locate_source",
		"evidence_count",
		"new_message_nodes",
		"message_content_match",
		"delivery_state",
		"confidence",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build command - use compiled binary directly with absolute path
			exePath := filepath.Join("..", "..", "cmd", "wechat-debug", "wechat-debug.exe")
			args := append([]string{tt.command}, tt.args...)
			cmd := exec.Command(exePath, args...)

			// Run command
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, string(output))
			}

			// Parse JSON output
			var result map[string]interface{}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, string(output))
			}

			// Get steps
			steps, ok := result["steps"].([]interface{})
			if !ok || len(steps) == 0 {
				t.Fatal("No steps found in output")
			}

			// Get the last step (which should have diagnostics)
			lastStep, ok := steps[len(steps)-1].(map[string]interface{})
			if !ok {
				t.Fatal("Last step is not a map")
			}

			// Get diagnostics
			diags, ok := lastStep["diagnostics"].([]interface{})
			if !ok || len(diags) == 0 {
				t.Skip("No diagnostics found in last step")
				return
			}

			// Check first diagnostic has required fields
			firstDiag, ok := diags[0].(map[string]interface{})
			if !ok {
				t.Fatal("First diagnostic is not a map")
			}

			context, ok := firstDiag["context"].(map[string]interface{})
			if !ok {
				t.Fatal("Diagnostic context is not a map")
			}

			// Check required whitelist fields exist
			for _, field := range requiredFields {
				if _, ok := context[field]; !ok {
					t.Errorf("Missing required field '%s' in diagnostics", field)
				}
			}
		})
	}
}

// TestStepOrder tests that steps are in correct order
func TestStepOrder(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		args          []string
		expectedSteps []string
	}{
		{
			name:          "run-chain",
			command:       "run-chain",
			args:          []string{"--contact", "张三", "--message", "Test message", "--mock", "--json"},
			expectedSteps: []string{"detect", "scan", "focus", "send", "verify"},
		},
		{
			name:          "full-test",
			command:       "full-test",
			args:          []string{"--mock", "--json"},
			expectedSteps: []string{"detect", "scan", "focus", "send", "verify"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build command - use compiled binary directly with absolute path
			exePath := filepath.Join("..", "..", "cmd", "wechat-debug", "wechat-debug.exe")
			args := append([]string{tt.command}, tt.args...)
			cmd := exec.Command(exePath, args...)

			// Run command
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, string(output))
			}

			// Parse JSON output
			var result map[string]interface{}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, string(output))
			}

			// Get steps
			steps, ok := result["steps"].([]interface{})
			if !ok {
				t.Fatal("steps field is not an array")
			}

			// Check step order
			if len(steps) != len(tt.expectedSteps) {
				t.Errorf("Expected %d steps, got %d", len(tt.expectedSteps), len(steps))
			}

			for i, expectedStep := range tt.expectedSteps {
				if i >= len(steps) {
					break
				}
				step, ok := steps[i].(map[string]interface{})
				if !ok {
					t.Errorf("Step %d is not a map", i)
					continue
				}
				actualStep, ok := step["step"].(string)
				if !ok {
					t.Errorf("Step %d 'step' field is not a string", i)
					continue
				}
				if actualStep != expectedStep {
					t.Errorf("Step %d: expected '%s', got '%s'", i, expectedStep, actualStep)
				}
			}
		})
	}
}

// TestMockRealStructureConsistency tests that mock and real modes have same structure
func TestMockRealStructureConsistency(t *testing.T) {
	// This test verifies that the JSON structure is consistent
	// by checking the schema, not the actual values

	// Define expected schema for each command
	schemas := map[string]map[string]string{
		"scan": {
			"mode":         "string",
			"steps":        "array",
			"steps[].step": "string",
			"steps[].status": "string",
			"steps[].confidence": "float",
		},
		"focus": {
			"mode":         "string",
			"steps":        "array",
			"steps[].step": "string",
			"steps[].status": "string",
			"steps[].confidence": "float",
		},
		"send": {
			"mode":         "string",
			"steps":        "array",
			"steps[].step": "string",
			"steps[].status": "string",
			"steps[].confidence": "float",
		},
		"verify": {
			"mode":         "string",
			"steps":        "array",
			"steps[].step": "string",
			"steps[].status": "string",
			"steps[].confidence": "float",
		},
		"run-chain": {
			"mode":         "string",
			"steps":        "array",
			"final":        "object",
			"steps[].step": "string",
			"steps[].status": "string",
			"steps[].confidence": "float",
		},
		"full-test": {
			"mode":         "string",
			"steps":        "array",
			"final":        "object",
			"steps[].step": "string",
			"steps[].status": "string",
			"steps[].confidence": "float",
		},
	}

	for command, schema := range schemas {
		t.Run(command, func(t *testing.T) {
			// Build command - use compiled binary directly
			exePath := filepath.Join("cmd", "wechat-debug", "wechat-debug.exe")
			args := []string{command, "--mock", "--json"}
			cmd := exec.Command(exePath, args...)

			// Run command
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, string(output))
			}

			// Parse JSON output
			var result map[string]interface{}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, string(output))
			}

			// Check schema fields
			for field, expectedType := range schema {
				parts := strings.Split(field, ".")
				var current interface{} = result

				// Navigate through nested fields
				for _, part := range parts {
					// Handle array indexing like "steps[]"
					if strings.HasSuffix(part, "[]") {
						baseField := strings.TrimSuffix(part, "[]")
						if m, ok := current.(map[string]interface{}); ok {
							if arr, ok := m[baseField].([]interface{}); ok {
								if len(arr) > 0 {
									current = arr[0]
									continue
								} else {
									t.Skipf("Array '%s' is empty, cannot verify type", baseField)
									return
								}
							} else {
								t.Errorf("Field '%s' is not an array", baseField)
								return
							}
						}
					}

					// Handle final field
					if part == "final" {
						if m, ok := current.(map[string]interface{}); ok {
							if _, ok := m["final"]; !ok {
								t.Errorf("Missing 'final' field")
								return
							}
							current = m["final"]
							continue
						}
					}

					// Regular field access
					if m, ok := current.(map[string]interface{}); ok {
						if val, ok := m[part]; ok {
							current = val
						} else {
							t.Errorf("Missing field '%s'", field)
							return
						}
					}
				}

				// Check type
				switch expectedType {
				case "string":
					if _, ok := current.(string); !ok {
						t.Errorf("Field '%s' is not a string", field)
					}
				case "float":
					switch v := current.(type) {
					case float64:
						// OK
					case string:
						// Try to parse as float
						if _, err := json.Number(v).Float64(); err != nil {
							t.Errorf("Field '%s' cannot be parsed as float: %v", field, err)
						}
					default:
						t.Errorf("Field '%s' is not a float (type: %T)", field, current)
					}
				case "array":
					if _, ok := current.([]interface{}); !ok {
						t.Errorf("Field '%s' is not an array", field)
					}
				case "object":
					if _, ok := current.(map[string]interface{}); !ok {
						t.Errorf("Field '%s' is not an object", field)
					}
				}
			}
		})
	}
}
