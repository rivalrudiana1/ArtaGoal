package entity

import "time"

// PushSubscription adalah langganan Web Push milik satu user,
// didaftarkan dari browser (PushManager) untuk menerima pengingat
// tenggat meski aplikasi tidak dibuka.
type PushSubscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"created_at"`
}
