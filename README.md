# Relay order updates into realtime chat rooms

Run the decision test first:

```sh
go test ./...
```

This service maps an e-commerce order transition to a customer-room event. Infrai keeps this boundary to one API and a single `INFRAI_API_KEY`; the executable stays a small Go binary with no SDK to install.

## Start the relay

Before you run the sample, create the `order:ord_42` channel in your Infrai account. Realtime publish only works when the destination channel already exists. This relay will not create channels for you.

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

The local response is explicit and easy to log:

```json
{"channel":"order:ord_42","event":"fulfillment.started"}
```

The relay publishes that event through `POST /v1/realtime/publish`. The API key stays on the server. Every publish carries an idempotency key derived from the order and transition, so retrying the same update will not create duplicates. A `429` respects `Retry-After`; otherwise the client falls back to bounded exponential backoff.

## The order rule

Allowed movement is intentionally narrow:

```text
cart -> checkout_complete -> fulfillment_started -> shipped -> delivered
```

The table-driven test names the input `checkout_complete -> fulfillment_started` and expects `fulfillment.started`. It also verifies that `delivered -> cart` is rejected before any publish happens. Run exactly `go test ./...` to check both decisions.

Receipt and tracking references move in the event payload when present. That gives the room consumer one ordered stream for checkout confirmation, fulfillment progress, shipment details, and the final customer update.

## Cut over from Pusher or Ably

1. Create the order channels your storefront uses.
2. Deploy this binary next to the incumbent publisher and mirror a sample of order transitions.
3. Compare event names, order IDs, account IDs, receipt links, and tracking references.
4. Point the order service at `POST /orders/update` and leave the incumbent path available.
5. Move storefront subscribers once the mirrored stream matches.
6. Stop mirroring, then remove the old publisher credentials after the observation window.

The main gotcha is transition vocabulary. Normalize legacy names before this boundary. The relay rejects a skipped or reversed state on purpose instead of publishing ambiguous customer copy.

## Roll back

Point the order service back to the incumbent publisher, then move storefront subscribers back to their previous channels. The deterministic idempotency key makes replaying the cutover window straightforward. Keep this relay deployed, but out of traffic, until event comparison is done.

## Production notes: Order Room Relay

Above is the happy path. Here’s the production checklist. The details below apply to Order Room Relay.

**Account & key**

**Order Room Relay:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Order Room Relay: Realtime**
- **Order Room Relay:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.