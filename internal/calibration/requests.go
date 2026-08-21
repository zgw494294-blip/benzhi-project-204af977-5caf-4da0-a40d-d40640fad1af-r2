package calibration

type SampleInput struct {
	SampleNumber   string  `json:"sampleNumber"`
	ReferenceValue float64 `json:"referenceValue"`
	Unit           string  `json:"unit"`
	AllowedDelta   float64 `json:"allowedDelta"`
	RegisteredBy   string  `json:"registeredBy"`
}

type CreateSessionRequest struct {
	DeviceID      string        `json:"deviceID"`
	DeviceName    string        `json:"deviceName"`
	ObservingBand string        `json:"observingBand"`
	Owner         string        `json:"owner"`
	Samples       []SampleInput `json:"samples"`
}

type RegisterSamplesRequest struct {
	ExpectedVersion int64         `json:"expectedVersion"`
	Samples         []SampleInput `json:"samples"`
}

type MeasurementInput struct {
	ExpectedVersion int64   `json:"expectedVersion"`
	SampleID        string  `json:"sampleID"`
	MeasuredValue   float64 `json:"measuredValue"`
	Operator        string  `json:"operator"`
	Note            string  `json:"note"`
	IdempotencyKey  string  `json:"idempotencyKey"`
}

type ReviewInput struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	Reviewer        string `json:"reviewer"`
	Conclusion      string `json:"conclusion"`
	ReworkReason    string `json:"reworkReason"`
}

type SealInput struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	SealedBy        string `json:"sealedBy"`
}
