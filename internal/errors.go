package internal

import (
	"errors"
)

var (
	errScheduleAlreadyExists = errors.New("schedule already exists")
	errScheduleNotFound      = errors.New("schedule not found")
)
