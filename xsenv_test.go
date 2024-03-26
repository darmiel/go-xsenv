package xsenv

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
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
	os.Setenv(EnvironmentKey, `{"VCAP_SERVICES": {"test_service": [{"name": "test"}]}}`)
	defer os.Unsetenv(EnvironmentKey)

	env, err := LoadEnv()
	assert.NoError(t, err)
	assert.Equal(t, EnvironmentSource, env.Source)
	_, exists := env.ServicesByName["test"]
	assert.True(t, exists)

	// Test loading from default file when environment variable is not set
	os.Unsetenv(EnvironmentKey)

	// create default-env.json if it does not exist
	if _, err := os.Stat(DefaultEnvFile); os.IsNotExist(err) {
		file, err := os.Create(DefaultEnvFile)
		assert.NoError(t, err)
		_, err = file.WriteString(`{"VCAP_SERVICES": {"test_service": [{"name": "test"}]}}`)
		assert.NoError(t, err)
		err = file.Close()
		assert.NoError(t, err)
		defer os.Remove(DefaultEnvFile)
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
	err := MissingFieldError("missing_field")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFieldMissing))
}

func TestCheckAllFields(t *testing.T) {
	fields := Fields{"field1": true, "field2": false}
	err := CheckAllFields(fields)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFieldMissing))

	fields = Fields{"field1": true, "field2": true}
	err = CheckAllFields(fields)
	assert.NoError(t, err)
}
