package memory

import "fmt"

var allowedStatusTransitions = map[Status]map[Status]struct{}{
	StatusActive: {
		StatusArchived: {},
		StatusRevoked:  {},
	},
	StatusArchived: {
		StatusActive:  {},
		StatusRevoked: {},
	},
}

func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}
	targets, ok := allowedStatusTransitions[from]
	if !ok {
		return false
	}
	_, ok = targets[to]
	return ok
}

func ValidateStatusTransition(from, to Status) error {
	if CanTransition(from, to) {
		return nil
	}
	return fmt.Errorf("invalid memory status transition from %q to %q", from, to)
}

func IsActive(status Status) bool {
	return status == StatusActive
}
