package main

import "time"

type Transaction struct {
	Date   time.Time
	Text   string
	Amount float64
}

type SubscriptionStatus string

const (
	StatusActive  SubscriptionStatus = "active"
	StatusStopped SubscriptionStatus = "stopped"
)

type Subscription struct {
	Name         string
	AvgAmount    float64
	LatestAmount float64 // most recent payment amount (used for totals)
	MinAmount    float64
	MaxAmount    float64
	Transactions []Transaction
	StartDate    time.Time
	LastDate     time.Time
	TypicalDay   int // typical day of month for payment
	Status       SubscriptionStatus
}

type DateRange struct {
	Start time.Time
	End   time.Time
}
