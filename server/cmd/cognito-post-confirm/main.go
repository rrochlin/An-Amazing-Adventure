// cognito-post-confirm is a Cognito Post Confirmation trigger Lambda.
// It fires after a user confirms their email address, creating a default
// restricted UserRecord in the users table. If clientMetadata contains an
// inviteCode the invite is redeemed and a membership record is written.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
)

func handler(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (
	events.CognitoEventUserPoolsPostConfirmation, error,
) {
	userID := event.Request.UserAttributes["sub"]
	if userID == "" {
		log.Printf("cognito-post-confirm: missing sub in user attributes — skipping")
		return event, nil // non-fatal — don't block signup
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		log.Printf("cognito-post-confirm: db init error: %v", err)
		return event, nil // non-fatal
	}

	now := time.Now().UnixMilli()
	record := db.UserRecord{
		UserID:      db.BinaryID(userID),
		Role:        "restricted",
		AIEnabled:   false,
		TokenLimit:  0,
		TokensUsed:  0,
		GamesLimit:  1,
		BillingMode: "admin_granted",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := dbClient.PutUser(ctx, record); err != nil {
		// Non-fatal: user can still log in; GetUser returns nil and the system
		// treats missing records as restricted.
		log.Printf("cognito-post-confirm: PutUser error (non-fatal): %v", err)
	}

	// Redeem invite code if the client passed one in signUp clientMetadata
	if code, ok := event.Request.ClientMetadata["inviteCode"]; ok && code != "" {
		if err := redeemInvite(ctx, dbClient, userID, code); err != nil {
			log.Printf("cognito-post-confirm: redeemInvite(%s) error (non-fatal): %v", code, err)
		}
	}

	log.Printf("cognito-post-confirm: created restricted user record for %s", userID)
	return event, nil
}

func redeemInvite(ctx context.Context, dbClient *db.Client, userID, code string) error {
	invite, err := dbClient.GetInvite(ctx, code)
	if err != nil {
		return fmt.Errorf("GetInvite: %w", err)
	}
	if invite == nil {
		return fmt.Errorf("invite %s not found or expired", code)
	}
	if invite.MaxUses > 0 && invite.Uses >= invite.MaxUses {
		return fmt.Errorf("invite %s is full (%d/%d uses)", code, invite.Uses, invite.MaxUses)
	}

	if err := dbClient.PutMembership(ctx, db.MembershipRecord{
		UserID:    db.BinaryID(userID),
		SessionID: invite.SessionID,
		Role:      "member",
		JoinedAt:  time.Now().UnixMilli(),
	}); err != nil {
		return fmt.Errorf("PutMembership: %w", err)
	}

	return dbClient.IncrementInviteUses(ctx, code)
}

func main() {
	lambda.Start(handler)
}
