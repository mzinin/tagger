// +build linux

package recognizer

import (
    "archive/tar"
    "bytes"
    "compress/gzip"
    "io"
    "os"
    "runtime"

    "github.com/mzinin/tagger/utils"
)

func fpUtil() string {
    return "fpcalc"
}

func urlToFpUtil() string {
    switch runtime.GOARCH {
    case "386":
        return "https://bitbucket.org/acoustid/chromaprint/downloads/chromaprint-fpcalc-" + fpUtilVersion + "-linux-i686.tar.gz"
    case "amd64":
        return "https://bitbucket.org/acoustid/chromaprint/downloads/chromaprint-fpcalc-" + fpUtilVersion + "-linux-x86_64.tar.gz"
    }
    
    utils.Log(utils.ERROR, "recognizer.urlToFpUtil: unsupported architecture: %v", runtime.GOARCH)
    return ""
}

func extractFpUtil(archive []byte) {
    gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
        utils.Log(utils.ERROR, "recognizer.extractFpUtil: failed to ungzip fingerprint util archive: %v", err)
		return
	}

    tarReader := tar.NewReader(gzipReader)

    for {
		header, err := tarReader.Next()
		if err == io.EOF  || err != nil {
			break
		}
        if header.Typeflag != tar.TypeReg ||
           len(header.Name) < len(fpUtil()) ||
           header.Name[len(header.Name) - len(fpUtil()):] != fpUtil() {
            continue
        }

		dst, err := os.OpenFile(pathToFpUtil(), os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0755)
		if err != nil {
            utils.Log(utils.ERROR, "recognizer.extractFpUtil: failed to open file '%v' for writing: %v", pathToFpUtil(), err)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, tarReader); err != nil {
            utils.Log(utils.ERROR, "recognizer.extractFpUtil: failed to save fingerprint util to '%v': %v", pathToFpUtil(), err)
			return
		}
	}
}
