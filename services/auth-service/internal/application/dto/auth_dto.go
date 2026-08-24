package dto

import "time"

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type LoginInput struct {
	Username string
	Password string
}

type ValidateTokenInput struct {
	Token string
}

type ValidateTokenOutput struct {
	Valid    bool
	UserID   string
	Username string
	Role     string
}

type UserOutput struct {
	ID        string
	Username  string
	Email     string
	Role      string
	CreatedAt time.Time
}

type AuthTokenOutput struct {
	Token     string
	ExpiresIn int64
	UserID    string
	Username  string
}
