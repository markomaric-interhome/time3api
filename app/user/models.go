package user

import "time"

type User struct {
	ID           int64
	Team         string
	Role         Role
	Email        string
	PasswordHash string

	Firstname string
	Lastname  string
	StartDate *time.Time

	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	LastLoginAt *time.Time
}

type UserRequest struct {
	Team      string `json:"team" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=12,max=72"`
	Firstname string `json:"firstname" validate:"required"`
	Lastname  string `json:"lastname" validate:"required"`
	StartDate string `json:"startDate" validate:"omitempty,datetime=2006-01-02"`
}

type UserResponse struct {
	ID    int64  `json:"id"`
	Team  string `json:"team"`
	Role  Role   `json:"role"`
	Email string `json:"email"`

	Firstname string     `json:"firstname"`
	Lastname  string     `json:"lastname"`
	StartDate *time.Time `json:"startDate"`

	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
}

type UserDetailsResponse struct {
	UserResponse

	Trainers    []UserSummary `json:"trainers,omitempty"`
	Apprentices []UserSummary `json:"apprentices,omitempty"`
}

type UserSummary struct {
	ID        int64  `json:"id"`
	Team      string `json:"team"`
	Role      Role   `json:"role"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleApprentice Role = "apprentice"
	RoleTrainer    Role = "trainer"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// The struct which is being sent from the client when updating a user
type UpdateUserRequest struct {
	Firstname *string `json:"firstname" validate:"omitempty,min=1"`
	Lastname  *string `json:"lastname" validate:"omitempty,min=1"`
}

// The struct which is being sent from the client when updating a user's role
type UpdateUserRoleRequest struct {
	Role Role `json:"role" validate:"required,oneof=apprentice trainer admin"`
}

func (u *User) Response() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Team:      u.Team,
		Role:      u.Role,
		Email:     u.Email,
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
		StartDate: u.StartDate,

		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		DeletedAt:   u.DeletedAt,
		LastLoginAt: u.LastLoginAt,
	}
}

func (u *User) Summary() UserSummary {
	return UserSummary{
		ID:        u.ID,
		Team:      u.Team,
		Role:      u.Role,
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
	}
}
