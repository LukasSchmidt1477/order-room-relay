package orderchat

import (
	"context"
	"testing"
)

type recordedPublish struct {
	calls int
	event string
	key   string
}

func (r *recordedPublish) Publish(_ context.Context, _, event string, _ any, _, key string) error {
	r.calls++
	r.event = event
	r.key = key
	return nil
}

func TestRelayBusinessTransitions(t *testing.T) {
	tests := []struct {
		name      string
		from      string
		to        string
		wantEvent string
		wantCalls int
	}{
		{"checkout starts fulfillment", "checkout_complete", "fulfillment_started", "fulfillment.started", 1},
		{"shipment reaches customer", "shipped", "delivered", "order.delivered", 1},
		{"delivery cannot return to cart", "delivered", "cart", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &recordedPublish{}
			relay := NewRelay(publisher)
			event, err := relay.Send(context.Background(), OrderUpdate{
				OrderID: "ord_42", AccountID: "acct_7", From: tt.from, To: tt.to,
			})
			if tt.wantCalls == 0 && err == nil {
				t.Fatal("expected invalid transition to be rejected")
			}
			if tt.wantCalls == 1 && err != nil {
				t.Fatalf("Send returned %v", err)
			}
			if event != tt.wantEvent || publisher.calls != tt.wantCalls {
				t.Fatalf("event=%q calls=%d, want event=%q calls=%d", event, publisher.calls, tt.wantEvent, tt.wantCalls)
			}
			if tt.wantCalls == 1 && publisher.key != "ord_42:"+tt.from+":"+tt.to {
				t.Fatalf("unexpected idempotency key %q", publisher.key)
			}
		})
	}
}
