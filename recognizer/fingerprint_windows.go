// +build windows

package recognizer

import (
    "archive/zip"
    "bytes"
    "io"
    "os"
    "runtime"

    "github.com/mzinin/tagger/utils"
)

func fpUtil() string {
    return "fpcalc.exe"
}

func urlToFpUtil() string {
    switch runtime.GOARCH {
    case "386":
        return "https://bitbucket.org/acoustid/chromaprint/downloads/chromaprint-fpcalc-" + fpUtilVersion + "-win-i686.zip"
    case "amd64":
        return "https://bitbucket.org/acoustid/chromaprint/downloads/chromaprint-fpcalc-" + fpUtilVersion + "-win-x86_64.zip"
    }
    
    utils.Log(utils.ERROR, "recognizer.urlToFpUtil: unsupported architecture: %v", runtime.GOARCH)
    return ""
}

func extractFpUtil(archive []byte) {
    zipReader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
        utils.Log(utils.ERROR, "recognizer.extractFpUtil: failed to unzip fingerprint util archive: %v", err)
		return
	}

    for _, file := range zipReader.File {
        if len(file.Name) < len(fpUtil()) || file.Name[len(file.Name) - len(fpUtil()):] != fpUtil() {
            continue
        }

        src, err := file.Open()
		if err != nil {
            utils.Log(utils.ERROR, "recognizer.extractFpUtil: failed to extract fingerprint util from archive: %v", err)
			return
		}
		defer src.Close()

		dst, err := os.OpenFile(pathToFpUtil(), os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0755)
		if err != nil {
            utils.Log(utils.ERROR, "recognizer.extractFpUtil: failed to open file '%v' for writing: %v", pathToFpUtil(), err)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
            utils.Log(utils.ERROR, "recognizer.extractFpUtil: failed to save fingerprint util to '%v': %v", pathToFpUtil(), err)
			return
		}

        break
    }
}
