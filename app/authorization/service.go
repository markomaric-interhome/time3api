package authorization

import (
	"context"
	"fmt"

	"time3api/app/assignment"
	"time3api/app/user"
)

type Service struct {
	assignments *assignment.Store
	users       *user.Store
}

func NewService(assignments *assignment.Store, users *user.Store) *Service {
	return &Service{
		assignments: assignments,
		users:       users,
	}
}

// Determine if the current user has permissions to view the target's profile
func (s *Service) CanAccessUser(ctx context.Context, currUsr *user.User, targetUsr *user.User) (bool, error) {
	switch currUsr.Role {
	case user.RoleAdmin:
		// admins can always view every user's information
		return true, nil

	case user.RoleApprentice:
		// apprentices can only load their own information
		return currUsr.ID == targetUsr.ID, nil

	case user.RoleTrainer:
		// trainers can view all trainer profiles + all their assigned apprentices
		if targetUsr.Role == user.RoleTrainer {
			return true, nil
		}

		allowed, err := s.assignments.IsTrainerFor(ctx, currUsr.ID, targetUsr.ID)
		if err != nil {
			return false, fmt.Errorf("check trainer for apprentice access: %w", err)
		}

		return allowed, nil

	default:
		return false, nil
	}
}

// Determine if the current user has permissions to delete a user
func (s *Service) CanDeleteUser(ctx context.Context, currUsr *user.User) (bool, error) {
	switch currUsr.Role {
	case user.RoleAdmin:
		// admins can always delete users
		return true, nil

	case user.RoleApprentice:
		return false, nil

	case user.RoleTrainer:
		return false, nil

	default:
		return false, nil
	}
}

// Determine if the current user has pemissions to update the target
func (s *Service) CanUpdateUser(ctx context.Context, currUsr *user.User, targetUsr *user.User) (bool, error) {
	switch currUsr.Role {
	case user.RoleAdmin:
		// admins can always update user's information
		return true, nil

	case user.RoleApprentice:
		// apprentices can never update user information
		return currUsr.Role == targetUsr.Role, nil

	case user.RoleTrainer:
		// trainers can update user information for their apprentices
		allowed, err := s.assignments.IsTrainerFor(ctx, currUsr.ID, targetUsr.ID)
		if err != nil {
			return false, fmt.Errorf("check trainer for apprentice access: %w", err)
		}

		return allowed, nil

	default:
		return false, nil
	}
}

// Determine if the current user has permissions to update the target's role
func (s *Service) CanUpdateUserRole(ctx context.Context, currUsr *user.User) (bool, error) {
	if currUsr.Role == user.RoleAdmin {
		return true, nil
	}

	return false, nil
}

/**
 * Determine if the current user has permissions to assign an apprentice to a trainer.
 */
func (s *Service) CanAssignApprentice(ctx context.Context, currUsr *user.User, trainer *user.User, apprentice *user.User) (bool, error) {
	// first check if both apprentice and trainer have the proper roles
	if trainer.Role != user.RoleTrainer && trainer.Role != user.RoleAdmin {
		return false, nil
	}
	if apprentice.Role != user.RoleApprentice {
		return false, nil
	}

	// check the permissions of the current user
	switch currUsr.Role {
	case user.RoleAdmin:
		// admins can always create valid assignments
		return true, nil

	case user.RoleTrainer:
		// trainers cannot create any assignments for now
		return false, nil

	case user.RoleApprentice:
		// apprentices cannot create any assignments
		return false, nil

	default:
		return false, nil
	}
}

/**
 * Determine if the current user has permissions to unassign an apprentice from a trainer.
 */
func (s *Service) CanUnassignApprentice(ctx context.Context, currUsr *user.User) (bool, error) {
	if currUsr.Role == user.RoleAdmin {
		return true, nil
	}

	return false, nil
}
