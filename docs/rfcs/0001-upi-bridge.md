# RFC 0001: UPI payments for Indian apps on RevenueCat

| | |
|---|---|
| Status | Draft, for discussion |
| Author | Pulkit Aggarwal ([@aggarwalpulkit596](https://github.com/aggarwalpulkit596)) |
| Created | 13 October 2026 |

## Summary

Indian apps that charge through UPI outside the stores can't use RevenueCat as the source of truth for who has access. Today each team writes its own glue between its payment gateway and RevenueCat.

This RFC proposes `rc-upi-bridge`, a small open-source service that turns UPI payments and UPI Autopay mandates from an Indian payment gateway into RevenueCat entitlements, so apps keep checking `customerInfo.entitlements` as they do now. For apps in Google Play's user choice billing program, it also reports each transaction to Google on time. rc-doctor would gain rules for the app-side mistakes this setup invites.

It closes with what only RevenueCat can add, revenue reporting first, because a bridge can't.

## Where UPI already works

Store billing already takes UPI, so most apps don't need any of this:

- **Google Play** has offered UPI Autopay for subscriptions in India since November 2022: the user picks "Pay with UPI" in the Play cart and approves the mandate in any UPI app that supports it ([Google, 2022](https://blog.google/intl/en-in/products/platforms/now-pay-for-subscriptions-via-upi-on-google-play/)).
- **The App Store** in India lists UPI among its payment methods ([Apple Support](https://support.apple.com/en-in/111741)).

Apps that sell through Play Billing or StoreKit with RevenueCat get UPI today, with no code change.

## The gap

The gap is UPI collected by the developer, outside store billing:

1. **Google Play user choice billing.** In India, developers can offer their own billing next to Play's. The Play service fee drops by 4% for those transactions. The developer must enroll, certify PCI DSS compliance, and report each transaction with the alternative billing APIs "within 24 hours of the payment being authorized", or self-report monthly totals by the 5th business day ([Play Console Help](https://support.google.com/googleplay/android-developer/answer/13306652)). Google is phasing manual reporting out: the backend reports each transaction through the Google Play Developer API with the token the app received ([Android Developers](https://developer.android.com/google/play/billing/alternative/alternative-billing-with-user-choice-in-app)).
2. **No SDK path.** RevenueCat's Android SDK has no dedicated APIs for user choice billing. RevenueCat's guidance is to implement it with the Play Billing Library next to the SDK, set `PurchasesAreCompletedBy.MY_APP` so purchases aren't acknowledged twice, and reconcile the alternative billing purchases with RevenueCat subscriber state on your backend ([RevenueCat's rc-alternative-billing skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat-play-billing/skills/rc-alternative-billing/SKILL.md)).
3. **Every team rebuilds the reconciliation.** Mapping gateway events to entitlements, handling mandate failures and cancellations, and meeting Google's 24-hour reporting window is the same work in every app, and it's easy to get subtly wrong.

UPI Autopay follows the RBI's e-mandate rules. Under the 2026 framework, recurring debits up to ₹15,000 go through without an extra authentication once the user has approved the mandate (up to ₹1 lakh for insurance premiums, mutual funds and credit card bills). The bank sends a pre-debit notification at least 24 hours before each debit, and users can pause, change or cancel a mandate at any time ([Business Today, April 2026](https://www.businesstoday.in/amp/personal-finance/banking/story/now-control-your-auto-debit-payments-rbis-new-rules-explained-526860-2026-04-22)). The bridge has to treat pause and cancel as ordinary events, not edge cases.

## Goals

- Access lives in RevenueCat. A paid UPI period becomes an entitlement on the RevenueCat customer, so app code, paywalls, the Customer Center and webhooks keep working unchanged.
- UPI Autopay's whole lifecycle maps to entitlements: mandate approved, debit succeeded, debit failed, mandate paused, mandate cancelled, refund.
- Apps in Play's user choice billing program report every transaction within Google's 24-hour window.
- It's small and auditable: one service, MIT-licensed, deployable as a container or a serverless function.

## Non-goals

- Being a payment gateway. Payment credentials, mandates and the RBI's notification rules stay with a licensed gateway, and PCI scope stays there.
- Revenue in RevenueCat's charts. The bridge can't provide it (see [What needs RevenueCat](#what-needs-revenuecat)).
- External payments on iOS, where Apple's rules differ by region and change often.

## Design

```
app ──(app user ID)──> gateway checkout (UPI intent or UPI Autopay mandate)
                              │ signed webhooks
                              ▼
                       rc-upi-bridge ──grant / revoke──> RevenueCat REST API v2
                              │
                              └──(user choice billing only)──> Google Play: report the transaction
app <── customerInfo.entitlements (refreshed) ── RevenueCat
```

### Checkout

The app starts the gateway's checkout with the RevenueCat app user ID in the order's metadata, so every webhook can be tied back to a RevenueCat customer. With user choice billing, the app also sends the bridge the external transaction token from the Play Billing Library (`UserChoiceDetails.externalTransactionToken`, handed to the `UserChoiceBillingListener` when the user picks the developer's billing), which Google's reporting needs.

### Webhooks to entitlements

The gateway adapter verifies each webhook's signature, drops duplicates by event ID, and turns it into one of a few events: period paid, debit failed, mandate paused, mandate cancelled, refunded. Event names and payloads differ by gateway, so each gateway gets its own adapter. The first one targets a single gateway; which one is an open question below.

For each paid period, the bridge grants the entitlement through RevenueCat's REST API v2:

```
POST /v2/projects/{project_id}/customers/{customer_id}/actions/grant_entitlement
{ "entitlement_id": "entl...", "expires_at": <end of the paid period, ms since epoch> }
```

A cancellation or refund revokes it:

```
POST /v2/projects/{project_id}/customers/{customer_id}/actions/revoke_granted_entitlement
{ "entitlement_id": "entl..." }
```

These are the calls behind `rc customers grant` and `rc customers revoke` in RevenueCat's CLI. Grants are computed from the customer's paid-through date, not from event order, so a late or repeated webhook can't shorten or double access. A failed debit leaves the current grant to run out, with a configurable grace period, rather than revoking at once.

### Reporting to Google Play

For transactions that started in Play's user choice flow, the bridge reports the transaction through the Google Play Developer API with the token from checkout, retrying until it succeeds, inside the 24-hour window. It alerts when a report is close to the deadline.

### The app after a purchase

RevenueCat caches `CustomerInfo`. After an external purchase, the app fetches fresh customer info (`getCustomerInfo` with `CacheFetchPolicy.FETCH_CURRENT` on Android) so the new entitlement shows up immediately.

### State and security

- The bridge stores, per customer: the gateway's mandate or subscription ID, the paid-through date, the status, and the IDs of processed events. Nothing else.
- Its RevenueCat secret key and the gateway's webhook secret come from the environment, and only the bridge holds them. Keys never reach the app.
- Webhooks without a valid signature are rejected; replays are dropped by event ID.

## rc-doctor rules for this setup

Proposed, not in this release. Each is detectable from code and each comes from a documented requirement:

| Rule | Finds | Source |
|---|---|---|
| `user-choice-completion` | The app calls `enableUserChoiceBilling`, but RevenueCat still completes purchases, so both may acknowledge them. | RevenueCat's rc-alternative-billing skill |
| `user-choice-reporting` | The app calls `enableUserChoiceBilling` but never reads `externalTransactionToken`, so its transactions can't be reported to Google within 24 hours. | Android Developers, Play Console Help |
| `external-purchase-refresh` | The app calls `enableUserChoiceBilling` but never fetches fresh customer info after a purchase, so a granted entitlement shows up late. | RevenueCat SDK caching |

The existing `purchases-completed-by-app` rule already checks that an app completing purchases itself acknowledges them.

## What needs RevenueCat

A bridge can grant access. It can't make RevenueCat see the money:

1. **Revenue in charts and integrations.** Promotional grants carry no price, so MRR, revenue charts, and integrations that forward revenue miss everything sold through the bridge. A server-side purchase API that takes a price, a currency, a period and a store name would let any external payment count like a store purchase, UPI included.
2. **An Indian processor in Web Billing.** RevenueCat Web Billing runs on Stripe. A Razorpay or Cashfree option would bring UPI to RevenueCat's own web checkout, with no bridge at all.
3. **User choice billing in the Android SDK.** First-class support would remove the parallel Play Billing Library code that RevenueCat's own guidance currently asks for.

## Milestones

1. This RFC, revised after feedback.
2. The bridge with one gateway adapter, entitlement grants and revokes, and tests against the gateway's test mode.
3. Reporting to Google Play for user choice billing.
4. The rc-doctor rules above.
5. A second gateway adapter.

## Open questions

- Which gateway first? The deciding factors are UPI Autopay support, webhook signing and replay protection, and the quality of the test mode.
- The default grace period after a failed debit: match the store billing grace period an app already uses, or make it explicit per app?
- Does RevenueCat want this as a community project, or would it rather build it into Web Billing?
- How should refunds of partial periods map to revocation timing?

## Sources

- Google, [Now pay for subscriptions via UPI on Google Play](https://blog.google/intl/en-in/products/platforms/now-pay-for-subscriptions-via-upi-on-google-play/) (15 November 2022).
- Google, [Play Console Help: alternative billing for users in India](https://support.google.com/googleplay/android-developer/answer/13306652).
- Google, [In-app integration guidance for alternative billing with user choice](https://developer.android.com/google/play/billing/alternative/alternative-billing-with-user-choice-in-app).
- Apple, [Payment methods you can use with your Apple Account (India)](https://support.apple.com/en-in/111741).
- RevenueCat, [rc-alternative-billing skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat-play-billing/skills/rc-alternative-billing/SKILL.md), from chapter 18 of RevenueCat's Android SDK handbook.
- RevenueCat, [rc CLI source](https://github.com/RevenueCat/cli): `internal/api/customers.go`, the grant and revoke endpoints.
- Business Today, [RBI's new rules for auto-debit payments](https://www.businesstoday.in/amp/personal-finance/banking/story/now-control-your-auto-debit-payments-rbis-new-rules-explained-526860-2026-04-22) (22 April 2026).
