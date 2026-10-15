package files

const (
	FormatText = "text"
	FormatSRT  = "srt"
	FormatVTT  = "vtt"
	FormatJSON = "json"
)

func Ext(format string) string {
	switch format {
	case FormatSRT:
		return ".srt"
	case FormatVTT:
		return ".vtt"
	case FormatJSON:
		return ".json"
	default:
		return ".txt"
	}
}
