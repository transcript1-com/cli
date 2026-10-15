# transcript1

transcript1 transcribes YouTube links and local audio or video files from your terminal.
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

`login` checks your API key before saving it. For scripts and agents, set `TRANSCRIPT1_API_KEY`; it takes priority over the
saved key. `transcript1 logout` removes the saved key.

## Use

```sh
transcript1 https://youtu.be/dQw4w9WgXcQ
transcript1 -f srt -o interview.srt interview.mp3
transcript1 -o transcripts/ ./recordings
```

By default, one input prints to stdout. Several inputs save one file each in `./transcripts`. Run `transcript1 --help`
for every flag.

Use `--json` to print one JSON line per input to stdout, with no progress or summary:

```sh
transcript1 --json https://youtu.be/dQw4w9WgXcQ
```

Run `transcript1 get <id>` to fetch an earlier transcription.

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

Set up releases once:

1. Cloudflare: create the R2 bucket `transcript1-downloads` with the custom domain `downloads.transcript1.com`. Have R2
   credentials, or `wrangler login`, in the shell that runs releases.
2. `go run ./cmd/release-tool keygen <path outside the repo>`. Commit `internal/update/release-key.pub`, back up the
   private key, and keep its path in `TRANSCRIPT1_RELEASE_KEY_FILE`.
3. Optional: create the GitHub repo `transcript1/cli` and run `gh auth login` for the mirror.
4. Run `scripts/release.sh 1.0.0`, then deploy the site.

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
| `GITHUB_MIRROR_REPO` | default `transcript1/cli`, used by the mirror script |

## License

MIT
