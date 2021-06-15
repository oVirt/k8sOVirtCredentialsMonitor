package k8sOVirtCredentialsMonitor

import (
	"testing"
)

// Logger provides pluggable logging for this library.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func NewTestLogger(t *testing.T) Logger {
	return &testLogger{
		t: t,
	}
}

type testLogger struct {
	t *testing.T
}

func (t *testLogger) Debugf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}

func (t *testLogger) Infof(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}

func (t *testLogger) Warningf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}

func (t *testLogger) Errorf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}

type nopLogger struct {
}

func (n nopLogger) Debugf(_ string, _ ...interface{}) {

}

func (n nopLogger) Infof(_ string, _ ...interface{}) {

}

func (n nopLogger) Warningf(_ string, _ ...interface{}) {

}

func (n nopLogger) Errorf(_ string, _ ...interface{}) {

}
