package server

import (
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/gin-gonic/gin"
)

// ValuationPreviewRequest is the request body for previewing a formula
type ValuationPreviewRequest struct {
	Formula  string  `json:"formula"`
	Amount   float64 `json:"amount"`
	DaysHeld float64 `json:"days_held"`
	Note     string  `json:"note"`
}

// ValuationValidationResult represents the validation result for a single valuation
type ValuationValidationResult struct {
	Name    string `json:"name"`
	Account string `json:"account"`
	Formula string `json:"formula"`
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
}

// ValidateValuations validates all custom valuations in config
func ValidateValuations() gin.H {
	valuations := config.GetCustomValuations()
	results := make([]ValuationValidationResult, 0, len(valuations))

	allValid := true
	for _, v := range valuations {
		result := ValuationValidationResult{
			Name:    v.Name,
			Account: v.Account,
			Formula: v.Formula,
			Valid:   true,
		}

		if err := service.ValidateFormula(v.Formula); err != nil {
			result.Valid = false
			result.Error = err.Error()
			allValid = false
		}

		results = append(results, result)
	}

	return gin.H{
		"valid":   allValid,
		"results": results,
	}
}

// PreviewValuation previews a formula with sample data
func PreviewValuation(request ValuationPreviewRequest) gin.H {
	// Set defaults
	if request.Amount == 0 {
		request.Amount = 10000
	}
	if request.DaysHeld == 0 {
		request.DaysHeld = 30
	}

	preview := service.PreviewFormula(
		request.Formula,
		request.Amount,
		request.DaysHeld,
		request.Note,
	)

	return gin.H{
		"preview": preview,
	}
}

