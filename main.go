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
    var path string
    if len(os.Args) > 1 {
        path = os.Args[1]
    } else {
        path = "D:\\projects\\Go\\bin\\barefoot.ogg"
    }

    // read tag
    editorObject := editor.NewEditor(editor.Ogg)
    tag, err := editorObject.ReadTag(path)
    if err != nil {
        log.Fatal(err)
        return
    }
    
    // print tag
    fmt.Println("File: ", path)
    fmt.Println(tag)
    save(tag.Cover, "read_cover.jpg")

    // make new tag
    var newTag editor.Tag
    newTag.Title = "Some new title"
    newTag.Artist = "Some new artist"
    newTag.Album = "Some new album"
    newTag.Track = 56
    newTag.Year = 2001
    newTag.Comment = "Here is the comment!"
    newTag.Genre = "My own genre"
    newTag.Cover.Mime = "image/png"
    newTag.Cover.Description = "a bit of description"
    newTag.Cover.Data, err = ioutil.ReadFile("D:\\Downloads\\317.png")

    // write new tag
    //err = editorObject.WriteTag(path, "D:\\projects\\Go\\bin\\new.mp3", newTag)
    //if err != nil {
    //    log.Fatal(err)
    //    return
    //}
}

func save(cover editor.Cover, path string) error {
    extension := ""
    if strings.Contains(cover.Mime, "jpg") || strings.Contains(cover.Mime, "jpeg") {
        extension = ".jpg"
    } else if strings.Contains(cover.Mime, "png") {
        extension = ".png"
    }

    filename := path[:len(path) - len(filepath.Ext(path))] + extension

    return ioutil.WriteFile(filename, cover.Data, os.ModePerm)
}