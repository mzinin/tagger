package editor


type Editor interface {
    ReadTag(path string) (Tag, error)
    WriteTag(src, dst string, tag Tag) error
}

type EditorType int

const (
    Mp3 EditorType = iota
    Ogg
    Flac
)

func NewEditor(editorType EditorType) Editor {
    switch editorType {
    case Mp3:
        return &Mp3TagEditor{}
    case Ogg:
        return &OggTagEditor{}
    case Flac:
        return &FlacTagEditor{}
    }
    return nil
}