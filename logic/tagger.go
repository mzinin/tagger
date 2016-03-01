package logic

import (
    "github.com/mzinin/tagger/editor"
    "github.com/mzinin/tagger/recognizer"
    "github.com/mzinin/tagger/utils"

    "fmt"
    "os"
    "os/signal"
    "path/filepath"
    "strings"
    "sync"
    "sync/atomic"
)

type FilterType int

const (
    All FilterType = 0
    NoTag    = 1 << iota
    NoTitle  = 1 << iota
    NoArtist = 1 << iota
    NoAlbum  = 1 << iota
    NoCover  = 1 << iota
)

const (
    numberOfThreads int = 8
)

type Tagger struct {
    source string
    sourceInfo os.FileInfo
    destination string
    filter FilterType
    counter *Counter
    stop atomic.Value
}

func NewTagger(source, dest, filter string) (*Tagger, error) {
    tagger := &Tagger{}
    if err := tagger.init(source, dest, filter); err != nil {
        return nil, err
    }
    return tagger, nil
}

func (tagger *Tagger) Run() error {
    tagger.counter.start()
    tagger.trapSignal()

    var err error
    if !tagger.sourceInfo.IsDir() {
        tagger.counter.setTotal(1)
        err = tagger.processFile(tagger.source, tagger.destination)
    } else {
        err = tagger.processDir(tagger.source, tagger.destination)
    }

    tagger.counter.stop()
    return err
}

func (tagger *Tagger) PrintReport() {
    tagger.counter.printReport()
}

func (tagger *Tagger) init(source, dest, filter string) error {
    utils.Log(utils.INFO, "Initializing tagger with source = '%v', destination = '%v', filter = '%v'", source, dest, filter)

    var err error
    tagger.source, err = filepath.Abs(source)
    if err != nil {
        utils.Log(utils.ERROR, "Failed to make source path '%v' absolute: %v", source, err)
        return err
    }

    tagger.sourceInfo, err = os.Stat(tagger.source)
    if err != nil {
        utils.Log(utils.ERROR, "Failed to get source path '%v' info: %v", tagger.source, err)
        return err
    }

    if !tagger.sourceInfo.IsDir() && !isSupportedFile(tagger.source) {
        return fmt.Errorf("Input file '%v' is unsupported", tagger.source)
    }

    if len(dest) == 0 {
        tagger.destination = tagger.source
    } else {
        tagger.destination, err = filepath.Abs(dest)
        if err != nil {
            utils.Log(utils.ERROR, "Failed to make destination path '%v' absolute: %v", dest, err)
            return err
        }

        destinationInfo, err := os.Stat(tagger.destination)
        if err != nil {
            // if input is file consider destination as path to non-existent file
            if !tagger.sourceInfo.IsDir() && !isSupportedFile(tagger.destination) {
                utils.Log(utils.ERROR, "Output file '%v' is unsupported", tagger.destination)
                return fmt.Errorf("Output file '%v' is unsupported", tagger.destination)
            }
            // if input is directory consider destination as non-existent directory and try to create it
            err = os.MkdirAll(tagger.destination, 0666)
            if err != nil {
                utils.Log(utils.ERROR, "Failed to create directory '%v': %v", tagger.destination, err)
                return err
            }
        } else {
            if tagger.sourceInfo.IsDir() && !destinationInfo.IsDir() {
                return fmt.Errorf("Cannot output directory '%v' into file '%v'", tagger.source, tagger.destination)
            }
            if !tagger.sourceInfo.IsDir() && destinationInfo.IsDir() {
                tagger.destination = filepath.Join(tagger.destination, filepath.Base(tagger.source))
            }
            if !destinationInfo.IsDir() && !isSupportedFile(tagger.destination) {
                return fmt.Errorf("Output file '%v' is unsupported", tagger.destination)
            }
        }
    }

    tagger.filter, err = filterStringToType(filter)
    if err != nil {
        return err
    }

    tagger.counter = &Counter{}
    return nil
}

func (tagger *Tagger) trapSignal() {
    tagger.stop.Store(false)

    ch := make(chan os.Signal, 1)
    signal.Notify(ch, os.Interrupt)

    go func() {
        _ = <- ch
        tagger.stop.Store(true)
        utils.Log(utils.INFO, "Got signal to stop")
    } ()
}

func (tagger *Tagger) processFile(src, dst string) error {
    utils.Log(utils.INFO, "Start processing file '%v'", src)

    tagEditor := makeEditor(src)
    tag, err := tagEditor.ReadTag(src)
    if err != nil {
        tagger.counter.addFail()
        utils.Log(utils.ERROR, "Failed to read tags from file '%v': %v", src, err)
        return err
    }

    if !tagger.filterByTag(tag) {
        utils.Log(utils.INFO, "Update tag is not required for file '%v'", src)
        return nil
    }
    tagger.counter.addFiltered()

    if tagger.stop.Load().(bool) {
        utils.Log(utils.WARNING, "Processing file '%v' interrupted by application stop", src)
        return fmt.Errorf("Processing file '%v' interrupted by application stop", src)
    }

    newTag, err := recognizer.Recognize(src, tag)
    if err != nil {
        tagger.counter.addFail()
        utils.Log(utils.ERROR, "Failed to recognize composition from file '%v': %v", src, err)
        return err
    }

    if newTag.Empty() {
        tagger.counter.addFail()
        utils.Log(utils.WARNING, "Composition from file '%v' is not recognized", src)
        return nil
    }

    // if we need only cover and there is no cover, return here
    if tagger.filter == NoCover && newTag.Cover.Empty() {
        tagger.counter.addFail()
        utils.Log(utils.WARNING, "Cover for file '%v' is not found", src)
        return nil
    }

    // if we need only cover and already has smth else, take only cover
    if tagger.filter == NoCover && !tag.Empty() {
        tag.Cover = newTag.Cover
        newTag = tag
    } else {
        newTag.MergeWith(tag)
    }

    err = tagger.preparePath(dst)
    if err != nil {
        tagger.counter.addFail()
        utils.Log(utils.ERROR, "Failed to prepare path '%v': %v", dst, err)
        return err
    }

    err = tagEditor.WriteTag(src, dst, newTag)
    if err != nil {
        tagger.counter.addFail()
        utils.Log(utils.ERROR, "Failed to write tag and save file '%v': %v", dst, err)
        return err
    }

    tagger.counter.addSuccess(!newTag.Cover.Empty())
    utils.Log(utils.INFO, "File '%v' successfully processed, cover found: %v", src, newTag.Cover.Empty())
    return nil
}

func (tagger *Tagger) filterByTag(tag editor.Tag) bool {
    if tagger.filter == All {
        return true
    }
    if (tagger.filter & NoTag) != 0 && tag.Empty() {
        return true
    }
    if (tagger.filter & NoTitle) != 0 && len(tag.Title) == 0 {
        return true
    }
    if (tagger.filter & NoArtist) != 0 && len(tag.Artist) == 0 {
        return true
    }
    if (tagger.filter & NoAlbum) != 0 && len(tag.Album) == 0 {
        return true
    }
    if (tagger.filter & NoCover) != 0 && tag.Cover.Empty() {
        return true
    }
    return false
}

func (tagger *Tagger) preparePath(path string) error {
    return os.MkdirAll(filepath.Dir(path), 0666)
}

func (tagger *Tagger) processDir(src, dst string) error {
    utils.Log(utils.INFO, "Start processing directory '%v'", src)

    allFiles := getAllFiles(src)
    tagger.counter.setTotal(len(allFiles))
    utils.Log(utils.INFO, "Found %v files", len(allFiles))

    var result atomic.Value
    var index int32 = -1
    var wg sync.WaitGroup

    for i := 0; i < numberOfThreads; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                if tagger.stop.Load().(bool) {
                    utils.Log(utils.WARNING, "Processing directory '%v' interrupted by application stop", src)
                    return
                }

                i := atomic.AddInt32(&index, 1)
                if i >= int32(len(allFiles)) {
                    return
                }

                fmt.Printf("\rProcessing %v/%v", i, len(allFiles))
                destination, err := tagger.getDestinationPath(allFiles[i])
                if err != nil {
                    result.Store(err)
                    utils.Log(utils.ERROR, "Failed to get destination path: %v", err)
                    continue
                }
                if err := tagger.processFile(allFiles[i], destination); err != nil {
                    utils.Log(utils.ERROR, "Failed to process file '%v': %v", allFiles[i], err)
                    result.Store(err)
                }
            }
        } ()
    }
    wg.Wait()
    fmt.Printf("\r                        \r")

    if result.Load() != nil {
        return result.Load().(error)
    }
    return nil
}

func (tagger *Tagger) getDestinationPath(src string) (string, error) {
    if !strings.HasPrefix(src, tagger.source) {
        return "", fmt.Errorf("File not from source dir: '%v'", src)
    }
    return filepath.Join(tagger.destination, src[len(tagger.source):]), nil
}
