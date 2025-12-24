package server

import (
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetLoans returns all tracked loans
func GetLoans(db *gorm.DB) gin.H {
	loans := service.GetLoans(db)
	return gin.H{"loans": loans}
}

// GetLoanSummary returns aggregate statistics about loans
func GetLoanSummary(db *gorm.DB) gin.H {
	summary := service.GetLoanSummary(db)
	return gin.H{"summary": summary}
}

// GetLoanAlerts returns actionable alerts for loans
func GetLoanAlerts(db *gorm.DB) gin.H {
	alerts := service.GetLoanAlerts(db)
	return gin.H{"alerts": alerts}
}

// GetLoansDashboard returns a combined view for the loans dashboard
func GetLoansDashboard(db *gorm.DB) gin.H {
	loans := service.GetLoans(db)
	summary := service.GetLoanSummary(db)
	alerts := service.GetLoanAlerts(db)
	
	return gin.H{
		"loans":   loans,
		"summary": summary,
		"alerts":  alerts,
	}
}

