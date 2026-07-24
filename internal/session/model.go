package session

import "time"

type Session struct {
	AccessToken string    `json:"access_token"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
}
