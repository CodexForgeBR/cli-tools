// Package model provides AI model configuration and setup logic.
package model

// SetupCrossValidation configures cross-validation AI and model.
// If CrossAI is empty, uses the opposite of the primary AI.
// If CrossModel is empty, uses the default for the cross AI.
func SetupCrossValidation(ai string, crossAI string, crossModel string) (string, string) {
	if crossAI == "" {
		crossAI = OppositeAI(ai)
	}
	if crossModel == "" {
		crossModel = DefaultModelForAI(crossAI)
	}
	return crossAI, crossModel
}

// SetupFinalPlanValidation configures final plan validation.
// Defaults to cross-validation settings if not specified.
func SetupFinalPlanValidation(crossAI, crossModel, fpAI, fpModel string) (string, string) {
	if fpAI == "" {
		fpAI = crossAI
	}
	if fpModel == "" {
		fpModel = crossModel
	}
	return fpAI, fpModel
}

// SetupTasksValidation configures tasks validation.
// Defaults to implementation settings if not specified.
func SetupTasksValidation(implAI, implModel, tvAI, tvModel string) (string, string) {
	if tvAI == "" {
		tvAI = implAI
	}
	if tvModel == "" {
		tvModel = implModel
	}
	return tvAI, tvModel
}
