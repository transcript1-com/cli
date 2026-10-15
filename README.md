# transcript1

transcript1 transcribes audio and video from files or YouTube links. Docs: https://transcript1.com/docs/cli

## Install

macOS and Linux:

```sh
curl -fsSL https://transcript1.com/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://transcript1.com/install.ps1 | iex
```

If you download an archive in a browser on macOS, remove the quarantine flag before the first run (the binary is not
notarized):

```sh
xattr -d com.apple.quarantine transcript1
```

## Sign in

```sh
transcript1 login
```

`login` checks the key and saves it. In scripts and agents, set `TRANSCRIPT1_API_KEY` instead; it takes priority over the
saved key. `transcript1 logout` removes the saved key.

## Use

```sh
transcript1 https://youtu.be/dQw4w9WgXcQ
transcript1 -f srt -o interview.srt interview.mp3
transcript1 -o transcripts/ ./recordings
```

One input prints to stdout. Several inputs are written to `./transcripts`, one file each. `transcript1 --help` lists
every flag.

Pass `--json` for one JSON line per input on stdout, with no progress or summary:

```sh
transcript1 --json https://youtu.be/dQw4w9WgXcQ
```

Fetch an earlier transcription with `transcript1 get <id>`.

## Update

```sh
transcript1 update
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | every input succeeded or was skipped because its output file exists |
| 1 | at least one input failed or was skipped for lack of minutes |
| 2 | usage or sign-in problem |
| 130 | interrupted |

## Build from source

```sh
go build ./cmd/transcript1
```

## Releasing

One-time steps:

1. Cloudflare: create the R2 bucket `transcript1-downloads` with the custom domain `downloads.transcript1.com`. Have R2
   credentials, or `wrangler login`, in the shell that runs releases.
2. `go run ./cmd/release-tool keygen <path outside the repo>`. Commit `internal/update/release-key.pub`, back up the
   private key, and keep its path in `TRANSCRIPT1_RELEASE_KEY_FILE`.
3. Optional: create the GitHub repo `transcript1/cli` and run `gh auth login` for the mirror.
4. Run `scripts/release.sh 1.0.0`, then deploy the site.

`scripts/release.sh <version>` checks the tree, vets and tests, builds the five targets, packages them, signs
`latest.json`, uploads to R2, verifies the public URLs, tags `v<version>`, and mirrors to GitHub when `gh` is installed.
A failed mirror prints a warning and a retry command; the release stays published.

| Variable | Meaning |
|---|---|
| `TRANSCRIPT1_RELEASE_KEY_FILE` | path of the private release key (required) |
| `R2_RELEASE_UPLOADER` | `aws` (default) or `wrangler` |
| `R2_RELEASE_BUCKET` | default `transcript1-downloads` |
| `R2_RELEASE_BASE_URL` | default `https://downloads.transcript1.com` |
| `R2_RELEASE_ENDPOINT` | R2 S3 endpoint, required for `aws` |
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | R2 credentials for `aws` |
| `R2_RELEASE_DRY_RUN=1` | build, package and sign, print the uploads, skip R2 and the tag |
| `GITHUB_MIRROR_REPO` | default `transcript1/cli` |

## License

MIT
