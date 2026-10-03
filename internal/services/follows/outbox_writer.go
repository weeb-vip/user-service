package follows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/weeb-vip/go-outbox-lib"
	"gorm.io/gorm"
)

// OutboxWriter is the EventWriter backed by go-outbox-lib.
type OutboxWriter struct{}

// Write implements EventWriter.
func (OutboxWriter) Write(ctx context.Context, tx *gorm.DB, subject string, id string, payload any) error {
	eventID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("follows: event id %q is not a uuid: %w", id, err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("follows: marshal event: %w", err)
	}

	return outbox.WriteEvent(ctx, tx, &outbox.Event{ID: eventID, Subject: subject, Payload: outbox.JSON(body)})
}
