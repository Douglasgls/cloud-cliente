package cli

import (
	"errors"
	"fmt"
)

var (
	ErrNoCommand     = errors.New("usage: cloud-client connect <token>")
	ErrUnknownCmd    = errors.New("unknown command")
	ErrMissingToken  = errors.New("missing authorization token")
)

type Command struct {
	Name  string
	Token string
	Debug bool
}

func Parse(args []string) (*Command, error) {
	var cleanArgs []string
	debug := false

	for _, arg := range args {
		if arg == "--debug" || arg == "-d" {
			debug = true
		} else {
			cleanArgs = append(cleanArgs, arg)
		}
	}

	if len(cleanArgs) == 0 {
		return nil, ErrNoCommand
	}

	cmdName := cleanArgs[0]
	switch cmdName {
	case "connect":
		if len(cleanArgs) < 2 || cleanArgs[1] == "" {
			return nil, fmt.Errorf("%w: usage: cloud-client connect <token>", ErrMissingToken)
		}
		return &Command{
			Name:  "connect",
			Token: cleanArgs[1],
			Debug: debug,
		}, nil
	case "debug":
		return &Command{
			Name:  "debug",
			Debug: true,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s. usage: cloud-client connect <token>", ErrUnknownCmd, cmdName)
	}
}
