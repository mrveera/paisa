package service

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/expr-lang/expr"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

// ValuationContext provides the variables available in valuation expressions
type ValuationContext struct {
	// Amount is the posting amount in default currency
	Amount float64 `expr:"amount"`
	// Quantity is the number of units
	Quantity float64 `expr:"quantity"`
	// Date is the posting date as a time.Time
	Date time.Time `expr:"date"`
	// DaysHeld is the number of days since the posting date
	DaysHeld float64 `expr:"days_held"`
	// MonthsHeld is the number of months since the posting date
	MonthsHeld float64 `expr:"months_held"`
	// YearsHeld is the number of years since the posting date
	YearsHeld float64 `expr:"years_held"`
	// Note is the transaction note
	Note string `expr:"note"`
	// Account is the account name
	Account string `expr:"account"`
	// Commodity is the commodity name
	Commodity string `expr:"commodity"`
}

// Custom functions available in expressions
var exprFunctions = []expr.Option{
	// ============ Note Parsing Functions ============

	// parse_note_float extracts a float value from note after a prefix
	// e.g., parse_note_float("Int:12.5 Per:M", "Int:") returns 12.5
	expr.Function(
		"parse_note_float",
		func(params ...any) (any, error) {
			note := params[0].(string)
			prefix := params[1].(string)
			return parseNoteFloat(note, prefix), nil
		},
		new(func(string, string) float64),
	),

	// parse_note_string extracts a string value from note after a prefix
	// e.g., parse_note_string("Int:12.5 Per:M", "Per:") returns "M"
	expr.Function(
		"parse_note_string",
		func(params ...any) (any, error) {
			note := params[0].(string)
			prefix := params[1].(string)
			return parseNoteString(note, prefix), nil
		},
		new(func(string, string) string),
	),

	// note_contains checks if note contains a substring
	expr.Function(
		"note_contains",
		func(params ...any) (any, error) {
			note := params[0].(string)
			substr := params[1].(string)
			return strings.Contains(note, substr), nil
		},
		new(func(string, string) bool),
	),

	// ============ Interest Calculation Functions ============

	// simple_interest calculates simple interest: principal * rate * time
	// simple_interest(principal, annual_rate_percent, days) -> interest amount
	expr.Function(
		"simple_interest",
		func(params ...any) (any, error) {
			principal := toFloat64(params[0])
			annualRate := toFloat64(params[1])
			days := toFloat64(params[2])
			return principal * (annualRate / 100) * (days / 365), nil
		},
		new(func(float64, float64, float64) float64),
	),

	// compound_interest calculates compound interest
	// compound_interest(principal, annual_rate_percent, days, compounds_per_year) -> total value
	expr.Function(
		"compound_interest",
		func(params ...any) (any, error) {
			principal := toFloat64(params[0])
			annualRate := toFloat64(params[1])
			days := toFloat64(params[2])
			compoundsPerYear := toFloat64(params[3])
			if compoundsPerYear == 0 {
				compoundsPerYear = 12 // default to monthly
			}
			years := days / 365
			// A = P(1 + r/n)^(nt)
			return principal * math.Pow(1+annualRate/100/compoundsPerYear, compoundsPerYear*years), nil
		},
		new(func(float64, float64, float64, float64) float64),
	),

	// monthly_interest calculates interest with monthly rate
	// monthly_interest(principal, monthly_rate_percent, days) -> interest amount
	expr.Function(
		"monthly_interest",
		func(params ...any) (any, error) {
			principal := toFloat64(params[0])
			monthlyRate := toFloat64(params[1])
			days := toFloat64(params[2])
			return principal * (monthlyRate / 100) * (days / 30), nil
		},
		new(func(float64, float64, float64) float64),
	),

	// daily_interest calculates interest with daily rate
	// daily_interest(principal, daily_rate_percent, days) -> interest amount
	expr.Function(
		"daily_interest",
		func(params ...any) (any, error) {
			principal := toFloat64(params[0])
			dailyRate := toFloat64(params[1])
			days := toFloat64(params[2])
			return principal * (dailyRate / 100) * days, nil
		},
		new(func(float64, float64, float64) float64),
	),

	// ============ Math Functions ============

	// min returns the minimum of two numbers
	expr.Function(
		"min",
		func(params ...any) (any, error) {
			a := toFloat64(params[0])
			b := toFloat64(params[1])
			return math.Min(a, b), nil
		},
		new(func(float64, float64) float64),
	),

	// max returns the maximum of two numbers
	expr.Function(
		"max",
		func(params ...any) (any, error) {
			a := toFloat64(params[0])
			b := toFloat64(params[1])
			return math.Max(a, b), nil
		},
		new(func(float64, float64) float64),
	),

	// round rounds to the nearest integer
	expr.Function(
		"round",
		func(params ...any) (any, error) {
			return math.Round(toFloat64(params[0])), nil
		},
		new(func(float64) float64),
	),

	// floor rounds down to the nearest integer
	expr.Function(
		"floor",
		func(params ...any) (any, error) {
			return math.Floor(toFloat64(params[0])), nil
		},
		new(func(float64) float64),
	),

	// ceil rounds up to the nearest integer
	expr.Function(
		"ceil",
		func(params ...any) (any, error) {
			return math.Ceil(toFloat64(params[0])), nil
		},
		new(func(float64) float64),
	),

	// abs returns the absolute value
	expr.Function(
		"abs",
		func(params ...any) (any, error) {
			return math.Abs(toFloat64(params[0])), nil
		},
		new(func(float64) float64),
	),

	// pow raises base to the power of exponent
	expr.Function(
		"pow",
		func(params ...any) (any, error) {
			base := toFloat64(params[0])
			exp := toFloat64(params[1])
			return math.Pow(base, exp), nil
		},
		new(func(float64, float64) float64),
	),

	// sqrt returns the square root
	expr.Function(
		"sqrt",
		func(params ...any) (any, error) {
			return math.Sqrt(toFloat64(params[0])), nil
		},
		new(func(float64) float64),
	),

	// ============ Conditional Functions ============

	// if_then_else returns trueVal if condition is true, else falseVal
	expr.Function(
		"if_else",
		func(params ...any) (any, error) {
			condition := params[0].(bool)
			trueVal := toFloat64(params[1])
			falseVal := toFloat64(params[2])
			if condition {
				return trueVal, nil
			}
			return falseVal, nil
		},
		new(func(bool, float64, float64) float64),
	),

	// clamp restricts a value to be within a range
	expr.Function(
		"clamp",
		func(params ...any) (any, error) {
			value := toFloat64(params[0])
			minVal := toFloat64(params[1])
			maxVal := toFloat64(params[2])
			return math.Max(minVal, math.Min(maxVal, value)), nil
		},
		new(func(float64, float64, float64) float64),
	),
}

// toFloat64 converts various numeric types to float64
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	default:
		return 0
	}
}

// parseNoteFloat extracts a float value from note after a given prefix
func parseNoteFloat(note, prefix string) float64 {
	if !strings.Contains(note, prefix) {
		return 0
	}
	parts := strings.Split(note, prefix)
	if len(parts) < 2 {
		return 0
	}
	valueStr := strings.Split(parts[1], " ")[0]
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}
	return value
}

// parseNoteString extracts a string value from note after a given prefix
func parseNoteString(note, prefix string) string {
	if !strings.Contains(note, prefix) {
		return ""
	}
	parts := strings.Split(note, prefix)
	if len(parts) < 2 {
		return ""
	}
	return strings.Split(parts[1], " ")[0]
}

// matchAccountPattern checks if an account matches a pattern (supports * wildcard)
func matchAccountPattern(account, pattern string) bool {
	// Convert wildcard pattern to regex
	// Assets:p2p:* -> ^Assets:p2p:.*$
	regexPattern := "^" + regexp.QuoteMeta(pattern) + "$"
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, `.*`)

	matched, err := regexp.MatchString(regexPattern, account)
	if err != nil {
		log.Warnf("Invalid account pattern %s: %v", pattern, err)
		return false
	}
	return matched
}

// FindCustomValuation finds a matching custom valuation rule for a posting
func FindCustomValuation(p posting.Posting) *config.CustomValuation {
	valuations := config.GetCustomValuations()

	for _, v := range valuations {
		// Check account pattern
		if !matchAccountPattern(p.Account, v.Account) {
			continue
		}

		// Check note_contains if specified
		if v.NoteContains != "" && !strings.Contains(p.TransactionNote, v.NoteContains) {
			continue
		}

		return &v
	}

	return nil
}

// EvaluateValuation evaluates a custom valuation formula for a posting
func EvaluateValuation(valuation *config.CustomValuation, p posting.Posting, evaluationDate time.Time) (decimal.Decimal, error) {
	ctx := buildValuationContext(p, evaluationDate)

	// Compile and run expression
	options := append([]expr.Option{expr.Env(ctx)}, exprFunctions...)
	program, err := expr.Compile(valuation.Formula, options...)
	if err != nil {
		log.Warnf("Failed to compile valuation formula '%s': %v", valuation.Formula, err)
		return p.Amount, err
	}

	result, err := expr.Run(program, ctx)
	if err != nil {
		log.Warnf("Failed to evaluate valuation formula '%s': %v", valuation.Formula, err)
		return p.Amount, err
	}

	// Convert result to decimal
	switch v := result.(type) {
	case float64:
		return decimal.NewFromFloat(v), nil
	case int:
		return decimal.NewFromInt(int64(v)), nil
	case int64:
		return decimal.NewFromInt(v), nil
	default:
		log.Warnf("Valuation formula returned unexpected type %T", result)
		return p.Amount, nil
	}
}

// buildValuationContext creates a ValuationContext from a posting
func buildValuationContext(p posting.Posting, evaluationDate time.Time) ValuationContext {
	daysHeld := evaluationDate.Sub(p.Date).Hours() / 24
	monthsHeld := daysHeld / 30.44 // average days per month
	yearsHeld := daysHeld / 365.25 // account for leap years

	return ValuationContext{
		Amount:     p.Amount.InexactFloat64(),
		Quantity:   p.Quantity.InexactFloat64(),
		Date:       p.Date,
		DaysHeld:   daysHeld,
		MonthsHeld: monthsHeld,
		YearsHeld:  yearsHeld,
		Note:       p.TransactionNote,
		Account:    p.Account,
		Commodity:  p.Commodity,
	}
}

// ValidateFormula validates a valuation formula without running it
// Returns nil if valid, or an error describing the problem
func ValidateFormula(formula string) error {
	// Create a dummy context for validation
	ctx := ValuationContext{
		Amount:     10000,
		Quantity:   1,
		Date:       time.Now(),
		DaysHeld:   30,
		MonthsHeld: 1,
		YearsHeld:  0.0822,
		Note:       "sample note Int:12 Per:M",
		Account:    "Assets:Test",
		Commodity:  "INR",
	}

	options := append([]expr.Option{expr.Env(ctx)}, exprFunctions...)
	program, err := expr.Compile(formula, options...)
	if err != nil {
		return fmt.Errorf("syntax error: %w", err)
	}

	// Try to run it with sample data
	result, err := expr.Run(program, ctx)
	if err != nil {
		return fmt.Errorf("evaluation error: %w", err)
	}

	// Check return type
	switch result.(type) {
	case float64, int, int64, int32:
		return nil
	default:
		return fmt.Errorf("formula must return a number, got %T", result)
	}
}

// ValidateAllValuations validates all custom valuations in config
// Returns a map of valuation name to error (empty map if all valid)
func ValidateAllValuations() map[string]error {
	errors := make(map[string]error)
	valuations := config.GetCustomValuations()

	for _, v := range valuations {
		if err := ValidateFormula(v.Formula); err != nil {
			errors[v.Name] = err
		}
	}

	return errors
}

// PreviewValuation shows what a formula would calculate for sample data
type ValuationPreview struct {
	Name       string         `json:"name"`
	Formula    string         `json:"formula"`
	SampleData map[string]any `json:"sample_data"`
	Result     float64        `json:"result"`
	Error      string         `json:"error,omitempty"`
}

// PreviewFormula evaluates a formula with sample data for preview/testing
func PreviewFormula(formula string, amount float64, daysHeld float64, note string) ValuationPreview {
	ctx := ValuationContext{
		Amount:     amount,
		Quantity:   1,
		Date:       time.Now().Add(-time.Duration(daysHeld*24) * time.Hour),
		DaysHeld:   daysHeld,
		MonthsHeld: daysHeld / 30.44,
		YearsHeld:  daysHeld / 365.25,
		Note:       note,
		Account:    "Assets:Preview",
		Commodity:  config.DefaultCurrency(),
	}

	preview := ValuationPreview{
		Formula: formula,
		SampleData: map[string]any{
			"amount":      amount,
			"days_held":   daysHeld,
			"months_held": ctx.MonthsHeld,
			"years_held":  ctx.YearsHeld,
			"note":        note,
		},
	}

	options := append([]expr.Option{expr.Env(ctx)}, exprFunctions...)
	program, err := expr.Compile(formula, options...)
	if err != nil {
		preview.Error = fmt.Sprintf("Compile error: %v", err)
		return preview
	}

	result, err := expr.Run(program, ctx)
	if err != nil {
		preview.Error = fmt.Sprintf("Evaluation error: %v", err)
		return preview
	}

	preview.Result = toFloat64(result)
	return preview
}

// GetCustomMarketPrice attempts to calculate a custom market price for a posting.
// Returns the calculated price and true if a custom valuation was applied,
// or zero and false if no custom valuation matches.
func GetCustomMarketPrice(p posting.Posting, evaluationDate time.Time) (decimal.Decimal, bool) {
	valuations := config.GetCustomValuations()
	log.Debugf("GetCustomMarketPrice: checking %d custom valuations for account %s", len(valuations), p.Account)

	valuation := FindCustomValuation(p)
	if valuation == nil {
		log.Debugf("GetCustomMarketPrice: no matching valuation found for account %s", p.Account)
		return decimal.Zero, false
	}

	log.Debugf("GetCustomMarketPrice: found valuation '%s' for account %s", valuation.Name, p.Account)

	price, err := EvaluateValuation(valuation, p, evaluationDate)
	if err != nil {
		log.Warnf("GetCustomMarketPrice: error evaluating valuation '%s' for account %s: %v", valuation.Name, p.Account, err)
		// Fall back to original amount on error
		return p.Amount, true
	}

	log.Debugf("Custom valuation '%s' applied to %s: %s -> %s",
		valuation.Name, p.Account, p.Amount.String(), price.String())

	return price, true
}
