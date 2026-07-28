package main

import "io"

type streams struct {
	stdout io.Writer
	stderr io.Writer
}
