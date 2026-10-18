package cli

const helpText = `transcript1 transcribes YouTube links and local audio or video files.

Usage:
  transcript1 [flags] <input>...   transcribe links, files or folders
  transcript1 get [flags] <id>     fetch an earlier transcription and wait if it is still running
  transcript1 login                check and save an API key
  transcript1 logout               remove the saved key
  transcript1 update               install the latest release
  transcript1 --version | --help

Pass a YouTube link, an audio or video file, or a folder of media files.
Use - to read one input per line from stdin. By default, one input prints to
stdout. Several inputs save one file each in ./transcripts.

Flags:
  -f, --format text|srt|vtt|json   output format (default text)
  -o, --out <path>                 output file or folder
      --speakers                   add speaker labels
      --timestamps                 put [hh:mm:ss] before each line (text only)
      --language <code>            ISO 639-1 code or auto (default auto)
      --source auto|captions|stt   YouTube transcript source (default auto)
      --jobs <n>                   inputs running at once, 1-20 (default 2)
      --fresh                      do not reuse a YouTube transcription from the last 24 hours
      --overwrite                  redo inputs whose output file already exists
      --no-wait                    start, print the id and status, and exit
      --json                       one JSON line per input on stdout
  -q, --quiet                      no progress or summary, only errors

get accepts -f, -o, --timestamps, --no-wait, --json and -q.

Examples:
  transcript1 'https://youtu.be/dQw4w9WgXcQ'
  transcript1 -f srt -o interview.srt interview.mp3
  transcript1 -o transcripts/ ./recordings

Tip: in zsh, quote links that contain ? or &, or use the alias the installer adds.
t1 is a short alias for transcript1.
Sign in first with transcript1 login, or set TRANSCRIPT1_API_KEY.
`
