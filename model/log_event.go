package model

import "sync"

type logEventSubscriber struct {
	userId     int
	includeAll bool
	events     chan struct{}
}

type logEventBroker struct {
	mu             sync.RWMutex
	nextSubscriber uint64
	subscribers    map[uint64]logEventSubscriber
}

func newLogEventBroker() *logEventBroker {
	return &logEventBroker{
		subscribers: make(map[uint64]logEventSubscriber),
	}
}

func (broker *logEventBroker) subscribe(userId int, includeAll bool) (<-chan struct{}, func()) {
	broker.mu.Lock()
	broker.nextSubscriber++
	subscriberId := broker.nextSubscriber
	events := make(chan struct{}, 1)
	broker.subscribers[subscriberId] = logEventSubscriber{
		userId:     userId,
		includeAll: includeAll,
		events:     events,
	}
	broker.mu.Unlock()

	var unsubscribeOnce sync.Once
	unsubscribe := func() {
		unsubscribeOnce.Do(func() {
			broker.mu.Lock()
			delete(broker.subscribers, subscriberId)
			broker.mu.Unlock()
		})
	}

	return events, unsubscribe
}

func (broker *logEventBroker) publish(userId int) {
	broker.mu.RLock()
	defer broker.mu.RUnlock()

	for _, subscriber := range broker.subscribers {
		if !subscriber.includeAll && subscriber.userId != userId {
			continue
		}

		select {
		case subscriber.events <- struct{}{}:
		default:
			// One pending refresh already covers every log written before it runs.
		}
	}
}

var usageLogEvents = newLogEventBroker()

func SubscribeLogEvents(userId int, includeAll bool) (<-chan struct{}, func()) {
	return usageLogEvents.subscribe(userId, includeAll)
}

func publishLogEvent(userId int) {
	usageLogEvents.publish(userId)
}
