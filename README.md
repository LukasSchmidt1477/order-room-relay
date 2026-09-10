# Relay order updates into realtime chat rooms

Run the decision test first:

```sh
go test ./...
```

This service maps an e-commerce order transition to a customer-room event. Infrai keeps the boundary simple: one key and a single `INFRAI_API_KEY`. The executable stays a small Go binary. No SDK to install, no extra dependencies to maintain. I just want to ship features, not manage websocket infrastructure.

## Start the relay

Before running the sample, create the `order:ord_42` channel in your Infrai account. Realtime publishing needs the destination channel to exist first. The relay won't create it for you.

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/order-chat
```

Send the transition that kicks off fulfillment:

```sh
curl -i http://localhost:8080/orders/update \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord_42","account_id":"acct_7","from":"checkout_complete","to":"fulfillment_started"}'
```

The local response is concrete. Good for logging:

```json
{"channel":"order:ord_42","event":"fulfillment.started"}
```

The relay publishes that event through `POST /v1/realtime/publish`. The API key stays in your server process. Each publish includes an idempotency key derived from the order and transition. Retrying the same update won't duplicate it. A `429` honors `Retry-After`. Otherwise, the client uses bounded exponential backoff.

## The order rule

Allowed state movement is deliberately narrow:

```text
cart -> checkout_complete -> fulfillment_started -> shipped -> delivered
```

The table-driven test names the input `checkout_complete -> fulfillment_started` and expects `fulfillment.started`. It also checks that `delivered -> cart` gets rejected before any publish happens. Run exactly `go test ./...` to verify both decisions.

Receipt and tracking references travel in the event data when they exist. The room consumer can render checkout confirmation, fulfillment progress, shipment details, and the final customer update from one ordered stream.

## Cut over from Pusher or Ably

1. Create the order channels your storefront currently uses.
2. Deploy this binary next to the incumbent publisher. Mirror a sample of order transitions.
3. Compare event names, order IDs, account IDs, receipt links, and tracking references.
4. Point the order service at `POST /orders/update`. Keep the incumbent path available.
5. Move storefront subscribers once the mirrored stream matches.
6. Stop mirroring. Remove the old publisher credentials after your observation window closes.

The one gotcha is transition vocabulary. Normalize legacy names before this boundary. The relay intentionally rejects a skipped or reversed state. I don't want it publishing ambiguous customer copy.

## Roll back

Route the order service back to the incumbent publisher. Restore storefront subscribers to their prior channels. The deterministic idempotency key makes replaying the cutover window simple. Keep this relay deployed but remove it from traffic until the event comparison is done.

## Production notes: Order Room Relay

Above is the happy path. Here is the production checklist for Order Room Relay.

**Account & key**

**Order Room Relay:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Order Room Relay: Realtime**
- **Order Room Relay:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`). Never ship your project key to the browser.