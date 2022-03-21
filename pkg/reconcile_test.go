package pkg

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gopkg.in/square/go-jose.v2/json"
)

type TestValidation struct {
	SetupRun      bool
	SetupError    bool
	ValidateRun   bool
	ValidateError bool
	SleepDuration int
}

func (t *TestValidation) Setup(ctx context.Context) error {
	if t.SetupError {
		return fmt.Errorf("Error during setup")
	}
	t.SetupRun = true
	return nil
}

func (t *TestValidation) Validate(ctx context.Context) ([]ValidationError, error) {
	if t.SleepDuration > 0 {
		finished := make(chan bool)
		go func() {
			time.Sleep(time.Duration(t.SleepDuration) * time.Second)
			finished <- true
		}()
		select {
		case <-finished:
			break
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
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

func TestNewRunnerConfig(t *testing.T) {
	runnerConfg := newRunnerConfig()
	assert.Equal(t, 0, runnerConfg.Timeout)
}

func TestValidationTimeoutFail(t *testing.T) {
	tv := TestValidation{
		ValidateRun:   false,
		SetupRun:      false,
		SleepDuration: 2,
	}

	os.Setenv("RUNNER_TIMEOUT", "1")

	vr := NewValidationRunner(&tv)
	err := vr.Run()
	assert.NotNil(t, err)
	assert.Error(t, err, "context.deadlineExceededError{}")
	assert.True(t, tv.SetupRun)
	assert.False(t, tv.ValidateRun)
}

func TestValidationTimeoutOK(t *testing.T) {
	tv := TestValidation{
		ValidateRun:   false,
		SetupRun:      false,
		SleepDuration: 2,
	}

	os.Setenv("RUNNER_TIMEOUT", "4")

	vr := NewValidationRunner(&tv)
	err := vr.Run()
	assert.Nil(t, err)
	assert.True(t, tv.SetupRun)
	assert.True(t, tv.ValidateRun)
}
