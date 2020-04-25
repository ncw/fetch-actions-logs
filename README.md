# fetch-actions-logs

This is a simple program to fetch your github actions logs using the
GitHub API.

## Install

Fetch-Actions-Logs is a Go program and comes as a single binary file.

Download the relevant binary from

- https://github.com/ncw/fetch-actions-logs/releases

Or alternatively if you have Go installed use

    go get github.com/ncw/fetch-actions-logs

and this will build the binary in `$GOPATH/bin`.

## Usage
-----

Use `fetch-actions-logs -h` to see all the options.

```
Usage: fetch-actions-logs [options] <project> <directory>

Fetch the GitHub actions from project (eg rclone/rclone) into the
output directory specified. The directory will be created if not
found.

The actions are the zip files you would download from the web site
with the entire logs for the workflow run.

They are stored as JOBID.zip where JOBID is the numerical job ID.

If you re-run the program it won't download any files that are already
in the output directory.

You'll need to supply a GitHub user and api token in -api-user and
-api-token or set the environment variables GITHUB_USER and
GITHUB_TOKEN respectively.

Note that actions logs aren't stored indefinitely and after a period
attempting to fetch them will give 410 Gone errors.

Example usage:

fetch-actions-logs -conclusion failure -user ncw rclone/rclone logs

Full options:
  -api-token string
    	API key password (or set GITHUB_TOKEN)
  -api-user string
    	API key user (or set GITHUB_USER)
  -branch string
    	Fetch logs for the branch specified (default "master")
  -conclusion string
    	Fetch logs for the status specified (eg success, failure, neutral, cancelled, timed_out, or action_required) (default "neutral")
  -event string
    	Fetch logs for the event specified (eg push, pull_request, issue)
  -status string
    	Fetch logs for the status specified (eg completed) (default "completed")
  -user string
    	Fetch logs for the user specified
```

## License

This is free software under the terms of the MIT license (check the
LICENSE file included in this package).

## Contact and support

The project website is at:

- https://github.com/ncw/fetch-actions-logs

There you can file bug reports, ask for help or contribute patches.

## Authors

- Nick Craig-Wood <nick@craig-wood.com>
- Your name goes here!
