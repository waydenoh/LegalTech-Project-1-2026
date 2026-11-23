package exporter

import (
	"fmt"
	"strings"

	"github.com/yourusername/sg-playbook-reviewer/internal/playbook"
)

func ToMarkdown(pb playbook.Playbook, report playbook.AuditReport) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Playbook Contract Review: %s\n\n", report.PlaybookName))
	b.WriteString(fmt.Sprintf("> **Disclaimer:** %s\n\n", pb.Disclaimer))
	b.WriteString(fmt.Sprintf("- **Model Used:** %s\n", report.ModelUsed))
	b.WriteString(fmt.Sprintf("- **Active Rules Evaluated:** %d\n", report.TotalRules))
	b.WriteString(fmt.Sprintf("- **Flagged Clauses:** %d\n", report.FlaggedCount))
	b.WriteString(fmt.Sprintf("- **Missing Clauses:** %d\n\n", report.MissingCount))

	for _, f := range report.Findings {
		b.WriteString(fmt.Sprintf("## [%s] %s (%s) - %s\n\n", f.Severity, f.RuleTitle, f.RuleID, f.Status))
		b.WriteString(fmt.Sprintf("- **Statute / Guideline:** %s\n", f.StatuteReference))
		b.WriteString(fmt.Sprintf("- **Explanation:** %s\n", f.PlainExplanation))
		if f.ExactQuote != "" {
			b.WriteString(fmt.Sprintf("- **Quoted Text (Verified: %v):** %q\n", f.QuoteVerified, f.ExactQuote))
		}
		if f.Redline != nil {
			b.WriteString(fmt.Sprintf("\n### Proposed Redline (%s)\n\n%s\n\n", f.Redline.ActionType, f.Redline.ProposedClause))
			b.WriteString(fmt.Sprintf("**Negotiation Note:** %s\n\n", f.Redline.NegotiationNote))
		}
		b.WriteString("---\n\n")
	}

	return b.String()
}
