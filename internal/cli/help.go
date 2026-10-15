package cli

const helpText = `transcript1 turns YouTube links, audio files and video files into transcripts.

Usage:
  transcript1 [flags] <input>...   transcribe links, files or folders
  transcript1 get [flags] <id>     fetch an earlier transcription, waiting while it runs
  transcript1 login                save an API key
  transcript1 logout               remove the saved key
  transcript1 update               install the latest release
  transcript1 --version | --help

An input is a YouTube link, an audio or video file, a folder of them, or - to read
one input per line from stdin. One input prints to stdout. Several inputs are
written to ./transcripts, one file each.

Flags:
  -f, --format text|srt|vtt|json   output format (default text)
  -o, --out <path>                 output file or folder
      --speakers                   label who is speaking
      --timestamps                 put [hh:mm:ss] before each line (text only)
      --language <code>            ISO 639-1 code or auto (default auto)
      --source auto|captions|stt   where a YouTube transcript comes from (default auto)
      --jobs <n>                   inputs running at once, 1-20 (default 2)
      --fresh                      do not reuse a YouTube transcription from the last 24 hours
      --overwrite                  redo inputs whose output file already exists
      --no-wait                    start, print the id and status, and exit
      --json                       one JSON line per input on stdout
  -q, --quiet                      no progress or summary, only errors

get accepts -f, -o, --timestamps, --no-wait, --json and -q.

Examples:
  transcript1 https://youtu.be/dQw4w9WgXcQ
  transcript1 -f srt -o interview.srt interview.mp3
  transcript1 -o transcripts/ ./recordings

Sign in first with transcript1 login, or set TRANSCRIPT1_API_KEY.
`
