package prompt

import _ "embed"

// Template files embedded at compile time
var (
	//go:embed templates/impl-first.txt
	ImplFirstTemplate string

	//go:embed templates/impl-continue.txt
	ImplContinueTemplate string

	//go:embed templates/inadmissible-rules.txt
	InadmissibleRules string

	//go:embed templates/evidence-rules.txt
	EvidenceRules string

	//go:embed templates/playwright-rules.txt
	PlaywrightRules string

	//go:embed templates/learnings-section.txt
	LearningsSection string

	//go:embed templates/learnings-output.txt
	LearningsOutput string

	//go:embed templates/validation.txt
	ValidationTemplate string

	//go:embed templates/cross-validation.txt
	CrossValidationTemplate string

	//go:embed templates/tasks-validation.txt
	TasksValidationTemplate string

	//go:embed templates/final-plan.txt
	FinalPlanTemplate string
)
