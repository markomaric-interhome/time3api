package user

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

type Store struct {
	db *sql.DB
}

type scanner interface {
	Scan(dest ...any) error
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db: db,
	}
}

func scanUser(row scanner) (*User, error) {
	u := &User{}

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
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}

	// convert the values where db and go struct differ in types
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
 * Create a new user with all the information provided. This automatically sets `createdAt`
 * and `updatedAt` timestamps for the user with the current UTC time.
 */
func (s *Store) Create(ctx context.Context, user *User) (*User, error) {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	// convert the "start_date" into a specific format string
	var startDate any
	if user.StartDate != nil {
		startDate = user.StartDate.Format(dateLayout)
	}

	// create the user in the database & return it
	row := s.db.QueryRowContext(ctx, queryCreate, user.Team, user.Role, user.Email, user.PasswordHash, user.Firstname, user.Lastname, startDate, user.CreatedAt.Unix(), user.UpdatedAt.Unix())
	usr, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("update user %d: %w", user.ID, err)
	}

	return usr, nil
}

/**
 * Delete the target user. Data is being kept in tact, only a flag is being set with the user.
 */
func (s *Store) Delete(ctx context.Context, id int64) (*User, error) {
	row := s.db.QueryRowContext(ctx, queryDelete, time.Now().UTC().Unix(), id)
	usr, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("delete user %d: %w", id, err)
	}

	return usr, nil
}

/**
 * Update the user with the provided information. The "updated_at" timestamp gets
 * updated automatically.
 */
func (s *Store) Update(ctx context.Context, id int64, req UpdateUserRequest) (*User, error) {
	sets := []string{}
	args := []any{}

	if req.Firstname != nil {
		sets = append(sets, "firstname = ?")
		args = append(args, strings.TrimSpace(*req.Firstname))
	}

	if req.Lastname != nil {
		sets = append(sets, "lastname = ?")
		args = append(args, strings.TrimSpace(*req.Lastname))
	}

	// add the update timestamp
	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC().Unix())

	// add the id of the user
	args = append(args, id)

	// build the query
	query := `
		UPDATE users
		SET ` + strings.Join(sets, ", ") + `
		WHERE id = ?
		RETURNING
			id,
			team,
			role,
			email,
			password_hash,
			firstname,
			lastname,
			start_date,
			created_at,
			updated_at,
			deleted_at,
			last_login_at
	`

	// run the query
	row := s.db.QueryRowContext(ctx, query, args...)
	usr, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("update user %d: %w", id, err)
	}

	return usr, nil
}

/**
 * Update the role of the target user.
 */
func (s *Store) UpdateRole(ctx context.Context, id int64, role Role) (*User, error) {
	row := s.db.QueryRowContext(ctx, queryUpdateRole, role, time.Now().UTC().Unix(), id)
	usr, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("update role for user %d: %w", id, err)
	}

	return usr, nil
}

/**
 * Update the "last_login_at" timestamp of the target user.
 */
func (s *Store) UpdateLastLoginAt(ctx context.Context, id int64) (*User, error) {
	row := s.db.QueryRowContext(ctx, queryUpdateLastLoginAt, time.Now().UTC().Unix(), id)
	usr, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("update last_login_at for user %d: %w", id, err)
	}

	return usr, nil
}

/**
 * Get every user.
 */
func (s *Store) GetAll(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, queryGetAll)
	if err != nil {
		return nil, fmt.Errorf("get all users: %w", err)
	}

	defer rows.Close()

	// convert the returned rows to go structs
	var users []User
	for rows.Next() {
		usr, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan all got users: %w", err)
		}

		users = append(users, *usr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all users: %w", err)
	}

	return users, nil
}

/**
 * Get all users with the provided role.
 */
func (s *Store) GetAllByRole(ctx context.Context, role Role) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, queryGetAllByRole, role)
	if err != nil {
		return nil, fmt.Errorf("get all users by role %s: %w", role, err)
	}

	defer rows.Close()

	// convert the returned rows to go structs
	var users []User
	for rows.Next() {
		usr, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan all got users by role %s: %w", role, err)
		}

		users = append(users, *usr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all users with role %s: %w", role, err)
	}

	return users, nil
}

/**
 * Get the target user by it's id.
 */
func (s *Store) GetByID(ctx context.Context, id int64) (*User, error) {
	row := s.db.QueryRowContext(ctx, queryGetById, id)
	usr, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("get user by id %d: %w", id, err)
	}

	return usr, nil
}

/**
 * Get the target user by it's email address.
 */
func (s *Store) GetByEmail(ctx context.Context, email string) (*User, error) {
	row := s.db.QueryRowContext(ctx, queryGetByEmail, email)
	usr, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("get user by email %s: %w", email, err)
	}

	return usr, nil
}

const queryCreate = `
	INSERT INTO users (
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at
	)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	RETURNING
		id,
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at,
		deleted_at,
		last_login_at
`

const queryDelete = `
	UPDATE users
	SET deleted_at = ?
	WHERE id = ?
	RETURNING
		id,
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at,
		deleted_at,
		last_login_at
`

const queryUpdateRole = `
	UPDATE users
	SET role = ?, updated_at = ?
	WHERE id = ?
	RETURNING
		id,
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at,
		deleted_at,
		last_login_at
`

const queryUpdateLastLoginAt = `
	UPDATE users
	SET last_login_at = ?
	WHERE id = ?
	RETURNING
		id,
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at,
		deleted_at,
		last_login_at
`

const queryGetAll = `
	SELECT
		id,
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at,
		deleted_at,
		last_login_at
	FROM users
`

const queryGetAllByRole = `
	SELECT
		id,
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at,
		deleted_at,
		last_login_at
	FROM users
	WHERE role = ?
`

const queryGetById = `
	SELECT
		id,
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at,
		deleted_at,
		last_login_at
	FROM users WHERE id = ?
`

const queryGetByEmail = `
	SELECT
		id,
		team,
		role,
		email,
		password_hash,
		firstname,
		lastname,
		start_date,
		created_at,
		updated_at,
		deleted_at,
		last_login_at
	FROM users WHERE email = ?
`
