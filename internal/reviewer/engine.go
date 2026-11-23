package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/yourusername/sg-playbook-reviewer/internal/llm"
	"github.com/yourusername/sg-playbook-reviewer/internal/playbook"
)

type Engine struct {
	Client      *llm.OllamaClient
	Concurrency int
}

func NewEngine(client *llm.OllamaClient, concurrency int) *Engine {
	if concurrency <= 0 {
		concurrency = 2
	}
	return &Engine{
		Client:      client,
		Concurrency: concurrency,
	}
}

type auditPassOutput struct {
	Findings []playbook.AuditFinding `json:"findings"`
}

var auditSchema = []byte(`{
  "type": "object",
  "properties": {
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "rule_id": { "type": "string" },
          "rule_title": { "type": "string" },
          "status": { "type": "string", "enum": ["PASS", "FLAGGED", "MISSING_CLAUSE"] },
          "severity": { "type": "string", "enum": ["HIGH", "MEDIUM", "LOW"] },
          "exact_quote": { "type": "string" },
          "statute_reference": { "type": "string" },
          "plain_explanation": { "type": "string" }
        },
        "required": ["rule_id", "rule_title", "status", "severity", "exact_quote", "statute_reference", "plain_explanation"]
      }
    }
  },
  "required": ["findings"]
}`)

var redlineSchema = []byte(`{
  "type": "object",
  "properties": {
    "proposed_clause": { "type": "string" },
    "negotiation_note": { "type": "string" },
    "action_type": { "type": "string", "enum": ["REPLACE", "INSERT"] }
  },
  "required": ["proposed_clause", "negotiation_note", "action_type"]
}`)

func (e *Engine) ReviewContract(ctx context.Context, pb playbook.Playbook, contractText string, generateRedlines bool) (*playbook.AuditReport, error) {
	activeRules := make([]playbook.Rule, 0, len(pb.Rules))
	for _, r := range pb.Rules {
		if r.Enabled {
			activeRules = append(activeRules, r)
		}
	}
	if len(activeRules) == 0 {
		return nil, fmt.Errorf("no active playbook rules enabled; enable at least one rule before running a review")
	}

	rulesJSON, err := json.MarshalIndent(activeRules, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal active rules: %w", err)
	}

	systemPrompt := "You are a Singapore contract playbook compliance assistant. Evaluate the provided contract against every active playbook rule. Set status to PASS if compliant, FLAGGED if it violates target_position or triggers unacceptable_triggers (include an exact verbatim substring from the contract in exact_quote), or MISSING_CLAUSE if required terms are absent (set exact_quote to empty string). Keep plain_explanation concise (1-2 sentences)."

	userPrompt := fmt.Sprintf("ACTIVE PLAYBOOK RULES:\n%s\n\nCONTRACT TEXT:\n%s", string(rulesJSON), contractText)

	var pass1 auditPassOutput
	if err := e.Client.GenerateStructured(ctx, systemPrompt, userPrompt, auditSchema, &pass1); err != nil {
		return nil, fmt.Errorf("pass 1 audit failed: %w", err)
	}

	flaggedCount := 0
	missingCount := 0
	for i := range pass1.Findings {
		f := &pass1.Findings[i]
		if f.ExactQuote != "" && strings.Contains(contractText, f.ExactQuote) {
			f.QuoteVerified = true
		}
		switch f.Status {
		case "FLAGGED":
			flaggedCount++
		case "MISSING_CLAUSE":
			missingCount++
		}
	}

	if generateRedlines {
		ruleMap := make(map[string]playbook.Rule, len(activeRules))
		for _, r := range activeRules {
			ruleMap[r.ID] = r
		}

		sem := make(chan struct{}, e.Concurrency)
		var wg sync.WaitGroup

		for i := range pass1.Findings {
			finding := &pass1.Findings[i]
			if finding.Status != "FLAGGED" && finding.Status != "MISSING_CLAUSE" {
				continue
			}
			wg.Add(1)
			go func(f *playbook.AuditFinding) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				rule := ruleMap[f.RuleID]
				rl, err := e.draftSingleRedline(ctx, pb.Jurisdiction, rule, *f, contractText)
				if err == nil {
					f.Redline = rl
				}
			}(finding)
		}
		wg.Wait()
	}

	return &playbook.AuditReport{
		PlaybookName: pb.Name,
		ModelUsed:    e.Client.Model,
		TotalRules:   len(activeRules),
		FlaggedCount: flaggedCount,
		MissingCount: missingCount,
		Findings:     pass1.Findings,
	}, nil
}

func (e *Engine) draftSingleRedline(ctx context.Context, jurisdiction string, rule playbook.Rule, finding playbook.AuditFinding, contractText string) (*playbook.Redline, error) {
	sysPrompt := fmt.Sprintf("You draft practical, balanced fallback clauses for %s contracts aligned with the playbook target_position.", jurisdiction)
	actionHint := "REPLACE"
	if finding.Status == "MISSING_CLAUSE" {
		actionHint = "INSERT"
	}
	userPrompt := fmt.Sprintf(
		"Rule: %s (%s)\nTarget Position: %s\nStatute Reference: %s\nFinding Status: %s\nFinding Explanation: %s\nExact Quote: %s\nAction Hint: %s\n\nContract Context:\n%s",
		rule.Title, rule.Category, rule.TargetPosition, rule.StatuteReference,
		finding.Status, finding.PlainExplanation, finding.ExactQuote, actionHint, contractText,
	)

	var rl playbook.Redline
	if err := e.Client.GenerateStructured(ctx, sysPrompt, userPrompt, redlineSchema, &rl); err != nil {
		return nil, err
	}
	return &rl, nil
}
