package inputs

type Kind int

const (
	Link Kind = iota
	File
)

type Input struct {
	Raw      string
	Kind     Kind
	Platform string
	URL      string
	VideoID  string
	Path     string
	Name     string
	Size     int64
}
