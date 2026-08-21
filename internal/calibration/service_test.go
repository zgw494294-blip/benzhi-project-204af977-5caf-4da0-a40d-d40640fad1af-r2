package calibration

import (
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestServiceNewIDIsUniqueUnderConcurrency(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)
	fixedNow := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixedNow }

	const count = 10000
	ids := make(chan string, count)
	var wait sync.WaitGroup
	wait.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wait.Done()
			ids <- service.newID("TEST")
		}()
	}
	wait.Wait()
	close(ids)

	seen := make(map[string]struct{}, count)
	for id := range ids {
		seen[id] = struct{}{}
	}
	if len(seen) != count {
		t.Fatalf("expected %s unique IDs, got %d", strconv.Itoa(count), len(seen))
	}
}

func TestNextCertificateNumberContinuesPersistedSequence(t *testing.T) {
	now := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	ledger := store.NewLedger()
	ledger.Certificates["sealed-session"] = domain.CalibrationCertificate{CertificateNo: "CAL-20260820-0007"}

	if got := nextCertificateNumber(&ledger, now); got != "CAL-20260820-0008" {
		t.Fatalf("expected persisted certificate sequence to continue at 0008, got %s", got)
	}
}

func TestServiceSupportsReworkAndSealing(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)
	session, samples, err := service.CreateSession(CreateSessionRequest{DeviceID: "AST-2", DeviceName: "校准光谱仪", ObservingBand: "可见光", Owner: "负责人", Samples: []SampleInput{{SampleNumber: "REF-1", ReferenceValue: 100, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"}, {SampleNumber: "REF-2", ReferenceValue: 20, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"}}})
	if err != nil || len(samples) != 2 {
		t.Fatalf("create session: %v", err)
	}
	session, first, err := service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[0].ID, MeasuredValue: 104, Operator: "值班员", IdempotencyKey: "m1"})
	if err != nil || first.WithinTolerance {
		t.Fatalf("first measurement: %v", err)
	}
	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[1].ID, MeasuredValue: 20, Operator: "值班员", IdempotencyKey: "m2"})
	if err != nil {
		t.Fatal(err)
	}
	session, review, err := service.SubmitReview(session.ID, ReviewInput{ExpectedVersion: session.Version, Reviewer: "质检员", Conclusion: "rework", ReworkReason: "首个样本超差"})
	if err != nil || review.Conclusion != "rework" || session.Status != domain.StatusRework {
		t.Fatalf("rework review: %v", err)
	}
	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[0].ID, MeasuredValue: 100.5, Operator: "值班员", IdempotencyKey: "m3"})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitReview(session.ID, ReviewInput{ExpectedVersion: session.Version, Reviewer: "质检员", Conclusion: "passed"})
	if err != nil || session.Status != domain.StatusReadyToSeal {
		t.Fatalf("passed review: %v", err)
	}
	session, certificate, err := service.SealSession(session.ID, SealInput{ExpectedVersion: session.Version, SealedBy: "负责人"})
	if err != nil || session.Status != domain.StatusSealed || len(certificate.SummaryHash) != 64 {
		t.Fatalf("seal: %v", err)
	}
	if _, _, err := service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[0].ID, MeasuredValue: 100, Operator: "值班员", IdempotencyKey: "after-seal"}); err == nil {
		t.Fatal("expected sealed session to reject measurement")
	}
}

func TestReworkAllowsEachFailedSampleToBeRetestedBeforeReview(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)
	session, samples, err := service.CreateSession(CreateSessionRequest{
		DeviceID:      "AST-MULTI-REWORK",
		DeviceName:    "多样本复测仪器",
		ObservingBand: "可见光",
		Owner:         "负责人",
		Samples: []SampleInput{
			{SampleNumber: "REF-1", ReferenceValue: 100, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"},
			{SampleNumber: "REF-2", ReferenceValue: 200, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{
		ExpectedVersion: session.Version,
		SampleID:        samples[0].ID,
		MeasuredValue:   104,
		Operator:        "值班员",
		IdempotencyKey:  "multi-rework-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{
		ExpectedVersion: session.Version,
		SampleID:        samples[1].ID,
		MeasuredValue:   204,
		Operator:        "值班员",
		IdempotencyKey:  "multi-rework-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitReview(session.ID, ReviewInput{
		ExpectedVersion: session.Version,
		Reviewer:        "质检员",
		Conclusion:      "rework",
		ReworkReason:    "两个样本均超差",
	})
	if err != nil {
		t.Fatal(err)
	}

	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{
		ExpectedVersion: session.Version,
		SampleID:        samples[0].ID,
		MeasuredValue:   100.5,
		Operator:        "值班员",
		IdempotencyKey:  "multi-rework-1-retest",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != domain.StatusRework {
		t.Fatalf("expected rework to continue while another sample is failed, got %s", session.Status)
	}

	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{
		ExpectedVersion: session.Version,
		SampleID:        samples[1].ID,
		MeasuredValue:   200.5,
		Operator:        "值班员",
		IdempotencyKey:  "multi-rework-2-retest",
	})
	if err != nil {
		t.Fatalf("expected remaining failed sample to be retestable: %v", err)
	}
	if session.Status != domain.StatusPendingReview {
		t.Fatalf("expected pending review after all failed samples were retested, got %s", session.Status)
	}
}

func TestMeasurementIdempotencyReturnsOriginalRecord(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)
	session, samples, err := service.CreateSession(CreateSessionRequest{DeviceID: "AST-3", DeviceName: "辐射计", ObservingBand: "红外", Owner: "负责人", Samples: []SampleInput{{SampleNumber: "REF-1", ReferenceValue: 2, Unit: "V", AllowedDelta: .1, RegisteredBy: "登记员"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, first, err := service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[0].ID, MeasuredValue: 2.01, Operator: "值班员", IdempotencyKey: "retry-1"})
	if err != nil {
		t.Fatal(err)
	}
	_, repeated, err := service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[0].ID, MeasuredValue: 2.01, Operator: "值班员", IdempotencyKey: "retry-1"})
	if err != nil || first.ID != repeated.ID {
		t.Fatalf("expected original record on retry, err=%v", err)
	}
}

func TestReworkCanRestartMeasurementWhenAllPreviousReadingsQualified(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)
	session, samples, err := service.CreateSession(CreateSessionRequest{DeviceID: "AST-REWORK", DeviceName: "复核重测仪器", ObservingBand: "可见光", Owner: "负责人", Samples: []SampleInput{{SampleNumber: "REF-1", ReferenceValue: 10, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"}}})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[0].ID, MeasuredValue: 10, Operator: "值班员", IdempotencyKey: "qualified-1"})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitReview(session.ID, ReviewInput{ExpectedVersion: session.Version, Reviewer: "质检员", Conclusion: "rework", ReworkReason: "需要复测确认稳定性"})
	if err != nil || session.Status != domain.StatusRework {
		t.Fatalf("expected rework state, err=%v status=%s", err, session.Status)
	}
	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[0].ID, MeasuredValue: 10.2, Operator: "值班员", IdempotencyKey: "qualified-retest"})
	if err != nil {
		t.Fatalf("rework measurement was blocked: %v", err)
	}
	if session.Status != domain.StatusPendingReview {
		t.Fatalf("expected pending review after rework measurement, got %s", session.Status)
	}
}
