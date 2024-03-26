// Package xsenv provides utilities for loading environment configurations
// from JSON files or readers, specifically focusing on service bindings.
package xsenv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Source string

const (
	FileSource        Source = "file"
	EnvironmentSource Source = "environment"
	RawSource         Source = "raw"
)

const (
	// DefaultEnvFile is the default filename for the environment configuration.
	DefaultEnvFile = "default-env.json"

	// EnvironmentKey is the key used to access the environment configuration.
	EnvironmentKey = "VCAP_SERVICES"
)

// ErrServiceNotFound indicates that the requested service was not found in the environment configuration.
var (
	ErrServiceNotFound = errors.New("service not found")
	ErrFieldMissing    = errors.New("field(s) missing")
)

// LoadEnv loads the environment configuration from environment variables if available, otherwise from the default file.
// It returns an Env instance on success or an error if loading fails.
func LoadEnv() (*Env, error) {
	env, ok := os.LookupEnv(EnvironmentKey)
	if ok {
		return loadEnvFromBytes([]byte(env), EnvironmentSource)
	}
	return LoadEnvFromFile(DefaultEnvFile)
}

// LoadEnvFromReader loads the environment configuration from an io.Reader.
// It returns an Env instance on success or an error if loading fails.
func LoadEnvFromReader(reader io.Reader) (*Env, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return loadEnvFromBytes(data, RawSource)
}

// LoadEnvFromFile loads the environment configuration from a specified file.
// It returns an Env instance on success or an error if loading fails.
func LoadEnvFromFile(fileName string) (*Env, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return loadEnvFromBytes(data, FileSource)
}

// loadEnvFromBytes is an internal function that loads environment configuration
// from a byte slice. It is used by LoadEnvFromReader and LoadEnvFromFile.
func loadEnvFromBytes(data []byte, source Source) (*Env, error) {
	parseEnv := struct {
		Services map[string][]*json.RawMessage `json:"VCAP_SERVICES"`
	}{}
	if err := json.Unmarshal(data, &parseEnv); err != nil {
		return nil, err
	}

	type parseName struct {
		Name string `json:"name"`
	}
	m := make(map[string]*json.RawMessage)
	for _, services := range parseEnv.Services {
		for _, service := range services {
			var name parseName
			if err := json.Unmarshal(*service, &name); err != nil {
				return nil, err
			}
			m[strings.ToLower(name.Name)] = service
		}
	}

	return &Env{source, m}, nil
}

// Env represents the environment configuration, holding service configurations by name.
type Env struct {
	Source Source
	// ServicesByName maps service names to their JSON configuration.
	ServicesByName map[string]*json.RawMessage
}

// LoadService loads a service configuration by name into a UnmarshalService.
// It returns an error if the service cannot be found or the unmarshaling fails.
func (e *Env) LoadService(target UnmarshalService, name string) error {
	msg, ok := e.ServicesByName[strings.ToLower(name)]
	if !ok {
		return ErrServiceNotFound
	}
	return target.UnmarshalService(msg)
}

// UnmarshalService is an interface for types that can unmarshal
// a service configuration from a JSON message.
type UnmarshalService interface {
	UnmarshalService(*json.RawMessage) error
}

// MissingFieldError returns an error indicating that a field is missing.
// This is useful when implementing UnmarshalService.
func MissingFieldError(field string) error {
	return fmt.Errorf("%w: %s", ErrFieldMissing, field)
}

type Fields = map[string]bool

// CheckAllFields checks if all fields in a map are set to true (present).
// If a field is missing, it returns an error.
func CheckAllFields(m Fields) error {
	var missing []string
	for name, ok := range m {
		if !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrFieldMissing, strings.Join(missing, ", "))
	}
	return nil
}
