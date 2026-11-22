package playbook

type Severity string

const (
	SeverityHigh   Severity = "HIGH"
	SeverityMedium Severity = "MEDIUM"
	SeverityLow    Severity = "LOW"
)

type Rule struct {
	ID                   string   `json:"id"`
	Category             string   `json:"category"`
	Title                string   `json:"title"`
	Enabled              bool     `json:"enabled"`
	TargetPosition       string   `json:"target_position"`
	UnacceptableTriggers string   `json:"unacceptable_triggers"`
	StatuteReference     string   `json:"statute_reference"`
	Severity             Severity `json:"severity"`
}

type Playbook struct {
	Name         string `json:"name"`
	DocumentType string `json:"document_type"`
	Jurisdiction string `json:"jurisdiction"`
	Disclaimer   string `json:"disclaimer"`
	Rules        []Rule `json:"rules"`
}

type AuditFinding struct {
	RuleID           string   `json:"rule_id"`
	RuleTitle        string   `json:"rule_title"`
	Status           string   `json:"status"`
	Severity         Severity `json:"severity"`
	ExactQuote       string   `json:"exact_quote"`
	QuoteVerified    bool     `json:"quote_verified"`
	StatuteReference string   `json:"statute_reference"`
	PlainExplanation string   `json:"plain_explanation"`
	Redline          *Redline `json:"redline,omitempty"`
}

type Redline struct {
	ProposedClause  string `json:"proposed_clause"`
	NegotiationNote string `json:"negotiation_note"`
	ActionType      string `json:"action_type"`
}

type AuditReport struct {
	PlaybookName string         `json:"playbook_name"`
	ModelUsed    string         `json:"model_used"`
	TotalRules   int            `json:"total_rules"`
	FlaggedCount int            `json:"flagged_count"`
	MissingCount int            `json:"missing_count"`
	Findings     []AuditFinding `json:"findings"`
}
