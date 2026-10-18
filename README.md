# transcript1

From your terminal, transcript1 transcribes links from YouTube, X, TikTok, Instagram, Facebook and Bluesky, and local
audio or video files.
Read the [CLI docs](https://transcript1.com/docs/cli) for setup, batch inputs and output options.

## Install

macOS and Linux:

```sh
curl -fsSL https://transcript1.com/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://transcript1.com/install.ps1 | iex
```

On macOS, the binary is not notarized. If you download an archive in a browser, remove the quarantine flag before
running it:

```sh
xattr -d com.apple.quarantine transcript1
```

## Sign in

```sh
transcript1 login
```

`transcript1 login` opens transcript1.com, or prints the link over SSH or without a display. Compare the terminal's code
with the page and click Approve. The CLI waits for approval and saves a key named `transcript1 CLI (<computer name>)`.
It appears on the [API keys page](https://transcript1.com/dashboard/account/tokens). Use `--no-browser` to print the link without
opening a browser. To paste a key, run `transcript1 login --with-key`. Piped input still works: `transcript1 login < key.txt`.

For CI and agents, set `TRANSCRIPT1_API_KEY`; it takes priority over the saved key. An agent without a terminal or a key
can run `transcript1 login < /dev/null` and send you the link to approve. `transcript1 logout` revokes the CLI key and
removes it from your computer. If revocation fails, it removes the saved key and asks you to revoke it on the API keys
page. Pasted keys must be revoked on that page.

## Use

```sh
transcript1 'https://youtu.be/dQw4w9WgXcQ'
transcript1 'https://x.com/jack/status/20'
transcript1 -f srt -o interview.srt interview.mp3
transcript1 -o transcripts/ ./recordings
```

Links from YouTube, X, TikTok, Instagram, Facebook and Bluesky work, including short share links. Only public posts can
be fetched. If a post has several videos, the first one is transcribed. `--source captions` works with YouTube links only.

By default, one input prints to stdout. Several inputs save one file each in `./transcripts`. Run `transcript1 --help`
(or `transcript1` alone) for every flag. `t1` is a short alias for `transcript1`.

File patterns like `recordings/*.mp3` work, also on Windows.

Tip: in zsh, quote links. An unquoted `&` splits the command, and `?` is a glob unless you use the alias the installer adds.

Use `--json` to print one JSON line per input to stdout, with no progress or summary:

```sh
transcript1 --json 'https://youtu.be/dQw4w9WgXcQ'
```

Run `transcript1 get <id>` to fetch an earlier transcription.

## Update

```sh
transcript1 update
```

Versions 1.1.0 and older accept YouTube links only. Update to use the other platforms.

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

Set up releases once:

1. Cloudflare: create the R2 bucket `transcript1-downloads` with the custom domain `downloads.transcript1.com`. Have R2
   credentials, or `wrangler login`, in the shell that runs releases.
2. `go run ./cmd/release-tool keygen <path outside the repo>`. Commit `internal/update/release-key.pub`, back up the
   private key, and keep its path in `TRANSCRIPT1_RELEASE_KEY_FILE`.
3. Optional: create the GitHub repo `transcript1-com/cli` and run `gh auth login` for the mirror.
4. Run `scripts/release.sh <version>`, then deploy the site.

`scripts/release.sh <version>` (strictly `x.y.z`) checks the working tree, runs vet and tests, builds the five targets,
packages them, signs `latest.json`, uploads to R2 and verifies the public URLs. release.sh tags locally; push master and
`v<version>` (`git push origin master v<version>`), then run `scripts/mirror-release-github.sh <version>`, which creates
the GitHub release from the pushed tag when `gh` is installed. release.sh also tries the mirror at the end; until the
tag is pushed, that attempt fails with a warning and the same retry command. The R2 release stays published. A dry run
never calls `gh`.

If a release stops after uploading the archives but before uploading `latest.json`, a re-run refuses to overwrite the
archives. Run
`R2_RELEASE_RESUME=1 scripts/release.sh <version>` to check the uploaded files against the local ones and publish
`latest.json`. It reuses the archives in `dist/<version>` and does not rebuild.

| Variable | Meaning |
|---|---|
| `TRANSCRIPT1_RELEASE_KEY_FILE` | path of the private release key (required) |
| `TRANSCRIPT1_RELEASE_PUB_FILE` | dry runs only: public key to check against instead of `internal/update/release-key.pub` |
| `R2_RELEASE_UPLOADER` | `aws` (default) or `wrangler` |
| `R2_RELEASE_BUCKET` | default `transcript1-downloads` |
| `R2_RELEASE_BASE_URL` | default `https://downloads.transcript1.com` |
| `R2_RELEASE_ENDPOINT` | R2 S3 endpoint, required for `aws` |
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | R2 credentials for `aws` |
| `R2_RELEASE_DRY_RUN=1` | build, package and sign, print the uploads, skip R2 and the tag |
| `R2_RELEASE_RESUME=1` | skip the versioned uploads, verify them, publish `latest.json` |
| `GITHUB_MIRROR_REPO` | default `transcript1-com/cli`, used by the mirror script |

## License

MIT
