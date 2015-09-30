package test

import (
	"testing"
	"time"

	. "github.com/prometheus/alertmanager/test"
)

var somethingConfig = `
routes:
- send_to: "default"
  group_wait:     1s
  group_interval: 1s

notification_configs:
- name: "default"
  send_resolved: true

  webhook_configs:
  - url: 'http://localhost:8088'
`

func TestSomething(t *testing.T) {
	t.Parallel()

	// Create a new acceptance test that instantiates new Alertmanagers
	// with the given configuration and verifies times with the given
	// tollerance.
	at := NewAcceptanceTest(t, &AcceptanceOpts{
		Tolerance: 150 * time.Millisecond,
		Config:    somethingConfig,
	})

	// Create a new Alertmanager process listening to a random port
	am := at.Alertmanager()
	// Create a collector to which alerts can be written and verified
	// against a set of expected alert notifications.
	co := at.Collector("webhook")

	// Run something that satisfies the webhook interface to which the
	// Alertmanager pushes as defined by its configuration.
	go NewWebhook(":8088", co).Run()

	// Declare pushes to be made to the Alertmanager at the given time.
	// Times are provided in fractions of seconds.
	am.Push(At(1), Alert("alertname", "test").Active(1))
	am.Push(At(3.5), Alert("alertname", "test").Active(1, 3))

	// Declare which alerts are expected to arrive at the collector within
	// the defined time intervals.
	co.Want(Between(2, 2.5), Alert("alertname", "test").Active(1))
	co.Want(Between(3, 3.5), Alert("alertname", "test").Active(1))
	co.Want(Between(4, 4.5), Alert("alertname", "test").Active(1, 3))

	// Start the flow as defined above and run the checks afterwards.
	at.Run()
}

var batchConfig = `
routes:
- send_to: "default"
  group_wait:     1s
  group_interval: 1s

notification_configs:
- name:            "default"
  send_resolved:   true
  repeat_interval: 5s

  webhook_configs:
  - url: 'http://localhost:8089'
`

func TestBatching(t *testing.T) {
	t.Parallel()

	at := NewAcceptanceTest(t, &AcceptanceOpts{
		Tolerance: 150 * time.Millisecond,
		Config:    batchConfig,
	})

	am := at.Alertmanager()
	co := at.Collector("webhook")

	go NewWebhook(":8089", co).Run()

	am.Push(At(1.1), Alert("alertname", "test1").Active(1))
	am.Push(At(1.9), Alert("alertname", "test5").Active(1))
	am.Push(At(2.3),
		Alert("alertname", "test2").Active(1.5),
		Alert("alertname", "test3").Active(1.5),
		Alert("alertname", "test4").Active(1.6),
	)

	co.Want(Between(2.0, 2.5),
		Alert("alertname", "test1").Active(1),
		Alert("alertname", "test5").Active(1),
	)
	// Only expect the new ones with the next group interval.
	co.Want(Between(3, 3.5),
		Alert("alertname", "test2").Active(1.5),
		Alert("alertname", "test3").Active(1.5),
		Alert("alertname", "test4").Active(1.6),
	)

	// While no changes happen expect no additional notifications
	// until the 5s repeat interval has ended.
	co.Want(Between(7, 8.5),
		Alert("alertname", "test1").Active(1),
		Alert("alertname", "test2").Active(1.5),
		Alert("alertname", "test3").Active(1.5),
		Alert("alertname", "test4").Active(1.6),
		Alert("alertname", "test5").Active(1),
	)

	at.Run()
}