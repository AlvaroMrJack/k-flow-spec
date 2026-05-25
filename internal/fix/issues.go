package fix

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type IssueType string

const (
	IssueNodeRenamed        IssueType = "node_renamed"
	IssueBranchRemoved      IssueType = "branch_removed"
	IssueVariableRenamed    IssueType = "variable_renamed"
	IssuePlaceholderMessage IssueType = "placeholder_message"
	IssueSnapshotOutdated   IssueType = "snapshot_outdated"
	IssuePathIncomplete     IssueType = "path_incomplete"
	IssueInvalidYAML        IssueType = "invalid_yaml"
)

type Issue struct {
	Type     IssueType  `json:"type"`
	Severity Severity   `json:"severity"`
	SpecFile string     `json:"spec_file"`
	Line     int        `json:"line,omitempty"`
	Message  string     `json:"message"`
	AutoFix  bool       `json:"auto_fix"`
	Fix      *Fix       `json:"fix,omitempty"`
}

type Fix struct {
	Description string `json:"description"`
	OldValue    string `json:"old_value,omitempty"`
	NewValue    string `json:"new_value,omitempty"`
	File        string `json:"file"`
}
