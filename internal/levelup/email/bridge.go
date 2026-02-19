package email

import (
	"context"
	"fmt"
)

func FormatInboundForAgent(msg IncomingMessage) string {
	return fmt.Sprintf("[email]\nfrom: %s\nsubject: %s\nbody:\n%s", msg.From, msg.Subject, msg.Body)
}

func ExecuteActions(ctx context.Context, sender Sender, actions []OutgoingMessage) error {
	if sender == nil || len(actions) == 0 {
		return nil
	}
	for _, action := range actions {
		if err := sender.Send(ctx, action); err != nil {
			return err
		}
	}
	return nil
}
