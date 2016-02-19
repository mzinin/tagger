package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "path/filepath"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/mzinin/tagger/editor"
    "github.com/mzinin/tagger/recognizer"
)

var (
    version string
)

func main() {
    var result bool

    //if len(os.Args) > 1 {
    //    result = testTagReadWrite(getType(os.Args[1]), os.Args[1])
    //} else {
    //    mp3Result := testTagReadWrite(editor.Mp3, "barefoot.mp3")
    //    oggResult := testTagReadWrite(editor.Ogg, "barefoot.ogg")
    //    flacResult := testTagReadWrite(editor.Flac, "loser.flac")
    //    result = mp3Result && oggResult && flacResult
    //}

    start := time.Now()

    if len(os.Args) > 1 {
        result = testUpdateTag(getType(os.Args[1]), os.Args[1])
    } else {
        var wg sync.WaitGroup
        wg.Add(3)
        channel := make(chan bool, 3)

        go func() {
            defer wg.Done()
            channel <- testUpdateTag(editor.Mp3, "barefoot.mp3")
        } ()

        go func() {
            defer wg.Done()
            channel <- testUpdateTag(editor.Ogg, "barefoot.ogg")
        } ()

        go func() {
            defer wg.Done()
            channel <- testUpdateTag(editor.Flac, "loser.flac")
        } ()

        wg.Wait()

        mp3Result, oggResult, flacResult := <- channel, <- channel, <- channel
        result = mp3Result && oggResult && flacResult
    }

    duration := time.Now().Sub(start)
    fmt.Println("Total time: ", duration)

    if result {
        fmt.Println("OK")
    } else {
        fmt.Println("FAILED!!!")
    }
}

func getType(path string) editor.EditorType {
    switch filepath.Ext(path) {
    case ".mp3":
        return editor.Mp3
    case ".ogg":
        return editor.Ogg
    case ".flac":
        return editor.Flac
    }
    return editor.Mp3
}

func testTagReadWrite(tagType editor.EditorType, path string) bool {
    // convert type into string
    var typeString string
    switch tagType {
    case editor.Mp3:
        typeString = "mp3"
    case editor.Ogg:
        typeString = "ogg"
    case editor.Flac:
        typeString = "flac"
    }

    // read tag
    editorObject := editor.NewEditor(tagType)
    tag, err := editorObject.ReadTag(path)
    if err != nil {
        log.Fatal(err)
        return false
    }

    // print tag
    fmt.Println(typeString + " file: ", path)
    fmt.Println(tag, "\n")
    err = saveCover(tag.Cover, "read_cover_" + typeString)
    if err != nil {
        log.Fatal(err)
        return false
    }

    // make new tag
    var newTag editor.Tag
    newTag.Title = "Some new title"
    newTag.Artist = "Some new artist"
    newTag.Album = "Some new album"
    newTag.Track = 56
    newTag.Year = 2001
    newTag.Comment = "Here is the comment!"
    newTag.Genre = "My own genre"
    newTag.Cover.Mime = "image/jpg"
    newTag.Cover.Description = "a bit of description"
    newTag.Cover.Data, err = ioutil.ReadFile("D:\\Downloads\\comix_15.jpg")

    // write new tag
    err = editorObject.WriteTag(path, "D:\\projects\\Go\\bin\\new." + typeString, newTag)
    if err != nil {
        log.Fatal(err)
        return false
    }

    return true
}

func testUpdateTag(tagType editor.EditorType, path string) bool {
    // convert type into string
    var typeString string
    switch tagType {
    case editor.Mp3:
        typeString = "mp3"
    case editor.Ogg:
        typeString = "ogg"
    case editor.Flac:
        typeString = "flac"
    }

    // read tag
    editorObject := editor.NewEditor(tagType)
    tag, err := editorObject.ReadTag(path)
    if err != nil {
        log.Fatal(err)
        return false
    }

    // get new tag
    newTag, err := recognizer.UpdateTag(tag, path, recognizer.Always)
    if err != nil {
        log.Fatal(err)
        return false
    }

    // write new tag
    err = editorObject.WriteTag(path, "D:\\projects\\Go\\bin\\new." + typeString, newTag)
    if err != nil {
        log.Fatal(err)
        return false
    }

    return true
}

func saveCover(cover editor.Cover, path string) error {
    if len(cover.Data) == 0 {
        return nil
    }

    extension := ""
    if strings.Contains(cover.Mime, "jpg") || strings.Contains(cover.Mime, "jpeg") {
        extension = ".jpg"
    } else if strings.Contains(cover.Mime, "png") {
        extension = ".png"
    }

    return ioutil.WriteFile(path + extension, cover.Data, os.ModePerm)
}