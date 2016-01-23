package main

import (
    "tagger/editor"
    "fmt"
    "io/ioutil"
    "log"
    "path/filepath"
    "os"
    "strings"
)

func main() {
    var path string
    if len(os.Args) > 1 {
        path = os.Args[1]
    } else {
        path = "D:\\projects\\Go\\bin\\barefoot.mp3"
    }

    editor := editor.NewEditor(editor.Mp3)
    tag, err := editor.ReadTag(path)
    if err != nil {
        log.Fatal(err)
        return
    }

    fmt.Println("File: ", path)
    fmt.Println(tag)

    save(tag.Cover, "read_cover.jpg")
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