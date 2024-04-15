package scanningResultsParser

import (
	"fmt"
	"github.com/tidwall/gjson"
	"time"
)

type Severity string

const (
	LOW      Severity = "Low"
	MEDIUM   Severity = "Medium"
	HIGH     Severity = "High"
	CRITICAL Severity = "Critical"
	UNKNOWN  Severity = "Unknown"
)

type Summary struct {
	ScannedOn  time.Time
	Severities map[Severity]int
}

func (summary Summary) String() string {
	return fmt.Sprintf("%d Critical, %d High, %d Medium, %d Low, %d Unknown", summary.Severities[CRITICAL], summary.Severities[HIGH], summary.Severities[MEDIUM], summary.Severities[LOW], summary.Severities[UNKNOWN])
}

type Licenses struct {
	Summary  Summary
	Licenses []License
}

type License struct {
	Classification string   // Category
	Severity       Severity // Severity
	License        string   // Name
	Package        string   // PkgName
	Source         string   // FilePath
}

func getLicense(licenseJson string) License {
	return License{
		Classification: gjson.Get(licenseJson, "Category").String(),
		Severity:       Severity(gjson.Get(licenseJson, "Severity").String()),
		License:        gjson.Get(licenseJson, "Name").String(),
		Package:        gjson.Get(licenseJson, "PkgName").String(),
		Source:         gjson.Get(licenseJson, "FilePath").String(),
	}
}

type Vulnerabilities struct {
	Summary         Summary
	Vulnerabilities []Vulnerability
}

type Vulnerability struct {
	CVEId          string   // VulnerabilityID
	Severity       Severity // Severity
	Package        string   // PkgName
	CurrentVersion string   // InstalledVersion
	FixedInVersion string   // FixedVersion
}

type MisConfigurationSummary struct {
	Success    int64
	Fail       int64
	Exceptions int64
}

func (summary MisConfigurationSummary) String() string {
	return fmt.Sprintf("%d Successes, %d Failures, %d Exceptions", summary.Success, summary.Fail, summary.Exceptions)
}

type Line struct {
	Number    int64  // Number
	Content   string // Content
	IsCause   bool   // IsCause
	Truncated bool   // Truncated
}

type CauseMetadata struct {
	StartLine int64  // StartLine
	EndLine   int64  // EndLine
	Lines     []Line // Code.Lines
}

type Configuration struct {
	Id            string        // ID
	Title         string        // Title
	Message       string        // Message
	Resolution    string        // Resolution
	Status        string        // Status
	Severity      Severity      // Severity
	CauseMetadata CauseMetadata // CauseMetadata
}

type MisConfiguration struct {
	FilePath       string                  // Target
	Type           string                  // Type
	MisConfSummary MisConfigurationSummary // MisConfSummary
	Summary        Summary
	Configurations []Configuration
}

type Secret struct {
	Severity Severity
	CauseMetadata
}

type ExposedSecret struct {
	FilePath string // target and class: secret
	Summary  Summary
	Secrets  []Secret
}

type MisConfigurations struct {
	Summary           Summary            `json:"summary"`
	MisConfigurations []MisConfiguration `json:"misConfigurations"`
}

type ExposedSecrets struct {
	Summary        Summary         `json:"summary"`
	ExposedSecrets []ExposedSecret `json:"exposedSecrets"`
}

type ImageScanResult struct {
	Summary       Summary         `json:"summary"`
	Image         string          `json:"image"`
	Vulnerability Vulnerabilities `json:"vulnerability"`
	License       Licenses        `json:"license"`
	Status        string          `json:"status"`
	StartedOn     time.Time       `json:"StartedOn"`
}

type CodeScanResult struct {
	Vulnerability     Vulnerabilities   `json:"vulnerability"`
	License           Licenses          `json:"license"`
	MisConfigurations MisConfigurations `json:"misConfigurations"`
	ExposedSecrets    ExposedSecrets    `json:"exposedSecrets"`
}

type ManifestScanResult struct {
	MisConfigurations MisConfigurations `json:"misConfigurations"`
}
