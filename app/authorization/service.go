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

/**
 * Determine if the current user has permissions to view the target's profile.
 */
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

/**
 * Determine if the current user has permissions to delete a user.
 */
func (s *Service) CanDeleteUser(ctx context.Context, currUsr *user.User) (bool, error) {
	if currUsr.Role == user.RoleAdmin {
		// admins can always delete any user
		return true, nil
	}

	// other users cannot delete a user
	return false, nil
}

/**
 * Determine if the current user has permissions to update the target user.
 *
 * Admin -> Can update the information of any user.
 * Trainer -> Can update their apprentices and themselves.
 * Apprentice -> Cannot update users at all.
 */
func (s *Service) CanUpdateUser(ctx context.Context, currUsr *user.User, targetUsr *user.User) (bool, error) {
	switch currUsr.Role {
	case user.RoleAdmin:
		return true, nil

	case user.RoleApprentice:
		return false, nil

	case user.RoleTrainer:
		allowed, err := s.assignments.IsTrainerFor(ctx, currUsr.ID, targetUsr.ID)
		if err != nil {
			return false, fmt.Errorf("check trainer for apprentice access: %w", err)
		}

		return allowed, nil

	default:
		return false, nil
	}
}

/**
 * Determine if the current user has the permissions to assign an apprentice to a trainer. This
 * will also make sure that the "trainer" is a trainer and that the "apprentice" is an apprentice.
 *
 * -> Admin -> Can create a valid assignment between apprentice and trainer.
 * -> Trainer -> Cannot create a valid assignment.
 * -> Apprentice -> Cannot create a valid assignment.
 */
func (s *Service) CanAssignApprentice(ctx context.Context, currUsr *user.User, trainer *user.User, apprentice *user.User) (bool, error) {
	// first, make sure this is a valid assignment
	if trainer.Role != user.RoleTrainer && trainer.Role != user.RoleAdmin {
		return false, nil
	}
	if apprentice.Role != user.RoleApprentice {
		return false, nil
	}

	// check the permissions of the current user
	if currUsr.Role == user.RoleAdmin {
		return true, nil
	}

	return false, nil
}

/**
 * Determine if the current user has the permissions to unassign an apprentice from a trainer.
 *
 * Admin -> Yes, can revoke an assignment between apprentice and trainer.
 * Trainer -> No
 * Apprentice -> No
 */
func (s *Service) CanUnassignApprentice(ctx context.Context, currUsr *user.User) (bool, error) {
	if currUsr.Role == user.RoleAdmin {
		return true, nil
	}

	return false, nil
}
