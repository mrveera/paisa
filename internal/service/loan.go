package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// LoanStatus represents the current status of a loan
type LoanStatus string

const (
	LoanStatusActive   LoanStatus = "active"
	LoanStatusMaturing LoanStatus = "maturing"
	LoanStatusOverdue  LoanStatus = "overdue"
	LoanStatusClosed   LoanStatus = "closed"
)

// Loan represents a tracked loan/P2P investment
type Loan struct {
	Account         string          `json:"account"`
	Principal       decimal.Decimal `json:"principal"`
	CurrentValue    decimal.Decimal `json:"current_value"`
	GainAmount      decimal.Decimal `json:"gain_amount"`
	InterestRate    float64         `json:"interest_rate"`
	Period          string          `json:"period"`
	StartDate       time.Time       `json:"start_date"`
	MaturityDate    *time.Time      `json:"maturity_date"`
	DaysToMaturity  int             `json:"days_to_maturity"`
	DaysHeld        int             `json:"days_held"`
	Status          LoanStatus      `json:"status"`
	RiskLevel       string          `json:"risk_level"`
	PercentComplete float64         `json:"percent_complete"`
	Postings        []posting.Posting `json:"postings"`
}

// LoanSummary provides aggregate statistics about loans
type LoanSummary struct {
	TotalLent     decimal.Decimal        `json:"total_lent"`
	TotalValue    decimal.Decimal        `json:"total_value"`
	TotalGain     decimal.Decimal        `json:"total_gain"`
	TotalAccounts int                    `json:"total_accounts"`
	ByStatus      map[LoanStatus]StatusSummary `json:"by_status"`
	ByRisk        map[string]RiskSummary       `json:"by_risk"`
}

// StatusSummary provides summary for a loan status
type StatusSummary struct {
	Count  int             `json:"count"`
	Amount decimal.Decimal `json:"amount"`
}

// RiskSummary provides summary for a risk level
type RiskSummary struct {
	Count  int             `json:"count"`
	Amount decimal.Decimal `json:"amount"`
}

// LoanAlert represents an actionable alert for a loan
type LoanAlert struct {
	Type     string          `json:"type"`
	Severity string          `json:"severity"`
	Account  string          `json:"account"`
	Message  string          `json:"message"`
	Amount   decimal.Decimal `json:"amount"`
	DaysOverdue int          `json:"days_overdue,omitempty"`
	DaysToMaturity int       `json:"days_to_maturity,omitempty"`
}

// GetLoans returns all tracked loans based on custom valuations config
func GetLoans(db *gorm.DB) []Loan {
	valuations := config.GetCustomValuations()
	if len(valuations) == 0 {
		return []Loan{}
	}

	var loans []Loan
	now := utils.EndOfToday()

	// Get all postings that might be loans
	for _, v := range valuations {
		// Get postings matching this valuation's account pattern
		accountPattern := v.Account
		if accountPattern == "" {
			continue
		}

		// Convert pattern to query
		pattern := accountPattern
		if pattern[len(pattern)-1] == '*' {
			pattern = pattern[:len(pattern)-1] + "%"
		}

		// Get ALL postings for the account pattern (don't filter by NoteContains)
		// We need all postings to properly detect closed status
		var postings []posting.Posting
		postings = query.Init(db).Like(pattern).All()
		postings = lo.Filter(postings, func(p posting.Posting, _ int) bool {
			return matchAccountPattern(p.Account, accountPattern)
		})

		// Group by account
		byAccount := lo.GroupBy(postings, func(p posting.Posting) string { return p.Account })

		for account, ps := range byAccount {
			loan := buildLoan(db, account, ps, now)
			if loan != nil {
				loans = append(loans, *loan)
			}
		}
	}

	// Sort by status (overdue first, then maturing, then active)
	sort.Slice(loans, func(i, j int) bool {
		statusOrder := map[LoanStatus]int{
			LoanStatusOverdue:  0,
			LoanStatusMaturing: 1,
			LoanStatusActive:   2,
			LoanStatusClosed:   3,
		}
		if statusOrder[loans[i].Status] != statusOrder[loans[j].Status] {
			return statusOrder[loans[i].Status] < statusOrder[loans[j].Status]
		}
		return loans[i].Account < loans[j].Account
	})

	return loans
}

// isLoanClosed checks if any posting has "closed" or "settled" in the note
func isLoanClosed(postings []posting.Posting) bool {
	for _, p := range postings {
		note := p.TransactionNote + " " + p.Note
		if strings.Contains(strings.ToLower(note), "closed") ||
			strings.Contains(strings.ToLower(note), "settled") {
			return true
		}
	}
	return false
}

// buildLoan creates a Loan from a set of postings for an account
func buildLoan(db *gorm.DB, account string, postings []posting.Posting, now time.Time) *Loan {
	if len(postings) == 0 {
		return nil
	}

	// Sort by date
	sort.Slice(postings, func(i, j int) bool {
		return postings[i].Date.Before(postings[j].Date)
	})

	// Check if loan is explicitly closed
	isClosed := isLoanClosed(postings)

	// Calculate totals
	var principal decimal.Decimal
	var currentValue decimal.Decimal
	var startDate time.Time

	for i, p := range postings {
		if i == 0 {
			startDate = p.Date
		}
		if p.Amount.GreaterThan(decimal.Zero) {
			principal = principal.Add(p.Amount)
		}
		
		// For closed loans, use actual amount (no interest calculation)
		// For active loans, use market price (with interest)
		if isClosed {
			currentValue = currentValue.Add(p.Amount)
		} else {
			currentValue = currentValue.Add(GetMarketPrice(db, p, now))
		}
	}

	// Determine if loan is closed by balance or explicit status
	balanceIsZero := currentValue.LessThanOrEqual(decimal.Zero)

	// Parse loan info from first posting's note
	firstPosting := postings[0]
	interestRate := parseNoteFloat(firstPosting.TransactionNote, "Int:")
	period := parseNoteString(firstPosting.TransactionNote, "Per:")
	targetDays := ParseDuration(parseNoteString(firstPosting.TransactionNote, "Target:"))
	riskLevel := ParseRiskLevel(firstPosting.TransactionNote)

	daysHeld := int(now.Sub(startDate).Hours() / 24)

	// Calculate maturity
	var maturityDate *time.Time
	var daysToMaturity int
	var percentComplete float64
	var status LoanStatus

	// Determine status
	if isClosed || balanceIsZero {
		status = LoanStatusClosed
		percentComplete = 100
	} else if targetDays > 0 {
		md := startDate.Add(time.Duration(targetDays*24) * time.Hour)
		maturityDate = &md
		daysToMaturity = int(md.Sub(now).Hours() / 24)
		percentComplete = float64(daysHeld) / targetDays * 100
		if percentComplete > 100 {
			percentComplete = 100
		}

		if daysToMaturity < 0 {
			status = LoanStatusOverdue
		} else if daysToMaturity <= 30 {
			status = LoanStatusMaturing
		} else {
			status = LoanStatusActive
		}
	} else {
		status = LoanStatusActive
		daysToMaturity = 0
	}

	gainAmount := currentValue.Sub(principal)
	
	// For closed loans, gain is actual balance (could be 0 or final settlement)
	if status == LoanStatusClosed && balanceIsZero {
		gainAmount = decimal.Zero
	}

	return &Loan{
		Account:         account,
		Principal:       principal,
		CurrentValue:    currentValue,
		GainAmount:      gainAmount,
		InterestRate:    interestRate,
		Period:          period,
		StartDate:       startDate,
		MaturityDate:    maturityDate,
		DaysToMaturity:  daysToMaturity,
		DaysHeld:        daysHeld,
		Status:          status,
		RiskLevel:       riskLevel,
		PercentComplete: percentComplete,
		Postings:        postings,
	}
}

// GetLoanSummary returns aggregate statistics about all loans
func GetLoanSummary(db *gorm.DB) LoanSummary {
	loans := GetLoans(db)

	summary := LoanSummary{
		TotalLent:     decimal.Zero,
		TotalValue:    decimal.Zero,
		TotalGain:     decimal.Zero,
		TotalAccounts: len(loans),
		ByStatus:      make(map[LoanStatus]StatusSummary),
		ByRisk:        make(map[string]RiskSummary),
	}

	for _, loan := range loans {
		summary.TotalLent = summary.TotalLent.Add(loan.Principal)
		summary.TotalValue = summary.TotalValue.Add(loan.CurrentValue)
		summary.TotalGain = summary.TotalGain.Add(loan.GainAmount)

		// By status
		ss := summary.ByStatus[loan.Status]
		ss.Count++
		ss.Amount = ss.Amount.Add(loan.Principal)
		summary.ByStatus[loan.Status] = ss

		// By risk
		rs := summary.ByRisk[loan.RiskLevel]
		rs.Count++
		rs.Amount = rs.Amount.Add(loan.Principal)
		summary.ByRisk[loan.RiskLevel] = rs
	}

	return summary
}

// GetLoanAlerts returns actionable alerts for loans
func GetLoanAlerts(db *gorm.DB) []LoanAlert {
	loans := GetLoans(db)
	var alerts []LoanAlert

	for _, loan := range loans {
		switch loan.Status {
		case LoanStatusOverdue:
			alerts = append(alerts, LoanAlert{
				Type:        "overdue",
				Severity:    "high",
				Account:     loan.Account,
				Message:     formatAlertMessage("Loan overdue by %d days", -loan.DaysToMaturity),
				Amount:      loan.Principal,
				DaysOverdue: -loan.DaysToMaturity,
			})
		case LoanStatusMaturing:
			alerts = append(alerts, LoanAlert{
				Type:           "maturing",
				Severity:       "medium",
				Account:        loan.Account,
				Message:        formatAlertMessage("Loan matures in %d days", loan.DaysToMaturity),
				Amount:         loan.Principal,
				DaysToMaturity: loan.DaysToMaturity,
			})
		}
	}

	// Sort by severity (high first)
	sort.Slice(alerts, func(i, j int) bool {
		severityOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
		return severityOrder[alerts[i].Severity] < severityOrder[alerts[j].Severity]
	})

	return alerts
}

// formatAlertMessage formats an alert message with proper pluralization
func formatAlertMessage(format string, days int) string {
	return fmt.Sprintf(format, days)
}

