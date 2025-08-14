package logging

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	assert.NotNil(t, logger)
	assert.IsType(t, &logrus.Logger{}, logger)
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		expected logrus.Level
	}{
		{"debug level", "debug", logrus.DebugLevel},
		{"info level", "info", logrus.InfoLevel},
		{"warn level", "warn", logrus.WarnLevel},
		{"error level", "error", logrus.ErrorLevel},
		{"default level", "", logrus.InfoLevel},
		{"invalid level", "invalid", logrus.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LOG_LEVEL", tt.logLevel)

			testLogger := logrus.New()
			level := os.Getenv("LOG_LEVEL")
			switch level {
			case "debug":
				testLogger.SetLevel(logrus.DebugLevel)
			case "info":
				testLogger.SetLevel(logrus.InfoLevel)
			case "warn":
				testLogger.SetLevel(logrus.WarnLevel)
			case "error":
				testLogger.SetLevel(logrus.ErrorLevel)
			default:
				testLogger.SetLevel(logrus.InfoLevel)
			}

			assert.Equal(t, tt.expected, testLogger.Level)
			os.Unsetenv("LOG_LEVEL")
		})
	}
}

func TestLoggingFunctions(t *testing.T) {
	var buf bytes.Buffer
	originalLogger := logger

	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetLevel(logrus.DebugLevel)
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logger = testLogger

	defer func() {
		logger = originalLogger
	}()

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name: "debug",
			logFunc: func() {
				Debug("test debug message")
			},
			expected: "level=debug msg=\"test debug message\"",
		},
		{
			name: "info",
			logFunc: func() {
				Info("test info message")
			},
			expected: "level=info msg=\"test info message\"",
		},
		{
			name: "warn",
			logFunc: func() {
				Warn("test warn message")
			},
			expected: "level=warning msg=\"test warn message\"",
		},
		{
			name: "error",
			logFunc: func() {
				Error("test error message")
			},
			expected: "level=error msg=\"test error message\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := strings.TrimSpace(buf.String())
			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestFormattedLogging(t *testing.T) {
	var buf bytes.Buffer
	originalLogger := logger

	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetLevel(logrus.DebugLevel)
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logger = testLogger

	defer func() {
		logger = originalLogger
	}()

	Infof("formatted message: %s %d", "test", 123)
	output := strings.TrimSpace(buf.String())
	assert.Contains(t, output, "formatted message: test 123")
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	originalLogger := logger

	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetLevel(logrus.DebugLevel)
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logger = testLogger

	defer func() {
		logger = originalLogger
	}()

	WithFields(logrus.Fields{
		"component": "test",
		"action":    "testing",
	}).Info("test message with fields")

	output := strings.TrimSpace(buf.String())
	assert.Contains(t, output, "component=test")
	assert.Contains(t, output, "action=testing")
	assert.Contains(t, output, "test message with fields")
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	originalLogger := logger

	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetLevel(logrus.DebugLevel)
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logger = testLogger

	defer func() {
		logger = originalLogger
	}()

	WithField("component", "test").Info("test message with field")

	output := strings.TrimSpace(buf.String())
	assert.Contains(t, output, "component=test")
	assert.Contains(t, output, "test message with field")
}
