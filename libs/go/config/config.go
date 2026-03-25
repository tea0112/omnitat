package config

import (
	"os"
	"strconv"
)

func GetEnv[T any](key string, fallback T) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	switch any(fallback).(type) {
	case int:
		v, err := strconv.Atoi(value)
		if err != nil {
			return fallback
		}

		i, ok := any(v).(T)
		if ok {
			return i
		}
	case bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fallback
		}

		b, ok := any(v).(T)
		if ok {
			return b
		}
	case string:
		return any(value).(T)
	case float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fallback
		}

		return any(f).(T)
	}

	return fallback
}
