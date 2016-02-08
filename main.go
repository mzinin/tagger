package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "path/filepath"
    "os"
    "strings"

    "github.com/mzinin/tagger/editor"
)

func main() {
    var result bool

    if len(os.Args) > 1 {
        result = testOnCustomFile(os.Args[1])
    } else {
        mp3Result := testTagReadWrite(editor.Mp3, "barefoot.mp3")
        oggResult := testTagReadWrite(editor.Ogg, "barefoot.ogg")
        flacResult := testTagReadWrite(editor.Flac, "loser.flac")
        result = mp3Result && oggResult && flacResult
    }

    if result {
        fmt.Println("OK")
    } else {
        fmt.Println("FAILED!!!")
    }
}

func testOnCustomFile(path string) bool {
    var tagType editor.EditorType
    switch filepath.Ext(path) {
    case ".mp3":
        tagType = editor.Mp3
    case ".ogg":
        tagType = editor.Ogg
    case ".flac":
        tagType = editor.Flac
    }

    return testTagReadWrite(tagType, path);
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

func saveCover(cover editor.Cover, path string) error {
    extension := ""
    if strings.Contains(cover.Mime, "jpg") || strings.Contains(cover.Mime, "jpeg") {
        extension = ".jpg"
    } else if strings.Contains(cover.Mime, "png") {
        extension = ".png"
    }

    return ioutil.WriteFile(path + extension, cover.Data, os.ModePerm)
}