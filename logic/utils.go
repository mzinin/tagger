package logic

import (
    "github.com/mzinin/tagger/editor"

    "fmt"
    "os"
    "path/filepath"
    "strings"
)

func isSupportedFile(file string) bool {
    switch filepath.Ext(strings.ToLower(file)) {
    case ".mp3", ".ogg", ".flac":
        return true
    }
    return false
}

func filterStringToType(filter string) (FilterType, error) {
    switch filter {
    case "ALL":
        return All, nil
    case "NO_TAG":
        return NoTag, nil
    case "NO_TITLE":
        return NoTitle, nil
    case "NO_TITLE_ARTIST":
        return NoTitle | NoArtist, nil
    case "NO_TITLE_ARTIST_ALBUM":
        return NoTitle | NoArtist | NoAlbum, nil
    case "NO_COVER":
        return NoCover, nil
    }
    return All, fmt.Errorf("Unknown filter '%v'", filter)
}

func makeEditor(file string) editor.Editor {
    switch filepath.Ext(strings.ToLower(file)) {
    case ".mp3":
        return editor.NewEditor(editor.Mp3)
    case ".ogg":
        return editor.NewEditor(editor.Ogg)
    case ".flac":
        return editor.NewEditor(editor.Flac)
    }
    return nil
}

func getAllFiles(dir string) []string {
    result := make([]string, 0, 100)
    walkFn := func(path string, info os.FileInfo, err error) error {
        if err == nil && !info.IsDir() && isSupportedFile(path) {
            result = append(result, path)
        }
        return nil
    }

    filepath.Walk(dir, walkFn);
    return result
}