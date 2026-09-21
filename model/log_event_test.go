package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogEventBrokerPublishesByViewerScope(t *testing.T) {
	broker := newLogEventBroker()
	userOneEvents, unsubscribeUserOne := broker.subscribe(1, false)
	defer unsubscribeUserOne()
	userTwoEvents, unsubscribeUserTwo := broker.subscribe(2, false)
	defer unsubscribeUserTwo()
	adminEvents, unsubscribeAdmin := broker.subscribe(1, true)
	defer unsubscribeAdmin()

	broker.publish(1)

	requireEvent(t, userOneEvents)
	assertNoEvent(t, userTwoEvents)
	requireEvent(t, adminEvents)
}

func TestLogEventBrokerStopsPublishingAfterUnsubscribe(t *testing.T) {
	broker := newLogEventBroker()
	events, unsubscribe := broker.subscribe(1, false)

	unsubscribe()
	broker.publish(1)

	assertNoEvent(t, events)
}

func requireEvent(t *testing.T, events <-chan struct{}) {
	t.Helper()

	select {
	case <-events:
		return
	default:
		require.Fail(t, "expected a log event")
	}
}

func assertNoEvent(t *testing.T, events <-chan struct{}) {
	t.Helper()

	select {
	case <-events:
		assert.Fail(t, "received an unexpected log event")
	default:
	}
}
