package main

import (
    "github.com/mzinin/tagger/logic"
    "github.com/mzinin/tagger/utils"

    "fmt"
    "os"
    "strings"
)

var (
    version string
    source string
    destination string = ""
    filter string = "ALL"
)

func parseCommandLineArguments() bool {
    if len(os.Args) < 2 {
        return false
    }
    if len(os.Args) == 2 && os.Args[1][0] != '-' {
        source = os.Args[1]
        return true
    }

    i := 1
    for i < len(os.Args) {
        switch os.Args[i] {
        case "-h", "--help":
            return false
        case "-s", "--source":
            source = os.Args[i+1]
            i += 2
        case "-d", "--destination":
            destination = os.Args[i+1]
            i += 2
        case "-f", "--filter":
            filter = strings.ToUpper(os.Args[i+1])
            i += 2
        default:
            fmt.Fprintf(os.Stderr, "Unexpected argument '%v'\n", os.Args[i])
            return false
        }
    }
    return true
}

func printUsage() {
    fmt.Printf("Usage of %v %v:\n", os.Args[0], version)
    fmt.Println("\t-h, --help          Print this message.")
    fmt.Println("\t-s, --source        Input file or directory.")
    fmt.Println("\t-d, --destination   Output file or directory, same as input by default.")
    fmt.Println("\t-f, --filter        File filter: ALL | NO_TAG | NO_TITLE | NO_TITLE_ARTIST | NO_TITLE_ARTIST_ALBUM | NO_COVER. NO_COVER by default.")
}

func main() {
    utils.Log(utils.INFO, "Starting Go Music Tagger %v", version)
    if !parseCommandLineArguments() {
        printUsage()
        return
    }

    tagger, err := logic.NewTagger(source, destination, filter)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        return
    }

    if err = tagger.Run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
    }
    tagger.PrintReport()
}
