package progress_cache_stale

import (
	"path/filepath"
	"testing"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestProgressReflectsCommittedMeasurementsAfterCachedRead(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := calibration.NewService(repository)
	session, samples, err := service.CreateSession(calibration.CreateSessionRequest{
		DeviceID:      "AST-PROGRESS-CACHE",
		DeviceName:    "进度缓存复现仪",
		ObservingBand: "可见光",
		Owner:         "工程师",
		Samples: []calibration.SampleInput{
			{SampleNumber: "REF-1", ReferenceValue: 100, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"},
			{SampleNumber: "REF-2", ReferenceValue: 200, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := service.GetProgress(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if initial.MeasuredCount != 0 || initial.NextSampleID != samples[0].ID {
		t.Fatalf("unexpected initial progress: %#v", initial)
	}
	updatedSession, _, err := service.SubmitMeasurement(session.ID, calibration.MeasurementInput{
		ExpectedVersion: session.Version,
		SampleID:        samples[0].ID,
		MeasuredValue:   100,
		Operator:        "值班员",
		IdempotencyKey:  "progress-cache-measurement",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.GetProgress(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.MeasuredCount != 1 || updated.CompletionPercent != 50 || updated.NextSampleID != samples[1].ID || updatedSession.Version != 2 {
		t.Fatalf("TestProgressReflectsCommittedMeasurementsAfterCachedRead: progress did not reflect committed revision: session=%#v progress=%#v", updatedSession, updated)
	}
}
