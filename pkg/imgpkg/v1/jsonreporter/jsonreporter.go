// Copyright 2025 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// Package jsonreporter progress reporter for JSONMessage
package jsonreporter

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	v1 "carvel.dev/imgpkg/pkg/imgpkg/v1"
)

// NewProgressTracker creates a new ProgressTracker for JSONMessages
func NewProgressTracker(out io.Writer) *JSONMessageTracker {
	return &JSONMessageTracker{
		out:        out,
		mutex:      sync.Mutex{},
		trackedIDs: make(map[string]bool),
	}
}

// NewJSONMessageReporter creates a new ProgressReporter for JSONMessages
func NewJSONMessageReporter(tracker v1.ProgressTracker) *JSONMessageReporter {
	return &JSONMessageReporter{
		tracker:  tracker,
		tracking: map[string]chan v1.TrackerEntry{},
		mutex:    sync.Mutex{},
	}
}

// JSONMessageReporter implementation of a ProgressReporter that can be used with JSONMessage
type JSONMessageReporter struct {
	tracker  v1.ProgressTracker
	tracking map[string]chan v1.TrackerEntry
	mutex    sync.Mutex
}

// ActiveReporter returns true
func (j *JSONMessageReporter) ActiveReporter() bool { return true }

// StartReporting function that initializes the reporting of a particular Identifier
func (j *JSONMessageReporter) StartReporting(id string, _ int64) error {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	j.tracking[id] = make(chan v1.TrackerEntry)
	return j.tracker.StartTracking(id, j.tracking[id])
}

// Report stores the current state of a particular id
func (j *JSONMessageReporter) Report(id string, completed int64, total int64, err error) error {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	j.tracking[id] <- v1.TrackerEntry{
		Total:     total,
		Completed: completed,
		Error:     err,
	}
	return nil
}

// Finish wraps up the reporting for the id
func (j *JSONMessageReporter) Finish(id string, total int64) error {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	j.tracking[id] <- v1.TrackerEntry{
		Total:     total,
		Completed: total,
		Error:     nil,
	}
	close(j.tracking[id])
	delete(j.tracking, id)

	j.tracker.EndTracking(id)
	return nil
}

// channelReader converts the provided channel into a reader, this is the format that is needed by the tracker
func channelReader(ch <-chan JSONMessage) io.Reader {
	r, w := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer w.Close()

		for msg := range ch {
			out, err := json.Marshal(msg)
			if err != nil {
				return
			}
			_, err = w.Write(out)
			if err != nil {
				return
			}
		}
	}()

	return r
}

// JSONMessageTracker implements v1.ProgressTracker and uses the docker functionality to display JSON Messages
type JSONMessageTracker struct {
	out            io.Writer
	started        bool
	stopped        bool
	mutex          sync.Mutex
	messageChannel chan JSONMessage
	trackedIDs     map[string]bool
}

// StartDisplay initializes the tracker output that will write to the console
func (j *JSONMessageTracker) StartDisplay() {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.started {
		return
	}
	j.messageChannel = make(chan JSONMessage, 10)
	go DisplayJSONMessagesStream(channelReader(j.messageChannel), j.out, 1, true, nil)
}

// StartTracking initializes the tracking of a particular id
func (j *JSONMessageTracker) StartTracking(id string, updateChannel <-chan v1.TrackerEntry) error {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if _, ok := j.trackedIDs[id]; ok {
		return errors.New("Already tracked: " + id)
	}
	j.trackedIDs[id] = true

	go func() {
		for {
			select {
			case entry, ok := <-updateChannel:
				if !ok {
					return
				}
				var jsonError JSONError
				if entry.Error != nil {
					jsonError = JSONError{
						Code:    1,
						Message: entry.Error.Error(),
					}
				}
				msg := JSONMessage{
					Status: "Done",
					Progress: &JSONProgress{
						Current: entry.Completed,
						Total:   entry.Total,
					},
					ID:    id,
					Error: &jsonError,
				}

				if entry.Completed < entry.Total {
					msg.Status = "Downloading"
				}
				j.messageChannel <- msg
			case <-time.After(time.Second):
				if j.stopped {
					return
				}
			}
		}
	}()

	return nil
}

// EndTracking cleanup after a particular id is no longer tracked
func (j *JSONMessageTracker) EndTracking(id string) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	delete(j.trackedIDs, id)
}
