package output

type Mode int

const (
	TTY Mode = iota
	Plain
	Quiet
	JSON
)

const (
	green  = "\x1b[32m"
	red    = "\x1b[31m"
	yellow = "\x1b[33m"
	reset  = "\x1b[0m"
)
