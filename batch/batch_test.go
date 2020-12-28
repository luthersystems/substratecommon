package batch_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/luthersystems/substratecommon"
	"github.com/luthersystems/substratecommon/batch"
	"github.com/luthersystems/substratecommon/substratewrapper"

	"github.com/sirupsen/logrus"
)

func Test001(t *testing.T) {
	var TS001 = "2000-01-01T00:00:00-08:00"
	var TS002 = "2000-01-02T00:00:00-08:00"
	var TS003 = "2000-01-03T00:00:00-08:00"

	var tsMutex *sync.Mutex
	var tsString string

	tsMutex = &sync.Mutex{}
	tsString = TS001

	tsAssign := func(tsInput string) {
		tsMutex.Lock()
		defer tsMutex.Unlock()

		tsString = tsInput
	}

	tsGenerator := func(ctx context.Context) string {
		tsMutex.Lock()
		defer tsMutex.Unlock()

		return tsString
	}

	log := logrus.New()

	log.SetLevel(logrus.DebugLevel)

	err := substratecommon.Connect(
		func(substrate substratecommon.Substrate) error {
			sw := substratewrapper.NewSubstrateWrapper(substrate)

			siwm, err := sw.NewMockFrom("test", "test", nil)
			if err != nil {
				return err
			}

			extraOpts := []substratecommon.Config{substratecommon.WithLog(log), substratecommon.WithTimestampGenerator(tsGenerator)}

			phylumString := `
(in-package 'user)
(use-package 'router)
(cc:infof () "init batch.lisp")

(defun schedule (batch-name req when-time)
  (let ([err (batch:schedule-request batch-name req when-time)])
    (if (nil? err)
      (route-success ())
      (route-failure err))))

(defendpoint init ()
  (schedule "init_batch" (sorted-map) (cc:timestamp (cc:now))))

(set 'storage-key-recent-input "RECENT_INPUT")

(batch:handler 'test_batch (lambda (batch-name rep bad)
  (if bad
    (progn
      (cc:storage-put storage-key-recent-input (string:join (list "error:" bad) " "))
      (cc:infof () "error: {}" bad))
    (progn
      (cc:storage-put storage-key-recent-input rep)
      (cc:infof () rep)))
  (route-success ())))

(batch:handler 'init_batch (lambda (batch-name rep bad)
  (cc:infof () (get rep "init message"))
  (route-success ())))

(defendpoint schedule_request (batch_name req when)
  (schedule batch_name req when))

(defendpoint schedule_request_now (batch_name req)
  (cc:infof () "in schedule_request_now")
  (cc:infof () batch_name)
  (schedule batch_name req (cc:timestamp (cc:now))))

(defendpoint set_batching_paused (val)
  (cc:set-app-property "BATCHING_PAUSED" val)
  (route-success ()))

(defendpoint get_recent_input ()
  (route-success (to-string (cc:storage-get storage-key-recent-input))))
`

			err = siwm.Init(substratecommon.EncodePhylumBytes(phylumString))
			if err != nil {
				t.Fatal(err)
			}

			driver := batch.NewDriver(siwm.Upcast(), batch.WithLog(log), batch.WithLogField("TESTFIELD", "TESTVALUE"))

			lastReceivedMessage := "none"

			ticker := driver.Register(context.Background(), "test_batch", time.Duration(1)*time.Hour, func(batchID string, requestID string, message json.RawMessage) (json.RawMessage, error) {
				/****/ if string(message) == "\"ping1\"" {
					lastReceivedMessage = "ping1"
					return []byte("\"pong1\""), nil
				} else if string(message) == "\"ping2\"" {
					lastReceivedMessage = "ping2"
					return nil, errors.New("ping2 error")
				} else if string(message) == "\"ping3\"" {
					lastReceivedMessage = "ping3"
					return []byte("\"pong3\""), nil
				} else {
					panic(nil)
				}
			}, extraOpts...)

			recentInput := ""

			doTick := func(t *testing.T) {
				ticker.Tick(context.Background())

				sr, err := siwm.Call("get_recent_input", append(extraOpts, substratecommon.WithParams([]interface{}{}))...)
				if err != nil || sr.HasError {
					t.Fatal()
				}

				err = json.Unmarshal(sr.ResultJSON, &recentInput)
				if err != nil {
					t.Fatal(err)
				}
			}

			table := []struct {
				name      string
				method    string
				params    interface{}
				validator func(*testing.T) bool
			}{
				{
					"first test - immediately scheduled batch request",
					"schedule_request_now",
					[]interface{}{
						"test_batch",
						"ping1",
					},
					func(t *testing.T) bool {
						return lastReceivedMessage == "ping1" && recentInput == "pong1"
					},
				},

				{
					"second test - immediately scheduled batch request with failure",
					"schedule_request_now",
					[]interface{}{
						"test_batch",
						"ping2",
					},
					func(t *testing.T) bool {
						return lastReceivedMessage == "ping2" && recentInput == "error: ping2 error"
					},
				},

				{
					"third test - schedule at times other than now",
					"schedule_request",
					[]interface{}{
						"test_batch",
						"ping3",
						TS002,
					},
					func(t *testing.T) bool {
						// should not have got ping3 yet
						if lastReceivedMessage != "ping2" {
							return false
						}

						// now artificially advance time
						tsAssign(TS003)

						// tick (again)
						doTick(t)

						// now it should have worked
						return lastReceivedMessage == "ping3" && recentInput == "pong3"
					},
				},
			}

			for _, tt := range table {
				t.Run(tt.name, func(t *testing.T) {
					sr, err := siwm.Call(tt.method, append(extraOpts, substratecommon.WithParams(tt.params))...)
					if err != nil || sr.HasError {
						t.Fatal()
					}

					doTick(t)

					if !(tt.validator(t)) {
						t.Fatal()
					}
				})
			}

			return nil
		},
		substratecommon.ConnectWithCommand(os.Getenv("SUBSTRATE_PLUGIN_FILE")))
	if err != nil {
		t.Fatal(err)
	}
}
