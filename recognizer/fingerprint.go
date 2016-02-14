package recognizer

import (
    "archive/zip"
    "bytes"
    "errors"
    "io"
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
        utils.Log(utils.ERROR, "failed to download fingerprint util from '%v': %v", url, err)
        return
    }

    content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
        utils.Log(utils.ERROR, "failed to read content of '%v': %v", url, err)
		return
	}

    // extract fingerprint util
    zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
        utils.Log(utils.ERROR, "failed to unzip content of '%v': %v", url, err)
		return
	}

    for _, file := range zipReader.File {
        if len(file.Name) < len(fpUtil()) || file.Name[len(file.Name) - len(fpUtil()):] != fpUtil() {
            continue
        }

        src, err := file.Open()
		if err != nil {
            utils.Log(utils.ERROR, "failed to extract fingerprint util from '%v': %v", url, err)
			return
		}
		defer src.Close()

		dst, err := os.OpenFile(pathToFpUtil(), os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0755)
		if err != nil {
            utils.Log(utils.ERROR, "failed to open file '%v' for writing: %v", pathToFpUtil(), err)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
            utils.Log(utils.ERROR, "failed to save fingerprint util to '%v': %v", pathToFpUtil(), err)
			return
		}

        break
    }

}

func fpUtil() string {
    return "fpcalc.exe"
}

func pathToFpUtil() string {
    return "./" + fpUtil()
}

func urlToFpUtil() string {
    return "https://bitbucket.org/acoustid/chromaprint/downloads/chromaprint-fpcalc-" + fpUtilVersion + "-win-x86_64.zip"
}