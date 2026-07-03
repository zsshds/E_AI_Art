package task

import (
	"errors"
	"testing"
)

func TestIsAMQPConnectionClosedErrorRecognizesRabbitMQ504(t *testing.T) {
	err := errors.New("open channel: Exception (504) Reason: \"channel/connection is not open\"")

	if !isAMQPConnectionClosedError(err) {
		t.Fatalf("isAMQPConnectionClosedError(%q) = false, want true", err)
	}
}

func TestIsAMQPConnectionClosedErrorIgnoresOtherErrors(t *testing.T) {
	err := errors.New("get message: context canceled")

	if isAMQPConnectionClosedError(err) {
		t.Fatalf("isAMQPConnectionClosedError(%q) = true, want false", err)
	}
}
