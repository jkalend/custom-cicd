package laya

import (
	"encoding/json"
	"fmt"
	"strings"

	"laya-cicd-backend/internal/engine"
)

// Analyzer implements engine.DecisionAnalyzer on top of the Laya client.
type Analyzer struct {
	Client *Client
}

// NewAnalyzer builds the default analyzer.
func NewAnalyzer(c *Client) *Analyzer { return &Analyzer{Client: c} }

// --- question banks ------------------------------------------------------------

var failureKinds = map[string]string{
	"flaky":  "Test or transient network flakiness — passes on retry",
	"build":  "Compilation, packaging, or dependency build error",
	"config": "Wrong config, missing env var, bad pipeline definition",
	"infra":  "Runner, container, network, or disk infrastructure issue",
	"code":   "A real defect in the code under test",
}

var logActions = map[string]string{
	"ignore":      "Normal output, nothing to do",
	"investigate": "A human should look at this soon",
	"restart":     "Restart the failing service or step",
	"page":        "Someone must be paged right now",
}

var issueComponents = map[string]string{
	"ci":       "Pipeline or CI configuration itself",
	"backend":  "Backend service code",
	"frontend": "Frontend or UI code",
	"infra":    "Infrastructure, deployment, or environment",
	"docs":     "Documentation only",
	"unknown":  "Cannot be determined from the issue text",
}

var notificationLevels = map[string]string{
	"ignore": "Not worth any attention",
	"defer":  "Show in a digest later, no interruption",
	"notify": "Surface now, non-urgent",
	"urgent": "Interrupt the user immediately",
}

// --- engine.DecisionAnalyzer implementation ------------------------------------

// AnalyzeStepFailure classifies a failed step and triages its logs in two
// batched requests; returns the JSON blob stored on the step.
func (a *Analyzer) AnalyzeStepFailure(step engine.Step, runName string) json.RawMessage {
	state := map[string]any{
		"pipeline":        runName,
		"step":            step.Name,
		"command":         step.Command,
		"exit_error":      tailStr(step.Error, 2000),
		"output_tail":     tailStr(step.Output, 2000),
		"timeout_seconds": step.Timeout,
	}
	questions := map[string]QuestionSpec{
		"kind": {
			Type:           "choice",
			Instructions:   "What kind of failure is this?",
			ChoiceCriteria: failureKinds,
		},
		"retri": {
			Type:         "boolean",
			Instructions: "Is this failure worth retrying — i.e. is it plausibly transient rather than deterministic?",
		},
		"severity": {
			Type:         "score",
			Instructions: "How severe is this failure for the pipeline?",
			ScoreCriteria: []string{
				"trivial: cosmetic issue, pipeline goal still met",
				"minor: should be fixed but nothing is broken for users",
				"major: pipeline goal compromised",
				"critical: release-blocking or data/security risk",
			},
		},
	}
	answers, err := a.Client.Evaluate(state, questions, map[string]any{
		"module": "classify_failure", "step": step.Name,
	})
	if err != nil {
		return nil
	}

	// Log triage on the combined output — second batched request.
	triage := a.TriageLogs(step.Error+"\n"+step.Output, "step:"+step.Name)

	kind := answers["kind"].Choice
	retri := answers["retri"].Probability != nil && *answers["retri"].Probability >= 0.5
	if retriable := map[string]bool{"flaky": true, "infra": true}; retriable[kind] {
		retri = true
	}
	summary := fmt.Sprintf("%s failure (retry: %t)", kind, retri)

	blob, _ := json.Marshal(map[string]any{
		"classification": map[string]any{
			"kind":              kind,
			"retri":             retri,
			"retry":             retri,
			"retry_probability": answers["retri"].Probability,
			"severity":          answers["severity"].Score,
			"summary":           summary,
		},
		"log_triage": triage,
	})
	return blob
}

// TriageLogs classifies log text: severity, action, page decision.
func (a *Analyzer) TriageLogs(logText, source string) map[string]any {
	state := map[string]any{
		"source":   source,
		"log_tail": tailStr(logText, 4000),
	}
	questions := map[string]QuestionSpec{
		"severity": {
			Type:         "score",
			Instructions: "How severe are these logs?",
			ScoreCriteria: []string{
				"info: normal operational output",
				"warning: degraded but functional",
				"error: a failure occurred",
				"critical: service down or data at risk",
			},
		},
		"action": {
			Type:           "choice",
			Instructions:   "What should be done with this log output?",
			ChoiceCriteria: logActions,
		},
		"page_engineer": {
			Type:         "boolean",
			Instructions: "Does this log require paging an on-call engineer immediately?",
		},
	}
	answers, _ := a.Client.Evaluate(state, questions, map[string]any{
		"module": "triage_logs", "source": source,
	})

	chosen := answers["action"].Choice
	pageProb := 0.0
	if answers["page_engineer"].Probability != nil {
		pageProb = *answers["page_engineer"].Probability
	}
	// Laya advises; Go decides. A page needs both signals agreeing.
	if pageProb < 0.5 && chosen == "page" {
		chosen = "notify"
	}
	return map[string]any{
		"severity":         answers["severity"].Score,
		"action":           chosen,
		"page_probability": pageProb,
	}
}

// RouteIssue routes an issue/ticket: bug?, component, severity, labels.
func (a *Analyzer) RouteIssue(title, body string, labels []string) map[string]any {
	state := map[string]any{
		"title":           title,
		"body":            tailStr(body, 4000),
		"existing_labels": labels,
	}
	questions := map[string]QuestionSpec{
		"is_bug": {
			Type:         "boolean",
			Instructions: "Is this issue reporting a bug (as opposed to a feature request or question)?",
		},
		"component": {
			Type:           "choice",
			Instructions:   "Which component does this issue belong to?",
			ChoiceCriteria: issueComponents,
		},
		"severity": {
			Type:         "score",
			Instructions: "How severe is this issue?",
			ScoreCriteria: []string{
				"low: cosmetic or trivial",
				"medium: normal priority work",
				"high: important, should be fixed soon",
				"urgent: release or production blocking",
			},
		},
		"regression": {
			Type:         "boolean",
			Instructions: "Does this issue describe a regression — something that used to work?",
		},
	}
	answers, _ := a.Client.Evaluate(state, questions, map[string]any{
		"module": "route_issue", "title": title,
	})

	isBug := 0.0
	if answers["is_bug"].Probability != nil {
		isBug = *answers["is_bug"].Probability
	}
	regression := 0.0
	if answers["regression"].Probability != nil {
		regression = *answers["regression"].Probability
	}

	labelSet := map[string]bool{}
	for _, l := range labels {
		labelSet[l] = true
	}
	if isBug >= 0.5 {
		labelSet["bug"] = true
	}
	if regression >= 0.5 {
		labelSet["regression"] = true
	}
	severity := 1.0
	if answers["severity"].Score != nil {
		severity = *answers["severity"].Score
	}
	if severity >= 3 {
		labelSet["urgent"] = true
	} else if severity >= 2 {
		labelSet["high-priority"] = true
	}
	component := answers["component"].Choice
	if component != "unknown" && component != "" {
		labelSet["component:"+component] = true
	}
	out := make([]string, 0, len(labelSet))
	for l := range labelSet {
		out = append(out, l)
	}
	sortStrings(out)

	return map[string]any{
		"is_bug":    isBug,
		"component": component,
		"severity":  severity,
		"labels":    out,
	}
}

// NotifyRunCompletion decides how a finished run should be surfaced.
func (a *Analyzer) NotifyRunCompletion(run engine.PipelineRun) (string, float64) {
	failed := []string{}
	for _, s := range run.Steps {
		if s.Status == engine.StepFailed {
			failed = append(failed, s.Name)
		}
	}
	message := "all steps passed"
	if len(failed) > 0 {
		message = "failed steps: " + joinStrings(failed, ", ")
	}
	event := map[string]any{
		"type":       "run_" + string(run.Status),
		"title":      fmt.Sprintf("Pipeline '%s' %s", run.Name, run.Status),
		"message":    message,
		"pipeline":   run.Name,
		"run_status": string(run.Status),
	}
	questions := map[string]QuestionSpec{
		"level": {
			Type:           "choice",
			Instructions:   "How should this notification be delivered to the user?",
			ChoiceCriteria: notificationLevels,
		},
		"urgent": {
			Type:         "boolean",
			Instructions: "Is this urgent enough to interrupt the user right now?",
		},
	}
	answers, _ := a.Client.Evaluate(event, questions, map[string]any{
		"module":     "filter_notification",
		"event_type": event["type"],
	})

	level := answers["level"].Choice
	if level == "" {
		level = "defer"
	}
	urgent := 0.0
	if answers["urgent"].Probability != nil {
		urgent = *answers["urgent"].Probability
	}
	// Laya advises; Go decides: urgency needs both signals.
	if urgent < 0.5 && level == "urgent" {
		level = "notify"
	}
	return level, urgent
}

func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func tailStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[len(runes)-n:])
}

var fixCauses = map[string]string{
	"missing_dependency": "Required module, package, binary, or CLI tool is not installed",
	"syntax_error":       "Code syntax, compilation error, or invalid parameter syntax",
	"test_failure":       "Test assertion failure or unmet test precondition",
	"network_timeout":    "Remote repository, API, or package registry timeout/unreachable",
	"permission_denied":  "Insufficient file permissions or unauthorized access",
	"config_error":       "Missing or malformed environment variable or configuration file",
}

// SuggestFix analyzes a step failure and proposes a concrete diagnostic hypothesis and fix command.
func (a *Analyzer) SuggestFix(step engine.Step, runName string) map[string]any {
	combined := step.Error + "\n" + step.Output
	lower := strings.ToLower(combined)

	state := map[string]any{
		"pipeline": runName,
		"step":     step.Name,
		"command":  step.Command,
		"error":    tailStr(step.Error, 2000),
		"output":   tailStr(step.Output, 2000),
	}

	questions := map[string]QuestionSpec{
		"cause": {
			Type:           "choice",
			Instructions:   "What is the root cause of this failure?",
			ChoiceCriteria: fixCauses,
		},
		"automated_fix": {
			Type:         "boolean",
			Instructions: "Can this problem be resolved by running a fixup command or installing dependencies?",
		},
		"confidence": {
			Type:         "score",
			Instructions: "How confident is this diagnosis?",
			ScoreCriteria: []string{
				"low: speculation based on generic exit status",
				"medium: partial keywords match known issues",
				"high: explicit error message identifies the exact fault",
				"certain: unambiguous failure pattern",
			},
		},
	}

	answers, _ := a.Client.Evaluate(state, questions, map[string]any{
		"module": "suggest_fix", "step": step.Name,
	})

	cause := ""
	if ans, ok := answers["cause"]; ok {
		cause = ans.Choice
	}

	// Deterministic heuristic overrides / enrichments
	var hypothesis string
	var suggestedCmd string
	var canAutoApply bool

	switch {
	case strings.Contains(lower, "command not found") || strings.Contains(lower, "is not recognized as an internal or external command"):
		cause = "missing_dependency"
		hypothesis = "A required binary or tool executable is not found in PATH."
		suggestedCmd = "# Verify tool installation and ensure it is available in system PATH"
		canAutoApply = false

	case strings.Contains(lower, "cannot find module") || strings.Contains(lower, "module_not_found"):
		cause = "missing_dependency"
		hypothesis = "Node.js dependencies are missing or package lockfile is out of date."
		suggestedCmd = "npm install"
		canAutoApply = true

	case strings.Contains(lower, "no required module provides package") || strings.Contains(lower, "cannot find package"):
		cause = "missing_dependency"
		hypothesis = "Go dependencies are missing from go.mod/go.sum."
		suggestedCmd = "go mod tidy && go mod download"
		canAutoApply = true

	case strings.Contains(lower, "permission denied") || strings.Contains(lower, "access is denied"):
		cause = "permission_denied"
		hypothesis = "Execution failed due to missing file write or executable permissions."
		suggestedCmd = "chmod +x <target-file> # or grant process file permissions"
		canAutoApply = false

	case strings.Contains(lower, "econnrefused") || (strings.Contains(lower, "dial tcp") && strings.Contains(lower, "connect: connection refused")):
		cause = "network_timeout"
		hypothesis = "Target service or network endpoint is unreachable or not started."
		suggestedCmd = "# Check that required backing services are running before step execution"
		canAutoApply = false

	case strings.Contains(lower, "fail:") || strings.Contains(lower, "test failed") || strings.Contains(lower, "failed tests:"):
		cause = "test_failure"
		hypothesis = "Unit or integration tests failed an assertion."
		suggestedCmd = step.Command + " -v # Run failed test suite in verbose mode to inspect assertions"
		canAutoApply = false

	case strings.Contains(lower, "timed out") || strings.Contains(lower, "context deadline exceeded"):
		cause = "network_timeout"
		hypothesis = "Step exceeded its configured execution timeout."
		suggestedCmd = fmt.Sprintf("# Increase step timeout (currently %ds) or optimize long-running step commands", step.Timeout)
		canAutoApply = false

	default:
		if cause == "" {
			cause = "unknown"
			hypothesis = "Step failed with non-zero exit code. Review output logs for details."
			suggestedCmd = step.Command
		} else {
			hypothesis = fmt.Sprintf("Diagnosis points to %s.", fixCauses[cause])
			suggestedCmd = step.Command
		}
	}

	confidenceScore := 1.0
	if ans, ok := answers["confidence"]; ok && ans.Score != nil {
		confidenceScore = *ans.Score
	}

	provider := "heuristic"
	if ans, ok := answers["cause"]; ok && ans.Provider != "" {
		provider = ans.Provider
	}

	return map[string]any{
		"step_name":         step.Name,
		"cause":             cause,
		"hypothesis":        hypothesis,
		"suggested_command": suggestedCmd,
		"can_auto_apply":    canAutoApply,
		"confidence":        confidenceScore,
		"provider":          provider,
	}
}
