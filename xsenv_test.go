package xsenv

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUnmarshalService struct {
	mock.Mock
}

func (m *MockUnmarshalService) UnmarshalService(data *json.RawMessage) error {
	args := m.Called(data)
	return args.Error(0)
}

func TestLoadEnv(t *testing.T) {
	// Set up environment variable for testing
	_ = os.Setenv(EnvironmentKey, `{"VCAP_SERVICES": {"test_service": [{"name": "test"}]}}`)
	defer func() {
		_ = os.Unsetenv(EnvironmentKey)
	}()

	env, err := LoadEnv()
	assert.NoError(t, err)
	assert.Equal(t, EnvironmentSource, env.Source)
	_, exists := env.ServicesByName["test"]
	assert.True(t, exists)

	// Test loading from default file when environment variable is not set
	_ = os.Unsetenv(EnvironmentKey)

	// create default-env.json if it does not exist
	if _, err := os.Stat(DefaultEnvFile); os.IsNotExist(err) {
		file, err := os.Create(DefaultEnvFile)
		assert.NoError(t, err)
		_, err = file.WriteString(`{"VCAP_SERVICES": {"test_service": [{"name": "test"}]}}`)
		assert.NoError(t, err)
		err = file.Close()
		assert.NoError(t, err)
		defer func() {
			_ = os.Remove(DefaultEnvFile)
		}()
	}

	env, err = LoadEnv()
	assert.NoError(t, err)
	assert.Equal(t, FileSource, env.Source)
}

func TestLoadEnvFromReader(t *testing.T) {
	reader := bytes.NewBufferString(`{"VCAP_SERVICES": {"test_service": [{"name": "test"}]}}`)
	env, err := LoadEnvFromReader(reader)
	assert.NoError(t, err)
	assert.Equal(t, RawSource, env.Source)
	_, exists := env.ServicesByName["test"]
	assert.True(t, exists)
}

func TestLoadService(t *testing.T) {
	data := `{"VCAP_SERVICES": {"test_service": [{"name": "test"}]}}`
	env, _ := loadEnvFromBytes([]byte(data), RawSource)

	mockService := new(MockUnmarshalService)
	mockService.On("UnmarshalService", mock.Anything).Return(nil)

	err := env.LoadService(mockService, "test")
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test service not found
	err = env.LoadService(mockService, "nonexistent")
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

func TestMissingFieldError(t *testing.T) {
	testCases := []struct {
		field    string
		expected error
	}{
		{"username", fmt.Errorf("%w: %s", ErrFieldMissing, "username")},
		{"password", fmt.Errorf("%w: %s", ErrFieldMissing, "password")},
		{"", fmt.Errorf("%w: %s", ErrFieldMissing, "")}, // Testing with an empty string
	}

	for _, tc := range testCases {
		err := MissingFieldError(tc.field)
		assert.True(t, errors.Is(err, ErrFieldMissing))
		assert.Equal(t, tc.expected.Error(), err.Error())
	}
}

func TestCheckAllFields(t *testing.T) {
	testCases := []struct {
		name     string
		input    Fields
		expected error
	}{
		{
			name: "All fields present",
			input: Fields{
				"username": true,
				"password": true,
			},
			expected: nil,
		},
		{
			name: "One field missing",
			input: Fields{
				"username": true,
				"password": false, // this field is missing
			},
			expected: fmt.Errorf("%w: %s", ErrFieldMissing, "password"),
		},
		{
			name: "Multiple fields missing",
			input: Fields{
				"username": false,
				"password": false, // both fields are missing
			},
			expected: fmt.Errorf("%w: %s", ErrFieldMissing, "username, password"),
		},
		{
			name:     "Empty fields map",
			input:    Fields{},
			expected: nil, // No fields to check, so it should not error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckAllFields(tc.input)
			if tc.expected == nil {
				assert.NoError(t, err)
			} else {
				assert.True(t, errors.Is(err, ErrFieldMissing))
				// Since map iteration is random, we need to compare without considering the order
				assert.ElementsMatch(t, strings.Split(tc.expected.Error(), ": ")[1:], strings.Split(err.Error(), ": ")[1:])
			}
		})
	}
}
