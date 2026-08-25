package assignment

import "time"

type Assignment struct {
	TrainerID    int64
	ApprenticeID int64

	CreatedAt time.Time
}

type CreateAssignmentRequest struct {
	TrainerID    int64 `json:"trainerID" validate:"required"`
	ApprenticeID int64 `json:"apprenticeID" validate:"required"`
}
