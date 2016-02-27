package main

import (
    "github.com/mzinin/tagger/logic"
    "flag"
    "log"
)

var (
    version string
    source string
    destination string
    filter string
)

func parseCommandLineArguments() {
    flag.StringVar(&source, "s", ".", "input durectory or file")
    flag.StringVar(&destination, "d", "", "output durectory or file, same as input by default")
    flag.StringVar(&filter, "filter", "ALL", "file filter")

    flag.Parse()
}

func main() {
    parseCommandLineArguments()
    
    tagger, err := logic.NewTagger(source, destination, filter)
    if err != nil {
        log.Fatal(err)
        return
    }

    err = tagger.Run()
    if err != nil {
        log.Fatal(err)
        return
    }
    tagger.PrintReport()
}
