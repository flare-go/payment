package enum

type SubscriptionStatus string

const (
	SubscriptionStatusActive            SubscriptionStatus = "ACTIVE"
	SubscriptionStatusCanceled          SubscriptionStatus = "CANCELED"
	SubscriptionStatusIncomplete        SubscriptionStatus = "INCOMPLETE"
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "INCOMPLETE_EXPIRED"
	SubscriptionStatusPastDue           SubscriptionStatus = "PAST_DUE"
	SubscriptionStatusTrialing          SubscriptionStatus = "TRIALING"
	SubscriptionStatusUnpaid            SubscriptionStatus = "UNPAID"
)
