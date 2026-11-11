package ir

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLossy    Severity = "lossy"
	SeverityBlocking Severity = "blocking"
)

type FidelityIssue struct {
	Severity   Severity `json:"severity"`
	Code       string   `json:"code"`
	Path       string   `json:"path,omitempty"`
	Feature    string   `json:"feature,omitempty"`
	Suggestion string   `json:"suggestion,omitempty"`
}

type CapabilityReport struct {
	Mapped          []string `json:"mapped"`
	Dropped         []string `json:"dropped"`
	TargetSupported []string `json:"target_supported"`
}
