package assignment

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"time3api/app/user"
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
 * Scan a row from the "assignments" table. Works for both "row" and "rows."
 */
func scanAssignment(row scanner) (*Assignment, error) {
	a := &Assignment{}

	var createdAt int64

	err := row.Scan(&a.TrainerID, &a.ApprenticeID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan assignment: %w", err)
	}

	a.CreatedAt = time.Unix(createdAt, 0).UTC()

	return a, nil
}

/**
 * Scan a row from the "users" table. Works for both "row" and "rows".
 */
func scanUser(row scanner) (*user.User, error) {
	u := &user.User{}

	var (
		startDate   sql.NullString
		createdAt   int64
		updatedAt   int64
		deletedAt   sql.NullInt64
		lastLoginAt sql.NullInt64
	)

	err := row.Scan(
		&u.ID,
		&u.Team,
		&u.Role,
		&u.Email,
		&u.PasswordHash,
		&u.Firstname,
		&u.Lastname,
		&startDate,
		&createdAt,
		&updatedAt,
		&deletedAt,
		&lastLoginAt,
	)

	// check for errors in the scanning process
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}

	// convert the values where db and go struct types differ
	u.CreatedAt = time.Unix(createdAt, 0).UTC()
	u.UpdatedAt = time.Unix(updatedAt, 0).UTC()

	if startDate.Valid {
		t, err := time.Parse(dateLayout, startDate.String)
		if err != nil {
			return nil, fmt.Errorf("parse user start date: %w", err)
		}

		u.StartDate = &t
	}

	if deletedAt.Valid {
		t := time.Unix(deletedAt.Int64, 0).UTC()
		u.DeletedAt = &t
	}

	if lastLoginAt.Valid {
		t := time.Unix(lastLoginAt.Int64, 0).UTC()
		u.LastLoginAt = &t
	}

	return u, nil
}

/**
 * Assign an apprentice to a trainer.
 */
func (s *Store) Assign(ctx context.Context, tID int64, aID int64) (*Assignment, error) {
	row := s.db.QueryRowContext(ctx, queryAssign, tID, aID, time.Now().UTC().Unix())
	ass, err := scanAssignment(row)
	if err != nil {
		return nil, fmt.Errorf("assign apprentice to trainer: %w", err)
	}

	return ass, nil
}

/**
 * Unassign an apprentice from a trainer.
 */
func (s *Store) Unassign(ctx context.Context, tID int64, aID int64) error {
	_, err := s.db.ExecContext(ctx, queryUnassign, tID, aID)
	if err != nil {
		return fmt.Errorf("unassign apprentice from trainer: %w", err)
	}

	return nil
}

/**
 * Get if the trainer is the trainer for the apprentice.
 */
func (s *Store) IsTrainerFor(ctx context.Context, tID int64, aID int64) (bool, error) {
	// check if an assignment exists
	var exists bool
	if err := s.db.QueryRowContext(ctx, queryIsTrainerFor, tID, aID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check assignment: %w", err)
	}

	return exists, nil
}

/**
 * Get all the apprentices assigned to the target trainer.
 */
func (s *Store) ApprenticesForTrainer(ctx context.Context, tID int64) ([]user.User, error) {
	rows, err := s.db.QueryContext(ctx, queryApprenticesForTrainer, tID)
	if err != nil {
		return nil, fmt.Errorf("get apprentices for trainer: %w", err)
	}
	defer rows.Close()

	var users []user.User

	// iterate over the returned rows
	for rows.Next() {
		usr, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan apprentice: %w", err)
		}

		users = append(users, *usr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate apprentices for trainer: %w", err)
	}

	return users, nil
}

/**
 * Get all the trainers supervising the target apprentice.
 */
func (s *Store) TrainersForApprentice(ctx context.Context, aID int64) ([]user.User, error) {
	rows, err := s.db.QueryContext(ctx, queryTrainersForApprentice, aID)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("get trainers for apprentice: %w", err)
	}
	defer rows.Close()

	// iterate over the returned rows
	var users []user.User
	for rows.Next() {
		usr, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan trainer: %w", err)
		}

		users = append(users, *usr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trainers for apprentice: %w", err)
	}

	return users, nil
}

const queryAssign = `
	INSERT INTO assignments (
		trainer_id,
		apprentice_id,
		created_at
	)
	VALUES (?, ?, ?)
	RETURNING
		trainer_id,
		apprentice_id,
		created_at
`

const queryUnassign = `
	DELETE FROM assignments
	WHERE trainer_id = ?
		AND apprentice_id = ?
`

const queryIsTrainerFor = `
	SELECT EXISTS (
		SELECT 1
		FROM assignments
		WHERE trainer_id = ?
			AND apprentice_id = ?
	)
`

const queryApprenticesForTrainer = `
	SELECT
		u.id,
		u.team,
		u.role,
		u.email,
		u.password_hash,
		u.firstname,
		u.lastname,
		u.start_date,
		u.created_at,
		u.updated_at,
		u.deleted_at,
		u.last_login_at
	FROM users AS u
	INNER JOIN assignments AS a
		ON a.apprentice_id = u.id
	WHERE a.trainer_id = ?
	ORDER BY u.firstname, u.lastname
`

const queryTrainersForApprentice = `
	SELECT
		u.id,
		u.team,
		u.role,
		u.email,
		u.password_hash,
		u.firstname,
		u.lastname,
		u.start_date,
		u.created_at,
		u.updated_at,
		u.deleted_at,
		u.last_login_at
	FROM users AS u
	INNER JOIN assignments AS a
		ON a.trainer_id = u.id
	WHERE a.apprentice_id = ?
		AND u.deleted_at IS NULL
	ORDER BY u.firstname, u.lastname
`
