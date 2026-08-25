package review

import "time"

type Review struct {
	ID           int64
	ApprenticeID int64
	TrainerID    int64

	Status      Status
	PeriodStart time.Time
	PeriodEnd   time.Time

	CreatedAt   time.Time
	UpdatedAt   time.Time
	SubmittedAt *time.Time
	CompletedAt *time.Time
}

type CreateReviewRequest struct {
	ApprenticeID int64  `json:"apprenticeId" validate:"required"`
	TrainerID    int64  `json:"trainerId" validate:"required"`
	PeriodStart  string `json:"periodStart" validate:"datetime=2006-01-02"`
	PeriodEnd    string `json:"periodEnd" validate:"datetime=2006-01-02"`
}

type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
	StatusCompleted Status = "completed"
)
