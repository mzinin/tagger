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
        error := parseOggTagPictureField(fieldValue, &tag.Cover)
        if error != nil {
            return error
        }
    }

    return nil
}

func parseOggTagPictureField(encoded string, cover *Cover) error {
    data, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return err
    }
    return parseVorbisTagPictureField(data, cover)
}

func parseVorbisTagPictureField(data []byte, cover *Cover) error {
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

func getUnsupportedVorbisTags(data []byte) ([]byte, int) {
    result := make([]byte, len(data))
    fields := 0
    size := 0

    numberOfFields := utils.ReadInt32Le(data[0 : 4])
    data = data[4:]

    for i := 0; i < numberOfFields; i++ {
        fieldSize := utils.ReadInt32Le(data[0 : 4])
        if len(data) < fieldSize + 4 {
            break;
        }
            
        pos := bytes.IndexByte(data[4 : 4 + fieldSize], 0x3d)
        if pos == -1 {
            break;
        }

        fieldName := strings.ToUpper(string(data[4 : 4 + pos]))
        switch fieldName {
        case "TITLE", "ARTIST", "ALBUM", "TRACKNUMBER", "DATE", "GENRE", "METADATA_BLOCK_PICTURE":
            break
        default:
            copy(result[size : size + 4 + fieldSize], data[: 4 + fieldSize])
            fields++
            size += 4 + fieldSize
        }

        data = data[4 + fieldSize:]
    }

    return result[:size], fields
}

func serializeVorbisTag(tag Tag, existingFields int) ([]byte, int) {
    // 256 bytes for possible overhead, 2* - for base64 cover encoding
    result := make([]byte, 2*tag.Size() + 256)
    size := 0
    
    if len(tag.Title) != 0 {
        size = serializeVorbisTagTextField(tag.Title, "TITLE", result, size)
        existingFields++
    }
    if len(tag.Artist) != 0 {
        size = serializeVorbisTagTextField(tag.Artist, "ARTIST", result, size)
        existingFields++
    }
    if len(tag.Album) != 0 {
        size = serializeVorbisTagTextField(tag.Album, "ALBUM", result, size)
        existingFields++
    }
    if tag.Track != 0 {
        size = serializeVorbisTagTextField(strconv.Itoa(tag.Track), "TRACKNUMBER", result, size)
        existingFields++
    }
    if tag.Year != 0 {
        size = serializeVorbisTagTextField(strconv.Itoa(tag.Year), "DATE", result, size)
        existingFields++
    }
    if len(tag.Genre) != 0 {
        size = serializeVorbisTagTextField(tag.Genre, "GENRE", result, size)
        existingFields++
    }
    if !tag.Cover.Empty() {
        data := serializeOggTagPictureField(tag.Cover)
        size = serializeVorbisTagTextField(data, "METADATA_BLOCK_PICTURE", result, size)
        existingFields++
    }

    // empty tag is not allowed
    if existingFields == 0 {
        size = serializeVorbisTagTextField("", "LYRICS", result, size)
        existingFields++
    }

    return result[:size], existingFields
}

func serializeVorbisTagTextField(text, frameName string, dst []byte, offset int) int {
    fieldSize := len(frameName) + len(text) + 1
    utils.WriteInt32Le(fieldSize, dst[offset : offset + 4])
    copy(dst[offset + 4 : offset + 4 + len(frameName)], frameName)
    dst[offset + 4 + len(frameName)] = 0x3d // '='
    copy(dst[offset + 5 + len(frameName) : offset + 4 + fieldSize], text)
    return offset + 4 + fieldSize
}

func serializeOggTagPictureField(cover Cover) string {
    return base64.StdEncoding.EncodeToString(serializeVorbisTagPictureField(cover))
}

func serializeVorbisTagPictureField(cover Cover) []byte {
    result := make([]byte, cover.Size() + 128)
    size := 0

    // cover type
    // TODO care for cover type
    utils.WriteInt32Be(3, result[0 : 4])
    size += 4

    // mime
    utils.WriteInt32Be(len(cover.Mime), result[size : size + 4])
    copy(result[size + 4 : size + 4 + len(cover.Mime)], cover.Mime)
    size += 4 + len(cover.Mime)

    // description
    utils.WriteInt32Be(len(cover.Description), result[size : size + 4])
    copy(result[size + 4 : size + 4 + len(cover.Description)], cover.Description)
    size += 4 + len(cover.Description)

    // width, height, colour depth, colours
    for i := size; i < size + 16; i++ {
        result[i] = 0
    }
    size += 16

    // image data
    utils.WriteInt32Be(len(cover.Data), result[size : size + 4])
    copy(result[size + 4 : size + 4 + len(cover.Data)], cover.Data)
    size += 4 + len(cover.Data)

    return result[:size]
}