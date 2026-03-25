package config

import (
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback any
		envValue string
		want     any
	}{
		{
			name:     "returns fallback when env not set",
			key:      "NOT_SET_VAR_INT",
			fallback: 42,
			envValue: "",
			want:     42,
		},
		{
			name:     "returns fallback when env not set for bool",
			key:      "NOT_SET_VAR_BOOL",
			fallback: true,
			envValue: "",
			want:     true,
		},
		{
			name:     "returns fallback when env not set for string",
			key:      "NOT_SET_VAR_STRING",
			fallback: "default",
			envValue: "",
			want:     "default",
		},
		{
			name:     "returns fallback when env not set for float64",
			key:      "NOT_SET_VAR_FLOAT",
			fallback: 3.14,
			envValue: "",
			want:     3.14,
		},
		{
			name:     "parses valid int",
			key:      "TEST_INT_VALID",
			fallback: 0,
			envValue: "123",
			want:     123,
		},
		{
			name:     "returns fallback for invalid int",
			key:      "TEST_INT_INVALID",
			fallback: 99,
			envValue: "not_a_number",
			want:     99,
		},
		{
			name:     "parses valid bool true",
			key:      "TEST_BOOL_TRUE",
			fallback: false,
			envValue: "true",
			want:     true,
		},
		{
			name:     "parses valid bool 1",
			key:      "TEST_BOOL_1",
			fallback: false,
			envValue: "1",
			want:     true,
		},
		{
			name:     "parses valid bool false",
			key:      "TEST_BOOL_FALSE",
			fallback: true,
			envValue: "false",
			want:     false,
		},
		{
			name:     "returns fallback for invalid bool",
			key:      "TEST_BOOL_INVALID",
			fallback: true,
			envValue: "maybe",
			want:     true,
		},
		{
			name:     "returns string value",
			key:      "TEST_STRING",
			fallback: "default",
			envValue: "hello",
			want:     "hello",
		},
		{
			name:     "parses valid float64",
			key:      "TEST_FLOAT_VALID",
			fallback: 0.0,
			envValue: "2.718",
			want:     2.718,
		},
		{
			name:     "returns fallback for invalid float64",
			key:      "TEST_FLOAT_INVALID",
			fallback: 1.41,
			envValue: "not_a_float",
			want:     1.41,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			var got any
			switch tt.fallback.(type) {
			case int:
				got = GetEnv(tt.key, tt.fallback.(int))
			case bool:
				got = GetEnv(tt.key, tt.fallback.(bool))
			case string:
				got = GetEnv(tt.key, tt.fallback.(string))
			case float64:
				got = GetEnv(tt.key, tt.fallback.(float64))
			}

			if got != tt.want {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
