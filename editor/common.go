package editor

import (
    "bytes"
    "encoding/base64"
    "errors"
    "strconv"
    "strings"

    "github.com/mzinin/tagger/utils"
)

func parseVorbisTags(data []byte, tag *Tag) error {
    if len(data) < 4 {
        return errors.New("vorbis data is too short to contain a tag")
    }

    numberOfFields := utils.ReadInt32Le(data[0:4])
    data = data[4:]

    for i := 0; i < numberOfFields; i++ {
        fieldSize := utils.ReadInt32Le(data[0:4])
        if fieldSize + 4 <= len(data) {
            parseVorbisTagField(data[4 : 4 + fieldSize], tag)
            data = data[4 + fieldSize:]
        }
    }

    return nil
}

func parseVorbisTagField(data []byte, tag *Tag) error {
    // search symbol '='
    pos := bytes.IndexByte(data, 0x3d)
    if pos == -1 {
        return errors.New("vorbis tag field is bad formatted")
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
        error := parseOggPictureTag(fieldValue, &tag.Cover)
        if error != nil {
            return error
        }
    }

    return nil
}

func parseOggPictureTag(encoded string, cover *Cover) error {
    data, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return err
    }
    if len(data) < 8 {
        return errors.New("ogg picture data is too short to contain a picture")
    }

    return parseVorbisPictureTag(data, cover)
}

func parseVorbisPictureTag(data []byte, cover *Cover) error {
    if len(data) < 4 {
        return errors.New("vorbis picture data is too short to contain a picture")
    }

    cover.Type = imageType[byte(utils.ReadInt32Be(data[0:4]))]

    mimeTypeSize := utils.ReadInt32Be(data[4 : 8])
    if len(data) < 8 + mimeTypeSize + 4 {
        return errors.New("vorbis picture data is incomplete")
    }
    cover.Mime = string(data[8 : 8 + mimeTypeSize])

    descriptionSize := utils.ReadInt32Be(data[8 + mimeTypeSize : 8 + mimeTypeSize + 4])
    if len(data) < 8 + mimeTypeSize + 4 + descriptionSize + 20 {
        return errors.New("vorbis picture data is incomplete")
    }
    cover.Description = string(data[8 + mimeTypeSize + 4 : 8 + mimeTypeSize + 4 + descriptionSize])

    cover.Data = make([]byte, len(data) - (8 + mimeTypeSize + 4 + descriptionSize + 20))
    copy(cover.Data, data[8 + mimeTypeSize + 4 + descriptionSize + 20:])

    return nil
}

var imageType = map[byte]string {
    0: "Other", 1:  "32x32 file icon", 2: "Other file icon", 3: "Cover (front)", 4: "Cover (back)",
    5: "Leaflet page", 6: "Media", 7: "Lead artist/lead performer/soloist", 8: "Artist/performer",  9: "Conductor",
    10: "Band/Orchestra", 11: "Composer", 12: "Lyricist/text writer", 13: "Recording Location", 14: "During recording",
    15: "During performance", 16: "Movie/video screen capture", 17: "A bright coloured fish", 18: "Illustration", 19: "Band/artist logotype",
    20: "Publisher/Studio logotype",
}
