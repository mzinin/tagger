package recognizer

import (
    "errors"
    "os/exec"
    "strconv"
    "strings"
)

const (
    fpUtil string = "./fpcalc.exe"
)

func getFingerPrint(path string) (string, int, error) {
    output, err := exec.Command(fpUtil, path).Output()
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