package service

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestParseNoteFloat(t *testing.T) {
	tests := []struct {
		note     string
		prefix   string
		expected float64
	}{
		{"Int:12.5 Per:M", "Int:", 12.5},
		{"Int:8 Per:Y", "Int:", 8},
		{"Some note Int:15.25 Per:M more text", "Int:", 15.25},
		{"No interest here", "Int:", 0},
		{"Per:M Int:10", "Int:", 10},
	}

	for _, tt := range tests {
		result := parseNoteFloat(tt.note, tt.prefix)
		assert.Equal(t, tt.expected, result, "parseNoteFloat(%q, %q)", tt.note, tt.prefix)
	}
}

func TestParseNoteString(t *testing.T) {
	tests := []struct {
		note     string
		prefix   string
		expected string
	}{
		{"Int:12.5 Per:M", "Per:", "M"},
		{"Int:8 Per:Y", "Per:", "Y"},
		{"No period here", "Per:", ""},
	}

	for _, tt := range tests {
		result := parseNoteString(tt.note, tt.prefix)
		assert.Equal(t, tt.expected, result, "parseNoteString(%q, %q)", tt.note, tt.prefix)
	}
}

func TestMatchAccountPattern(t *testing.T) {
	tests := []struct {
		account  string
		pattern  string
		expected bool
	}{
		{"Assets:p2p:Lender1", "Assets:p2p:*", true},
		{"Assets:p2p:Lender1:Loan1", "Assets:p2p:*", true},
		{"Assets:p2p", "Assets:p2p:*", false},
		{"Assets:Equity:Stock", "Assets:p2p:*", false},
		{"Assets:p2p:Lender1", "Assets:p2p:Lender1", true},
		{"Assets:p2p:Lender2", "Assets:p2p:Lender1", false},
		{"Assets:Checking", "Assets:*", true},
	}

	for _, tt := range tests {
		result := matchAccountPattern(tt.account, tt.pattern)
		assert.Equal(t, tt.expected, result, "matchAccountPattern(%q, %q)", tt.account, tt.pattern)
	}
}

func TestEvaluateValuation(t *testing.T) {
	now := time.Now()
	postingDate := now.Add(-30 * 24 * time.Hour) // 30 days ago

	p := posting.Posting{
		Account:         "Assets:p2p:Lender1",
		Amount:          decimal.NewFromInt(10000),
		Quantity:        decimal.NewFromInt(1),
		Date:            postingDate,
		TransactionNote: "live Int:12 Per:M",
	}

	// Test simple interest calculation: amount + (amount * interest/100 * days/365)
	valuation := &config.CustomValuation{
		Name:         "P2P Monthly Interest",
		Account:      "Assets:p2p:*",
		NoteContains: "live",
		Formula:      "amount + (amount * parse_note_float(note, \"Int:\") / 100 / 365 * days_held)",
	}

	result, err := EvaluateValuation(valuation, p, now)
	assert.NoError(t, err)

	// Expected: 10000 + (10000 * 12 / 100 / 365 * 30) = 10000 + 98.63 ≈ 10098.63
	expected := 10000 + (10000 * 12.0 / 100 / 365 * 30)
	assert.InDelta(t, expected, result.InexactFloat64(), 0.01)
}

func TestEvaluateValuationMonthlyRate(t *testing.T) {
	now := time.Now()
	postingDate := now.Add(-30 * 24 * time.Hour) // 30 days ago

	p := posting.Posting{
		Account:         "Assets:p2p:Lender1",
		Amount:          decimal.NewFromInt(10000),
		Quantity:        decimal.NewFromInt(1),
		Date:            postingDate,
		TransactionNote: "live Int:1 Per:M", // 1% per month
	}

	// Monthly interest: amount * (1 + interest/100 * days/30)
	valuation := &config.CustomValuation{
		Name:         "P2P Monthly Interest",
		Account:      "Assets:p2p:*",
		NoteContains: "live",
		Formula:      "amount * (1 + parse_note_float(note, \"Int:\") / 100 * days_held / 30)",
	}

	result, err := EvaluateValuation(valuation, p, now)
	assert.NoError(t, err)

	// Expected: 10000 * (1 + 1/100 * 30/30) = 10000 * 1.01 = 10100
	expected := 10000 * (1 + 1.0/100*30/30)
	assert.InDelta(t, expected, result.InexactFloat64(), 0.01)
}

func TestSimpleInterestFunction(t *testing.T) {
	now := time.Now()
	postingDate := now.Add(-365 * 24 * time.Hour) // 1 year ago

	p := posting.Posting{
		Account:         "Assets:FD:HDFC",
		Amount:          decimal.NewFromInt(100000),
		Quantity:        decimal.NewFromInt(1),
		Date:            postingDate,
		TransactionNote: "Rate:7.5",
	}

	valuation := &config.CustomValuation{
		Name:    "FD Interest",
		Account: "Assets:FD:*",
		Formula: "amount + simple_interest(amount, parse_note_float(note, \"Rate:\"), days_held)",
	}

	result, err := EvaluateValuation(valuation, p, now)
	assert.NoError(t, err)

	// Expected: 100000 + simple_interest(100000, 7.5, 365) = 100000 + 7500 = 107500
	expected := 100000 + 100000*7.5/100*365/365
	assert.InDelta(t, expected, result.InexactFloat64(), 1)
}

func TestCompoundInterestFunction(t *testing.T) {
	now := time.Now()
	postingDate := now.Add(-365 * 24 * time.Hour) // 1 year ago

	p := posting.Posting{
		Account:         "Assets:FD:HDFC",
		Amount:          decimal.NewFromInt(100000),
		Quantity:        decimal.NewFromInt(1),
		Date:            postingDate,
		TransactionNote: "Rate:7.5",
	}

	valuation := &config.CustomValuation{
		Name:    "FD Compound Interest",
		Account: "Assets:FD:*",
		Formula: "compound_interest(amount, parse_note_float(note, \"Rate:\"), days_held, 4)", // quarterly
	}

	result, err := EvaluateValuation(valuation, p, now)
	assert.NoError(t, err)

	// Expected: 100000 * (1 + 0.075/4)^(4*1) = 100000 * 1.0771 ≈ 107713
	expected := 100000 * 1.0771
	assert.InDelta(t, expected, result.InexactFloat64(), 50)
}

func TestMonthlyInterestFunction(t *testing.T) {
	now := time.Now()
	postingDate := now.Add(-30 * 24 * time.Hour) // 30 days ago

	p := posting.Posting{
		Account:         "Assets:p2p:Lender",
		Amount:          decimal.NewFromInt(10000),
		Quantity:        decimal.NewFromInt(1),
		Date:            postingDate,
		TransactionNote: "Rate:1", // 1% per month
	}

	valuation := &config.CustomValuation{
		Name:    "P2P Monthly",
		Account: "Assets:p2p:*",
		Formula: "amount + monthly_interest(amount, parse_note_float(note, \"Rate:\"), days_held)",
	}

	result, err := EvaluateValuation(valuation, p, now)
	assert.NoError(t, err)

	// Expected: 10000 + (10000 * 1/100 * 30/30) = 10100
	expected := 10000 + 10000*1.0/100*30/30
	assert.InDelta(t, expected, result.InexactFloat64(), 0.01)
}

func TestMathFunctions(t *testing.T) {
	now := time.Now()
	p := posting.Posting{
		Account:         "Assets:Test",
		Amount:          decimal.NewFromFloat(1234.567),
		Quantity:        decimal.NewFromInt(1),
		Date:            now,
		TransactionNote: "",
	}

	tests := []struct {
		formula  string
		expected float64
	}{
		{"round(amount)", 1235},
		{"floor(amount)", 1234},
		{"ceil(amount)", 1235},
		{"abs(-100)", 100},
		{"min(amount, 1000)", 1000},
		{"max(amount, 1000)", 1234.567},
		{"pow(2, 10)", 1024},
		{"sqrt(144)", 12},
		{"clamp(amount, 1000, 1200)", 1200},
		{"clamp(amount, 1000, 1500)", 1234.567},
		{"clamp(amount, 1300, 1500)", 1300},
	}

	for _, tt := range tests {
		valuation := &config.CustomValuation{
			Name:    "Test",
			Account: "Assets:*",
			Formula: tt.formula,
		}

		result, err := EvaluateValuation(valuation, p, now)
		assert.NoError(t, err, "Formula: %s", tt.formula)
		assert.InDelta(t, tt.expected, result.InexactFloat64(), 0.001, "Formula: %s", tt.formula)
	}
}

func TestConditionalFunctions(t *testing.T) {
	now := time.Now()
	p := posting.Posting{
		Account:         "Assets:Test",
		Amount:          decimal.NewFromFloat(1000),
		Quantity:        decimal.NewFromInt(1),
		Date:            now.Add(-100 * 24 * time.Hour),
		TransactionNote: "",
	}

	tests := []struct {
		formula  string
		expected float64
	}{
		{"if_else(days_held > 90, amount * 1.1, amount)", 1100}, // > 90 days
		{"if_else(days_held < 30, amount * 1.05, amount * 1.1)", 1100}, // >= 30 days
	}

	for _, tt := range tests {
		valuation := &config.CustomValuation{
			Name:    "Test",
			Account: "Assets:*",
			Formula: tt.formula,
		}

		result, err := EvaluateValuation(valuation, p, now)
		assert.NoError(t, err, "Formula: %s", tt.formula)
		assert.InDelta(t, tt.expected, result.InexactFloat64(), 0.001, "Formula: %s", tt.formula)
	}
}

func TestValidateFormula(t *testing.T) {
	tests := []struct {
		formula   string
		expectErr bool
	}{
		{"amount + simple_interest(amount, 12, days_held)", false},
		{"amount * 1.1", false},
		{"compound_interest(amount, 8, days_held, 12)", false},
		{"invalid_function(amount)", true},
		{"amount +", true}, // syntax error
		{"\"string result\"", true}, // wrong return type
	}

	for _, tt := range tests {
		err := ValidateFormula(tt.formula)
		if tt.expectErr {
			assert.Error(t, err, "Formula: %s", tt.formula)
		} else {
			assert.NoError(t, err, "Formula: %s", tt.formula)
		}
	}
}

func TestPreviewFormula(t *testing.T) {
	preview := PreviewFormula(
		"amount + simple_interest(amount, 12, days_held)",
		10000,
		365,
		"",
	)

	assert.Empty(t, preview.Error)
	// Expected: 10000 + (10000 * 12/100 * 365/365) = 11200
	assert.InDelta(t, 11200, preview.Result, 1)
	assert.Equal(t, float64(10000), preview.SampleData["amount"])
	assert.Equal(t, float64(365), preview.SampleData["days_held"])
}

func TestMonthsAndYearsHeld(t *testing.T) {
	now := time.Now()
	postingDate := now.Add(-365 * 24 * time.Hour) // 1 year ago

	p := posting.Posting{
		Account:         "Assets:Test",
		Amount:          decimal.NewFromInt(10000),
		Quantity:        decimal.NewFromInt(1),
		Date:            postingDate,
		TransactionNote: "",
	}

	// Test months_held
	valuation := &config.CustomValuation{
		Name:    "Test",
		Account: "Assets:*",
		Formula: "months_held",
	}

	result, err := EvaluateValuation(valuation, p, now)
	assert.NoError(t, err)
	assert.InDelta(t, 12, result.InexactFloat64(), 0.5) // ~12 months

	// Test years_held
	valuation.Formula = "years_held"
	result, err = EvaluateValuation(valuation, p, now)
	assert.NoError(t, err)
	assert.InDelta(t, 1, result.InexactFloat64(), 0.01) // ~1 year
}

func TestNoteContainsFunction(t *testing.T) {
	now := time.Now()
	p := posting.Posting{
		Account:         "Assets:Test",
		Amount:          decimal.NewFromInt(10000),
		Quantity:        decimal.NewFromInt(1),
		Date:            now,
		TransactionNote: "live compound Int:12",
	}

	tests := []struct {
		formula  string
		expected float64
	}{
		{"if_else(note_contains(note, \"live\"), amount * 1.1, amount)", 11000},
		{"if_else(note_contains(note, \"compound\"), amount * 1.2, amount)", 12000},
		{"if_else(note_contains(note, \"simple\"), amount * 1.1, amount)", 10000},
	}

	for _, tt := range tests {
		valuation := &config.CustomValuation{
			Name:    "Test",
			Account: "Assets:*",
			Formula: tt.formula,
		}

		result, err := EvaluateValuation(valuation, p, now)
		assert.NoError(t, err, "Formula: %s", tt.formula)
		assert.InDelta(t, tt.expected, result.InexactFloat64(), 0.001, "Formula: %s", tt.formula)
	}
}
