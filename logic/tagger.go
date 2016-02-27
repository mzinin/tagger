package logic

import (
    "github.com/mzinin/tagger/editor"
    "github.com/mzinin/tagger/recognizer"
    "github.com/mzinin/tagger/utils"

    "fmt"
    "os"
    "path/filepath"
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

type Tagger struct {
    source string
    sourceInfo os.FileInfo
    destination string
    filter FilterType
    counter *Counter
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

    var err error
    if !tagger.sourceInfo.IsDir() {
        tagger.counter.setTotal(1)
        err = tagger.processFile(tagger.source, tagger.destination)
    } else {
        // TODO
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
            // consider destination as path to non-existent file
            if !isSupportedFile(tagger.destination) {
                return fmt.Errorf("Output file '%v' is unsupported", tagger.destination)
            }
        } else {
            if tagger.sourceInfo.IsDir() && !destinationInfo.IsDir() {
                return fmt.Errorf("Cannot output directory '%v' into file '%v'", tagger.source, tagger.destination)
            }

            if !tagger.sourceInfo.IsDir() && destinationInfo.IsDir() {
                tagger.destination = filepath.Join(tagger.destination, filepath.Base(tagger.source))
            } else if !destinationInfo.IsDir() && !isSupportedFile(tagger.destination) {
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

    newTag, err := recognizer.Recognize(src)
    if err != nil {
        tagger.counter.addFail()
        utils.Log(utils.ERROR, "Failed to recognize composition from file '%v': %v", src, err)
        return err
    }

    if newTag.Empty() {
        tagger.counter.addFail()
        utils.Log(utils.ERROR, "Got empty tag for file '%v'", src)
        return nil
    }

    // if we need only cover, take only cover
    if tagger.filter == NoCover {
        tag.Cover = newTag.Cover
        newTag = tag
    } else {
        newTag.MergeWith(tag)
    }

    err = tagEditor.WriteTag(src, dst, newTag)
    if err != nil {
        tagger.counter.addFail()
        utils.Log(utils.ERROR, "Failed to write tag and save file '%v': %v", dst, err)
        return err
    }

    tagger.counter.addSuccess(!newTag.Cover.Empty())
    utils.Log(utils.INFO, "File '%v' successfully processed", src)
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
