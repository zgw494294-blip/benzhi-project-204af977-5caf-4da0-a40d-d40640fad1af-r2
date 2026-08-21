package domain

type SessionProgress struct {
	SampleCount       int    `json:"sampleCount"`
	MeasuredCount     int    `json:"measuredCount"`
	QualifiedCount    int    `json:"qualifiedCount"`
	FailedCount       int    `json:"failedCount"`
	CompletionPercent int    `json:"completionPercent"`
	NextSampleID      string `json:"nextSampleID"`
	ReadyForReview    bool   `json:"readyForReview"`
	ReadyForSeal      bool   `json:"readyForSeal"`
}

func BuildProgress(samples []ReferenceSample, measurements []MeasurementRecord, reviews []QualityReview) SessionProgress {
	progress := SessionProgress{SampleCount: len(samples), NextSampleID: NextSampleID(samples, measurements)}
	for _, sample := range samples {
		latest, ok := LatestMeasurement(measurements, sample.ID)
		if !ok {
			continue
		}
		progress.MeasuredCount++
		if latest.WithinTolerance {
			progress.QualifiedCount++
		} else {
			progress.FailedCount++
		}
	}
	if progress.SampleCount > 0 {
		progress.CompletionPercent = progress.MeasuredCount * 100 / progress.SampleCount
	}
	progress.ReadyForReview = progress.SampleCount > 0 && progress.MeasuredCount == progress.SampleCount
	progress.ReadyForSeal = progress.ReadyForReview && progress.QualifiedCount == progress.SampleCount && hasPassedReview(reviews)
	return progress
}

func hasPassedReview(reviews []QualityReview) bool {
	for index := len(reviews) - 1; index >= 0; index-- {
		if reviews[index].Conclusion == "passed" {
			return true
		}
		if reviews[index].Conclusion == "rework" {
			return false
		}
	}
	return false
}
