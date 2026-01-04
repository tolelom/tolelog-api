package service

import "errors"

var (
	// Auth errors
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")

	// Post errors
	ErrPostNotFound     = errors.New("post not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNoFieldsToUpdate = errors.New("no fields to update")
)
