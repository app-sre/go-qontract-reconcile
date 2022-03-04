package pkg

import (
	"bytes"
	"fmt"
	"net/url"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gopkg.in/square/go-jose.v2/json"
)

type TestValidation struct {
	SetupRun      bool
	SetupError    bool
	ValidateRun   bool
	ValidateError bool
}

func (t *TestValidation) Setup() error {
	if t.SetupError {
		return fmt.Errorf("Error during setup")
	}
	t.SetupRun = true
	return nil
}

func (t *TestValidation) Validate() ([]ValidationError, error) {
	if t.ValidateError {
		return nil, fmt.Errorf("Error during validate")
	}
	if !t.SetupRun {
		return nil, fmt.Errorf("Setup not run")
	}
	t.ValidateRun = true
	return []ValidationError{{
		Path:       "/foo/bar",
		Validation: "test",
		Error:      fmt.Errorf("test"),
	},
	}, nil
}

func TestValidationRunner(t *testing.T) {
	tv := TestValidation{
		SetupRun:    false,
		ValidateRun: false,
	}
	vr := NewValidationRunner(&tv)
	err := vr.Run()
	assert.Nil(t, err)
	assert.True(t, tv.SetupRun)
	assert.True(t, tv.ValidateRun)
}

func TestValidationRunnerSetupFailed(t *testing.T) {
	tv := TestValidation{
		SetupError:  true,
		SetupRun:    false,
		ValidateRun: false,
	}
	vr := NewValidationRunner(&tv)
	err := vr.Run()
	assert.NotNil(t, err)
	assert.False(t, tv.ValidateRun)
}

func TestValidationRunnerValidateFailed(t *testing.T) {
	tv := TestValidation{
		ValidateError: true,
		SetupRun:      false,
		ValidateRun:   false,
	}
	vr := NewValidationRunner(&tv)
	err := vr.Run()
	assert.NotNil(t, err)
	assert.True(t, tv.SetupRun)
	assert.False(t, tv.ValidateRun)
}

type MemorySink struct {
	*bytes.Buffer
}

func (s *MemorySink) Close() error { return nil }
func (s *MemorySink) Sync() error  { return nil }

func TestValidationRunnerLogs(t *testing.T) {
	sink := &MemorySink{new(bytes.Buffer)}
	zap.RegisterSink("memory", func(*url.URL) (zap.Sink, error) {
		return sink, nil
	})

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.OutputPaths = []string{"memory://"}
	logger, err := loggerConfig.Build()
	assert.Nil(t, err)

	zap.ReplaceGlobals(logger)

	tv := TestValidation{}
	vr := NewValidationRunner(&tv)
	vr.Run()

	var structuredOutput map[string]interface{}
	err = json.Unmarshal(sink.Bytes(), &structuredOutput)
	assert.Nil(t, err)
	assert.Equal(t, "Validation error", structuredOutput["msg"])
	assert.Equal(t, "/foo/bar", structuredOutput["path"])
	assert.Equal(t, "test", structuredOutput["validation"])
	assert.Equal(t, "test", structuredOutput["error"])
}

func reconcileSetupViper() *viper.Viper {
	v := viper.New()

	qontract := make(map[string]interface{})
	qontract["foo"] = "bar"
	vault := make(map[string]interface{})
	vault["foo"] = "bar"

	v.Set("dry_run", false)
	v.Set("qontract", qontract)
	v.Set("vault", vault)
	return v
}

func TestNewRunnerConfig(t *testing.T) {
	runnerConfg := NewRunnerConfig(reconcileSetupViper())
	assert.False(t, runnerConfg.DryRun)
}
