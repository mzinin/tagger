package recognizer

import (
    "errors"
    "io/ioutil"
    "net/http"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "sync"

    "github.com/mzinin/tagger/utils"
)

const (
    fpUtilVersion = "1.3.1"
)

var (
    once sync.Once
)

func getFingerPrint(path string) (string, int, error) {
    once.Do(getFpUtil)

    output, err := exec.Command(pathToFpUtil(), path).Output()
    if err != nil {
        return "", 0, err
    }

    fingerPrint, duration := parseFpcalcOutput(output)

    if len(fingerPrint) == 0 {
        return "", 0, errors.New("empty fingerprint")
    }
    if duration == 0 {
        return "", 0, errors.New("zero duration")
    }

    return fingerPrint, duration, nil
}

func parseFpcalcOutput(data []byte) (string, int) {
    var fingerPrint string
    var duration int

    values := strings.Split(string(data), "\n")
    for _, value := range values {
        tokens := strings.Split(strings.Trim(value, "\r\n"), "=")
        if len(tokens) != 2 {
            continue
        }
        switch tokens[0] {
        case "DURATION":
            duration, _ = strconv.Atoi(tokens[1])
        case "FINGERPRINT":
            fingerPrint = tokens[1]
        }
    }

    return fingerPrint, duration
}

func getFpUtil() {
    // fingerprint util is here, do nothing
    if _, err := os.Stat(pathToFpUtil()); err == nil {
        return
    }

    // download fingerprint util
    url := urlToFpUtil()
    response, err := http.Get(url)
    if err != nil || response.StatusCode != 200 {
        utils.Log(utils.ERROR, "recognizer.getFpUtil: failed to download fingerprint util from '%v': %v", url, err)
        return
    }

    content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
        utils.Log(utils.ERROR, "recognizer.getFpUtil: failed to read content of '%v': %v", url, err)
		return
	}

    // extract fingerprint util
    extractFpUtil(content)
}

func pathToFpUtil() string {
    return "./" + fpUtil()
}
