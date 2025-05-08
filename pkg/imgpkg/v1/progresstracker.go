// Copyright 2025 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1

// TrackerEntry information needed for tracking
type TrackerEntry struct {
	Total     int64
	Completed int64
	Error     error
}

// ProgressTracker Displays and keeps a list of all the id's that need to be tracked
type ProgressTracker interface {
	StartDisplay()
	StartTracking(id string, updateChannel <-chan TrackerEntry) error
	EndTracking(id string)
}

// ProgressReporter Used to report all the activity for a particular id
type ProgressReporter interface {
	StartReporting(id string, total int64) error
	Report(id string, completed int64, total int64, err error) error
	Finish(id string, total int64) error
	ActiveReporter() bool
}

// NoopProgressReporter does no reporting, is used by default
type NoopProgressReporter struct{}

// ActiveReporter will return false given this report does not actively keep any record
func (n NoopProgressReporter) ActiveReporter() bool { return false }

// StartReporting does nothing
func (n NoopProgressReporter) StartReporting(_ string, _ int64) error { return nil }

// Report does nothing
func (n NoopProgressReporter) Report(_ string, _ int64, _ int64, _ error) error {
	return nil
}

// Finish does nothing
func (n NoopProgressReporter) Finish(_ string, _ int64) error { return nil }
