package game

import (
	"github.com/pkg/errors"
)

var (
	errUnexpectedMoveStatus    = errors.New("unexpected move status")
	errInvalidSelectedPosition = errors.New("invalid selected cell position")
)
