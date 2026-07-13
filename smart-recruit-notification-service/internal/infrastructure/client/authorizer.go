package client

import (
	"context"
	"fmt"

	"smart-recruit-platform-go/metadata"
)

type ActorVerifier struct{}

func NewActorVerifier() ActorVerifier {
	return ActorVerifier{}
}

func (ActorVerifier) VerifyActorMatch(ctx context.Context, requestUserID int64) error {
	authUserID := metadata.GetAuthUserID(ctx)
	if authUserID == 0 {
		return fmt.Errorf("authenticated user not found in context — gRPC metadata x-authenticated-user-id is required for this operation")
	}
	if requestUserID != authUserID {
		return fmt.Errorf("actor mismatch: authenticated user %d attempted to access resources of user %d", authUserID, requestUserID)
	}
	return nil
}
