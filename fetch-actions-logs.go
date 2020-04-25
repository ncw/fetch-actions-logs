// Get the logs from GitHub Actions
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Flags
	actor      = flag.String("user", "", "Fetch logs for the user specified")
	branch     = flag.String("branch", "master", "Fetch logs for the branch specified")
	event      = flag.String("event", "", "Fetch logs for the event specified (eg push, pull_request, issue)")
	status     = flag.String("status", "completed", "Fetch logs for the status specified (eg completed)")
	conclusion = flag.String("conclusion", "neutral", "Fetch logs for the status specified (eg success, failure, neutral, cancelled, timed_out, or action_required)")
	apiUser    = flag.String("api-user", os.Getenv("GITHUB_USER"), "API key user (or set GITHUB_USER)")
	apiToken   = flag.String("api-token", os.Getenv("GITHUB_TOKEN"), "API key password (or set GITHUB_TOKEN)")
	// Globals
	matchProject = regexp.MustCompile(`^([\w-]+)/([\w-]+)$`)
	project      string
	outputDir    string
	baseURL      = "https://api.github.com/repos/"
	errors       int
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s [options] <project> <directory>

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
`, os.Args[0])
	flag.PrintDefaults()
}

// read the body or an error message
func readBody(in io.Reader) string {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return fmt.Sprintf("Error reading body: %v", err.Error())
	}
	return string(data)
}

// Parses a Link into a URL returns "" on failure
func parseLink(in string) (url string) {
	url = in
	leftTriangle := strings.IndexRune(url, '<')
	if leftTriangle < 0 {
		return ""
	}
	url = url[leftTriangle+1:]
	rightTriangle := strings.IndexRune(url, '>')
	if rightTriangle < 0 {
		return ""
	}
	url = url[:rightTriangle]
	return url
}

// Do an HTTP transaction based on the URL path, adding auth if set
func doRequest(method, url string) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make http request %q: %w", url, err)
	}
	if *apiUser != "" && *apiToken != "" {
		req.SetBasicAuth(*apiUser, *apiToken)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info %q: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error: %s", readBody(resp.Body))
		return nil, fmt.Errorf("bad status %d when fetching %q release info: %s", resp.StatusCode, url, resp.Status)
	}
	return resp, nil
}

// Get the workflows for the project
func getWorkflows(project string) error {
	parameters := url.Values{}
	if *actor != "" {
		parameters.Add("actor", *actor)
	}
	if *branch != "" {
		parameters.Add("branch", *branch)
	}
	if *event != "" {
		parameters.Add("event", *event)
	}
	if *status != "" {
		parameters.Add("status", *status)
	}
	url := baseURL + project + "/actions/runs"
	if len(parameters) > 0 {
		url += "?" + parameters.Encode()
	}

	var total = 0
	for {
		log.Printf("Fetching workflows info for %q from %q", project, url)
		resp, err := doRequest("GET", url)
		if err != nil {
			return fmt.Errorf("failed to get workflow: %w", err)
		}
		var runs WorkflowRuns
		err = json.NewDecoder(resp.Body).Decode(&runs)
		if err != nil {
			return fmt.Errorf("failed to decode runs info: %w", err)
		}
		err = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("Failed to close body: %w", err)
		}
		for _, run := range runs.WorkflowRuns {
			examineRun(&run)
		}
		total += len(runs.WorkflowRuns)
		log.Printf("Fetched %d/%d workflow runs", total, runs.TotalCount)
		// this is really annoying to parse - there is probably a better way!
		url = ""
		if links := resp.Header.Get("Link"); links != "" {
			for _, link := range strings.Split(links, ",") {
				// log.Printf("link = %q", link)
				if strings.Contains(link, `rel="next"`) {
					url = parseLink(link)
				}
			}
		}
		if url == "" {
			break
		}
	}
	return nil
}

// get a file for download
func getFile(url, fileName string) error {
	log.Printf("Downloading %q from %q", fileName, url)
	tmpFile := fileName + ".tmp"

	resp, err := doRequest("GET", url)
	if err != nil {
		return fmt.Errorf("download file request failed: %w", err)
	}

	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", tmpFile, err)
	}

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error while downloading: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to close body: %w", err)
	}
	err = out.Close()
	if err != nil {
		return fmt.Errorf("failed to close output file: %w", err)
	}

	err = os.Rename(tmpFile, fileName)
	if err != nil {
		return fmt.Errorf("failed to rename output file: %w", err)
	}

	log.Printf("Downloaded %q (%d bytes)", fileName, n)
	return nil
}

func examineRun(run *WorkflowRun) {
	if *conclusion != "" && *conclusion != run.Conclusion {
		return
	}
	log.Printf("Found Job ID = %d, Status = %q, Conclusion = %q, HeadBranch = %q\n", run.ID, run.Status, run.Conclusion, run.HeadBranch)
	ID := strconv.FormatInt(run.ID, 10)
	url := baseURL + project + "/actions/runs/" + ID + "/logs"
	fileName := filepath.Join(outputDir, ID+".zip")
	_, err := os.Stat(fileName)
	if err == nil {
		log.Printf("NOT Fetching log for %s as %s already exists", ID, fileName)
	} else {
		log.Printf("Fetching log for %s to %s", ID, fileName)
		err := getFile(url, fileName)
		if err != nil {
			log.Printf("Failed to download %s: %v", fileName, err)
			errors++
		}
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		usage()
		log.Fatal("Wrong number of arguments")
	}
	project, outputDir = args[0], args[1]
	if !matchProject.MatchString(project) {
		log.Fatalf("Project %q must be in form user/project", project)
	}
	err := os.MkdirAll(outputDir, 0777)
	if err != nil {
		log.Fatalf("Failed to create output dir: %v", err)
	}

	err = getWorkflows(project)
	if err != nil {
		log.Fatalf("Get workflows failed: %v", err)
	}
	if errors != 0 {
		log.Printf("%d errors fetching logs", errors)
	}
}
