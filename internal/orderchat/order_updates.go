package orderchat

import (
	"context"
	"fmt"
)

type OrderUpdate struct {
	OrderID     string `json:"order_id"`
	AccountID   string `json:"account_id"`
	From        string `json:"from"`
	To          string `json:"to"`
	ReceiptURL  string `json:"receipt_url,omitempty"`
	TrackingRef string `json:"tracking_ref,omitempty"`
}

type Publisher interface {
	Publish(ctx context.Context, channel, event string, data any, accountID, idempotencyKey string) error
}

type Relay struct {
	publisher Publisher
}

func NewRelay(publisher Publisher) *Relay { return &Relay{publisher: publisher} }

var transitions = map[string]map[string]string{
	"cart":                {"checkout_complete": "checkout.completed"},
	"checkout_complete":   {"fulfillment_started": "fulfillment.started"},
	"fulfillment_started": {"shipped": "fulfillment.shipped"},
	"shipped":             {"delivered": "order.delivered"},
}

func EventFor(from, to string) (string, bool) {
	next, ok := transitions[from]
	if !ok {
		return "", false
	}
	event, ok := next[to]
	return event, ok
}

func (r *Relay) Send(ctx context.Context, update OrderUpdate) (string, error) {
	event, ok := EventFor(update.From, update.To)
	if !ok {
		return "", fmt.Errorf("order %s cannot move from %s to %s", update.OrderID, update.From, update.To)
	}
	channel := "order:" + update.OrderID
	idempotencyKey := update.OrderID + ":" + update.From + ":" + update.To
	if err := r.publisher.Publish(ctx, channel, event, update, update.AccountID, idempotencyKey); err != nil {
		return "", err
	}
	return event, nil
}
