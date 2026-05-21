package mq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRetryCountFromHeaders(t *testing.T) {
	headers := amqp.Table{retryHeader: int32(3)}
	if got := retryCountFromHeaders(headers); got != 3 {
		t.Fatalf("retry count = %d, want 3", got)
	}
	if got := retryCountFromHeaders(nil); got != 0 {
		t.Fatalf("nil headers retry count = %d, want 0", got)
	}
}
