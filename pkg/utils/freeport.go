package utils

import (
	"errors"
	"fmt"
	"net"
)

func FreePort(preferred int) (port int, err error) {
	if preferred != 0 {
		l, err := net.ListenTCP("tcp", &net.TCPAddr{Port: preferred})
		if err != nil {
			return 0, fmt.Errorf("failed to listen: %w", err)
		}

		defer l.Close()

		addr, ok := l.Addr().(*net.TCPAddr)
		if !ok {
			return 0, errors.New("failed to get port from listener address")
		}

		return addr.Port, nil
	}

	var a *net.TCPAddr

	a, err = net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("failed to resolve address: %w", err)
	}

	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return 0, fmt.Errorf("failed to listen: %w", err)
	}

	defer l.Close()

	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("failed to get port from listener address")
	}

	return addr.Port, nil
}
