package review

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

type Store struct {
	db *sql.DB
}

type scanner interface {
	Scan(desc ...any) error
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db: db,
	}
}

/**
 * Scan a row from the "reviews" table. Works for both "row" and "rows".
 */
func scanReview(row scanner) (*Review, error) {
	r := &Review{}

	var (
		periodStart string
		periodEnd   string
		createdAt   int64
		updatedAt   int64
		submittedAt sql.NullInt64
		completedAt sql.NullInt64
	)

	err := row.Scan(
		&r.ID,
		&r.ApprenticeID,
		&r.TrainerID,
		&periodStart,
		&periodEnd,
		&r.Status,
		&createdAt,
		&updatedAt,
		&submittedAt,
		&completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan review: %w", err)
	}

	// convert the values where db and go struct type differ
	r.CreatedAt = time.Unix(createdAt, 0).UTC()
	r.UpdatedAt = time.Unix(updatedAt, 0).UTC()

	if submittedAt.Valid {
		t := time.Unix(submittedAt.Int64, 0).UTC()
		r.SubmittedAt = &t
	}

	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0).UTC()
		r.CompletedAt = &t
	}

	ps, err := time.Parse(dateLayout, periodStart)
	if err != nil {
		return nil, fmt.Errorf("parse review period start: %w", err)
	}
	r.PeriodStart = ps

	pe, err := time.Parse(dateLayout, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("parse review period end: %w", err)
	}
	r.PeriodEnd = pe

	return r, nil
}

/**
 * Create a new review. This automatically sets `createdAt` and `updatedAt`
 * timestamps for the user with the current UTC time.
 */
func (s *Store) Create(ctx context.Context, req *CreateReviewRequest) (*Review, error) {
	var periodStart *time.Time
	if req.PeriodStart != "" {
		t, err := time.Parse("2006-01-02", req.PeriodStart)
		if err != nil {
			return nil, fmt.Errorf("parse review period start: %w", err)
		}
		periodStart = &t
	}

	var periodEnd *time.Time
	if req.PeriodEnd != "" {
		t, err := time.Parse("2006-01-02", req.PeriodEnd)
		if err != nil {
			return nil, fmt.Errorf("parse review period end: %w", err)
		}
		periodEnd = &t
	}

	now := time.Now().UTC()

	r := &Review{
		ApprenticeID: req.ApprenticeID,
		TrainerID:    req.TrainerID,
		PeriodStart:  *periodStart,
		PeriodEnd:    *periodEnd,
		Status:       StatusDraft,

		CreatedAt: now,
		UpdatedAt: now,
	}

	// store the new review in the database and populate the missing fields
	row := s.db.QueryRowContext(ctx, queryCreate, r.ApprenticeID, r.TrainerID, r.PeriodStart, r.PeriodEnd, r.Status, r.CreatedAt, r.UpdatedAt)
	rev, err := scanReview(row)
	if err != nil {
		return nil, fmt.Errorf("create review: %w", err)
	}

	return rev, nil
}

/**
 * Delete an existing review. This will completely remove it from the database.
 */
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, queryDelete, id)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}

	return nil
}

const queryCreate = `
	INSERT INTO reviews (
		apprentice_id,
		trainer_id,
		period_start,
		period_end,
		status,
		created_at,
		updated_at,
	)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	RETURNING
		id,
		apprentice_id,
		trainer_id,
		period_start,
		period_end,
		status,
		created_at,
		updated_at,
		submitted_at,
		completed_at
`

const queryDelete = `
	DELETE FROM reviews
	WHERE id = ?
`
