package reviewsummaryhistory

import (
	"path/filepath"
	"testing"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestPassedReviewSummarizesLatestMeasurementsOnly(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := calibration.NewService(repository)
	session, samples, err := service.CreateSession(calibration.CreateSessionRequest{
		DeviceID:      "AST-REWORK-SUMMARY",
		DeviceName:    "复测摘要仪器",
		ObservingBand: "可见光",
		Owner:         "负责人",
		Samples: []calibration.SampleInput{{
			SampleNumber: "REF-1", ReferenceValue: 100, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitMeasurement(session.ID, calibration.MeasurementInput{
		ExpectedVersion: session.Version,
		SampleID:        samples[0].ID,
		MeasuredValue:   104,
		Operator:        "值班员",
		IdempotencyKey:  "summary-first",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitReview(session.ID, calibration.ReviewInput{
		ExpectedVersion: session.Version,
		Reviewer:        "质检员",
		Conclusion:      "rework",
		ReworkReason:    "首测超差，要求复测",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitMeasurement(session.ID, calibration.MeasurementInput{
		ExpectedVersion: session.Version,
		SampleID:        samples[0].ID,
		MeasuredValue:   100.5,
		Operator:        "值班员",
		IdempotencyKey:  "summary-retest",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, review, err := service.SubmitReview(session.ID, calibration.ReviewInput{
		ExpectedVersion: session.Version,
		Reviewer:        "质检员",
		Conclusion:      "passed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if review.DeviationSummary != "全部样本均在允许偏差内" {
		t.Fatalf("通过复核仍包含历史超差摘要: %s", review.DeviationSummary)
	}
}
