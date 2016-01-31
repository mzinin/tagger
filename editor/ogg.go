package editor

import (
    "bytes"
    "encoding/base64"
    "errors"
    "io/ioutil"
    "strconv"
    "strings"

    "github.com/mzinin/tagger/utils"
)

const (
    oggHeaderSize int = 27
    oggHeaderMagic string = "OggS"
    oggVendorMagic = "vorbis"
    headerTypeContinue byte = 1
)

type OggTagEditor struct {
    file []byte
}

func (editor *OggTagEditor) ReadTag(path string) (Tag, error) {
    err := editor.readFile(path)
    if err != nil {
        return Tag{}, err
    }

    _, secondFrame, _ := splitFileData(editor.file)
    if len(secondFrame) == 0 {
        return Tag{}, errors.New("no tag")
    }

    tagData := extractTagData(secondFrame)
    return parseTagData(tagData)
}

func (editor *OggTagEditor) WriteTag(src, dst string, tag Tag) error {
    return nil
}

func (editor *OggTagEditor) readFile(path string) error {
    var err error
    editor.file, err = ioutil.ReadFile(path)
    return err
}

func splitFileData(data []byte) ([]byte, []byte, []byte) {
    firstFrameSize := getFrameSize(data)
    secondFrameSize := getFrameSize(data[firstFrameSize:])

    for secondFrameSize >= oggHeaderSize && data[firstFrameSize + 5] & headerTypeContinue != 0 {
        firstFrameSize += secondFrameSize
        secondFrameSize = getFrameSize(data[firstFrameSize:])
    }

    thirdHeaderSize := getFrameSize(data[firstFrameSize + secondFrameSize:])
    for thirdHeaderSize >= oggHeaderSize && data[firstFrameSize + secondFrameSize + 5] & headerTypeContinue != 0 {
        secondFrameSize += thirdHeaderSize
        thirdHeaderSize = getFrameSize(data[firstFrameSize + secondFrameSize:])
    }

    return data[:firstFrameSize], data[firstFrameSize : firstFrameSize + secondFrameSize], data[firstFrameSize + secondFrameSize:]
}

func getFrameSize(data []byte) int {
    if len(data) < oggHeaderSize {
        return 0
    }
    if string(data[0:4]) != oggHeaderMagic {
        return 0
    }

    pageSegmentsNumber := int(data[oggHeaderSize - 1])
    headerSize := oggHeaderSize + pageSegmentsNumber
    if (len(data) < headerSize) {
        return 0
    }

    pageSegmentsSize := 0
    for _, byteValue := range data[oggHeaderSize : headerSize] {
        pageSegmentsSize += int(byteValue)
    }

    return headerSize + pageSegmentsSize
}

func extractTagData(data []byte) []byte {
    var result []byte = nil

    for len(data) > oggHeaderSize {
        headerSize := oggHeaderSize + int(data[oggHeaderSize - 1])

        // it's the 1st header of the frame, the one with vendor
        if data[5] & headerTypeContinue == 0 {
            headerSize += 1 + len(oggVendorMagic)
            if len(data) < headerSize + 4 {
                return nil
            }
            headerSize += 4 + utils.ReadInt32Le(data[headerSize : headerSize + 4])
        }

        frameSize := getFrameSize(data)
        result = append(result, data[headerSize : frameSize] ...)
        data = data[frameSize:]
    }

    return result
}

func parseTagData(data []byte) (Tag, error) {
    numberOfFields := utils.ReadInt32Le(data[0 : 4])
    data = data[4:]

    var tag Tag
    for i := 0; i < numberOfFields; i++ {
        fieldSize := utils.ReadInt32Le(data[0 : 4])
        if fieldSize + 4 <= len(data) {
            parseTagField(data[4 : 4 + fieldSize], &tag)
            data = data[4 + fieldSize:]
        }
    }

    return tag, nil
}

func parseTagField(data []byte, tag *Tag) {
    // search symbol '='
    pos := bytes.IndexByte(data, 0x3d)
    if pos == -1 {
        return
    }

    fieldName := strings.ToUpper(string(data[:pos]))
    fieldValue := string(data[pos + 1:])

    switch fieldName {
    case "TITLE":
        tag.Title = fieldValue
    case "ARTIST":
        tag.Artist = fieldValue
    case "ALBUM":
        tag.Album = fieldValue
    case "TRACKNUMBER":
        tag.Track, _ = strconv.Atoi(fieldValue)
    case "DATE":
        tag.Year, _ = strconv.Atoi(fieldValue)
    case "GENRE":
        tag.Genre = fieldValue
    case "METADATA_BLOCK_PICTURE":
        tag.Cover = parseCoverTagField(fieldValue)
    }
}

func parseCoverTagField(encoded string) Cover {
    var cover Cover

    data, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil || len(data) < 8 {
        return cover
    }

    cover.Type = imageType[byte(utils.ReadInt32Be(data[0 : 4]))]

    mimeTypeSize := utils.ReadInt32Be(data[4 : 8])
    if len(data) < 8 + mimeTypeSize + 4 {
        return cover
    }
    cover.Mime = string(data[8 : 8 + mimeTypeSize])

    descriptionSize := utils.ReadInt32Be(data[8 + mimeTypeSize : 8 + mimeTypeSize + 4])
    if len(data) < 8 + mimeTypeSize + 4 + descriptionSize + 20 {
        return cover
    }
    cover.Description = string(data[8 + mimeTypeSize + 4 : 8 + mimeTypeSize + 4 + descriptionSize])

    cover.Data = make([]byte, len(data) - (8 + mimeTypeSize + 4 + descriptionSize + 20))
    copy(cover.Data, data[8 + mimeTypeSize + 4 + descriptionSize + 20:])

    return cover
}
