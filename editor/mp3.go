package editor

import (
    "bytes"
    "errors"
    "io/ioutil"
    "strconv"
    "strings"

    "github.com/mzinin/tagger/utils"
)

const (
    id3v1TagSize int = 128
    id3v1TagMagic string = "TAG"
    id3v2TagMagic string = "ID3"
    id3v2HeaderSize int = 10
    id3v2FrameHeaderSize int = 10
    id3v2FrameIdSize int = 4
)

type Mp3TagEditor struct {
    file []byte
}

func (editor *Mp3TagEditor) ReadTag(path string) (Tag, error) {
    err := editor.readFile(path)
    if err != nil {
        return Tag{}, err
    }

    tag10Data, tag23Data, tag24Data, _ := editor.splitFileData(editor.file)

    tag10 := editor.parseID3v1Tag(tag10Data)
    tag23 := editor.parseID3v2Tag(tag23Data, 3)
    tag24 := editor.parseID3v2Tag(tag24Data, 4)

    if tag10.Empty() && tag23.Empty() && tag24.Empty() {
        return Tag{}, errors.New("no ID3 tag")
    }

    tag23.MergeWith(tag24)
    tag23.MergeWith(tag10)

    return tag23, nil
}

func (editor *Mp3TagEditor) WriteTag(src, dst string, tag Tag) error {
    err := editor.readFile(src)
    if err != nil {
        return err
    }

    _, existingTagData, _, soundData := editor.splitFileData(editor.file)

    newTagData := editor.makeNewID3v23TagData(existingTagData, tag)
    return ioutil.WriteFile(dst, append(newTagData, soundData ...), 0666)
}

func (editor *Mp3TagEditor) readFile(path string) error {
    var err error
    editor.file, err = ioutil.ReadFile(path)
    return err
}

func (editor *Mp3TagEditor) splitFileData(data []byte) ([]byte, []byte, []byte, []byte) {
    var id3v1TagData []byte = nil
    if len(data) > id3v1TagSize && string(data[len(data) - id3v1TagSize : len(data) - id3v1TagSize + 3]) == id3v1TagMagic {
        id3v1TagData = data[len(editor.file) - id3v1TagSize:]
        data = data[:len(editor.file) - id3v1TagSize]
    }

    id3v23TagData := editor.findIDv2Data(3, data)
    id3v24TagData := editor.findIDv2Data(4, data)

    for len(data) > id3v2HeaderSize && string(data[0:3]) == id3v2TagMagic {
        tagSize := editor.readSyncInt32Be(data[6:10])
        data = data[id3v2HeaderSize + tagSize:]
    }

    return id3v1TagData, id3v23TagData, id3v24TagData, data
}

func (editor *Mp3TagEditor) findIDv2Data(version byte, data []byte) []byte {
    if len(data) < id3v2HeaderSize {
        return nil
    }
    if string(data[0:3]) != id3v2TagMagic {
        return nil
    }

    size := editor.readSyncInt32Be(data[6:10])
    if len(data) < id3v2HeaderSize + size || size == 0 {
        return nil
    }

    // wrong version, continue search
    if data[3] != version || data[4] != 0 {
        return editor.findIDv2Data(version, data[(id3v2HeaderSize + size):])
    }

    // exclude extended header if any
    extendedHeaderSize := 0
    if data[5] & 0x40 != 0 {
        switch version {
        case 3:
            extendedHeaderSize = utils.ReadInt32Be(data[11:15]) + 4
        case 4:
            extendedHeaderSize = editor.readSyncInt32Be(data[11:15])
        }
    }

    return data[(id3v2HeaderSize + extendedHeaderSize):(id3v2HeaderSize + extendedHeaderSize + size)]
}

func (editor *Mp3TagEditor) readSyncInt32Be(data []byte) int {
    return ((int(data[0]) * 0x80 + int(data[1])) * 0x80 + int(data[2])) * 0x80 + int(data[3])
}

func (editor *Mp3TagEditor) writeSyncInt32Be(size int, dst []byte) {
    if len(dst) < 4 {
        return
    }
    for i := 3; i >= 0; i-- {
        dst[i] = byte(size % 0x80)
        size = size / 0x80
    }
}

func (editor *Mp3TagEditor) parseID3v1Tag(data []byte) Tag {
    if len(data) == 0 {
        return Tag{}
    }

    var tag Tag

    tag.Title = strings.Trim(string(data[3:33]), " ")
    tag.Artist = strings.Trim(string(data[33:63]), " ")
    tag.Album = strings.Trim(string(data[63:93]), " ")
    tag.Year, _ = strconv.Atoi(string(data[93:97]))
    switch data[125] {
    case 0:
        tag.Comment = strings.Trim(string(data[97:125]), " \0000")
        tag.Track = int(data[126])
    default:
        tag.Comment = strings.Trim(string(data[97:127]), " \0000")
    }
    tag.Genre = genreCodeToString[int(data[127])]

    return tag
}

func (editor *Mp3TagEditor) parseID3v2Tag(data []byte, version int) Tag {
    if len(data) == 0 {
        return Tag{}
    }

    var tag Tag
    for len(data) > id3v2FrameHeaderSize {
        frameId := string(data[0:4])
        if frameId == "\x00\x00\x00\x00" {
            break
        }

        frameSize := 0
        switch version {
        case 3:
            frameSize = utils.ReadInt32Be(data[4:8])
        case 4:
            frameSize = editor.readSyncInt32Be(data[4:8])
        }

        editor.parseID3v2Frame(&tag, frameId, data[id3v2FrameHeaderSize:(id3v2FrameHeaderSize + frameSize)])
        data = data[id3v2FrameHeaderSize + frameSize:]
    }

    return tag
}

func (editor *Mp3TagEditor) parseID3v2Frame(tag *Tag, frameId string, frameData []byte) {
    switch frameId {
    case "APIC":
        tag.Cover = editor.readID3v2Cover(frameData)
    case "COMM":
        tag.Comment = editor.readID3v2Text(frameData)
    case "TALB":
        tag.Album = editor.readID3v2Text(frameData)
    case "TCON":
        tag.Genre = editor.readID3v2Text(frameData)
    case "TIT2":
        tag.Title = editor.readID3v2Text(frameData)
    case "TPE1":
        tag.Artist = editor.readID3v2Text(frameData)
    case "TRCK":
        tag.Track, _ = strconv.Atoi(editor.readID3v2Text(frameData))
    case "TYER":
    case "TDRC":
        tag.Year, _ = strconv.Atoi(editor.readID3v2Text(frameData))
    }
}

func (editor *Mp3TagEditor) readID3v2Text(data []byte) string {
    if len(data) == 0 {
        return ""
    }
    encoding := data[0]
    return editor.decodeText(encoding, data[1:])
}

func (editor *Mp3TagEditor) decodeText(encoding byte, data []byte) string {
    switch encoding {
    case 0, 3:
        return strings.Trim(string(data), " \x00")
    case 1:
        return utils.Utf16LeToUtf8(data)
    case 2:
        return utils.Utf16BeToUtf8(data)
    }
    return ""
}

func (editor *Mp3TagEditor) readID3v2Cover(data []byte) Cover {
    if len(data) == 0 {
        return Cover{}
    }

    var cover Cover
    encoding := data[0]
    pos := bytes.IndexByte(data[1:], 0)
    if pos != -1 {
        cover.Mime = editor.decodeText(encoding, data[1:pos + 1])
        data = data[pos + 2:]
    }

    if len(data) < 2 {
        return Cover{}
    }
    cover.Type = imageType[data[0]]

    pos = bytes.IndexByte(data[1:], 0)
    if pos != -1 {
        cover.Description = editor.decodeText(encoding, data[1:pos + 1])
        data = data[pos + 2:]
    }

    cover.Data = make([]byte, len(data))
    copy(cover.Data, data)

    return cover
}

func (editor *Mp3TagEditor) makeNewID3v23TagData(existingTagData []byte, tag Tag) []byte {
    newTags := editor.serializeTag(tag)
    oldTags := editor.getUnsupportedID3v2Tags(existingTagData)
    header := editor.makeIdv23TagHeader(len(newTags) + len(oldTags))
    return append(append(header, newTags ...), oldTags ...)
}

func (editor *Mp3TagEditor) serializeTag(tag Tag) []byte {
    // x2 for possible transform into UTF16, 256 bytes for possible overhead
    result := make([]byte, 2 * tag.Size() + 256)
    size := 0
    
    if len(tag.Title) != 0 {
        size = editor.serializeTextField(tag.Title, "TIT2", result, size)
    }
    if len(tag.Artist) != 0 {
        size = editor.serializeTextField(tag.Artist, "TPE1", result, size)
    }
    if len(tag.Album) != 0 {
        size = editor.serializeTextField(tag.Album, "TALB", result, size)
    }
    if tag.Track != 0 {
        size = editor.serializeTextField(strconv.Itoa(tag.Track), "TRCK", result, size)
    }
    if tag.Year != 0 {
        size = editor.serializeTextField(strconv.Itoa(tag.Year), "TYER", result, size)
    }
    if len(tag.Comment) != 0 {
        size = editor.serializeTextField(tag.Comment, "COMM", result, size)
    }
    if len(tag.Genre) != 0 {
        size = editor.serializeTextField(tag.Genre, "TCON", result, size)
    }
    if !tag.Cover.Empty() {
        size = editor.serializeCover(tag.Cover, "APIC", result, size)
    }

    return result[:size]
}

func (editor *Mp3TagEditor) serializeTextField(text, frameId string, dst []byte, offset int) int {
    if len(frameId) != id3v2FrameIdSize {
        return offset
    }

    utf16Text := utils.Utf8ToUtf16Le(text)
    copy(dst[offset:], frameId)
    utils.WriteInt32Be(len(utf16Text) + 3, dst[offset + 4 : offset + 8])
    dst[offset + 8] = 0 // flag
    dst[offset + 9] = 0 // flag
    dst[offset + 10] = 1 // text encoding
    dst[offset + 11] = 0xFF // UTF BOM
    dst[offset + 12] = 0xFE // UTF BOM
    copy(dst[offset + 13 : offset + 13 + len(utf16Text)], utf16Text)
    return offset + 13 + len(utf16Text)
}

func (editor *Mp3TagEditor) serializeCover(cover Cover, frameId string, dst []byte, offset int) int {
    if len(frameId) != id3v2FrameIdSize {
        return offset
    }

    data := editor.coverToData(cover)
    copy(dst[offset:], frameId)
    utils.WriteInt32Be(len(data), dst[offset + 4 : offset + 8])
    dst[offset + 8] = 0 // flag
    dst[offset + 9] = 0 // flag
    copy(dst[offset + 10 : offset + 10 + len(data)], data)

    return offset + 10 + len(data)
}

func (editor *Mp3TagEditor) coverToData(cover Cover) []byte {
    result := make([]byte, cover.Size() + 128)
    size := 0

    result[0] = 0 // encoding
    size++

    copy(result[size : size + len(cover.Mime)], cover.Mime)
    size += len(cover.Mime)

    result[size] = 0
    size++

    // TODO care for cover type
    result[size] = 3 // cover type
    size++

    copy(result[size : size + len(cover.Description)], cover.Description)
    size += len(cover.Description)

    result[size] = 0
    size++

    copy(result[size : size + len(cover.Data)], cover.Data)
    size += len(cover.Data)

    return result[:size]
}

func (editor *Mp3TagEditor) getUnsupportedID3v2Tags(existingTagData []byte) []byte {
    result := make([]byte, len(existingTagData))
    size := 0

    for len(existingTagData) > id3v2FrameHeaderSize {
        frameId := string(existingTagData[0:4])
        frameSize := utils.ReadInt32Be(existingTagData[4:8])

        stop := false
        switch frameId {
        case "\x00\x00\x00\x00":
            stop = true
        case "APIC", "COMM", "TALB", "TCON", "TIT2", "TPE1", "TRCK", "TYER", "TDRC":
            break
        default:
            copy(result[size : size + id3v2FrameHeaderSize + frameSize], existingTagData[:id3v2FrameHeaderSize + frameSize])
            size += id3v2FrameHeaderSize + frameSize
        }

        if stop {
            break
        }

        existingTagData = existingTagData[id3v2FrameHeaderSize + frameSize:]
    }

    return result[:size]
}

func (editor *Mp3TagEditor) makeIdv23TagHeader(size int) []byte {
    header := make([]byte, id3v2HeaderSize)
    header[0] = 0x49 // I
    header[1] = 0x44 // D
    header[2] = 0x33 // 3
    header[3] = 0x03
    header[4] = 0x00
    header[5] = 0x00
    editor.writeSyncInt32Be(size, header[6:id3v2HeaderSize])
    return header
}

var genreCodeToString = map[int]string {
    0: "Blues", 1: "Classic Rock", 2: "Country", 3: "Dance", 4: "Disco", 5: "Funk",
    6: "Grunge", 7: "Hip-Hop", 8: "Jazz", 9: "Metal", 10: "New Age",
    11: "Oldies", 12: "Other", 13: "Pop", 14: "R&B", 15: "Rap",
    16: "Reggae", 17: "Rock", 18: "Techno", 19: "Industrial", 20: "Alternative",
    21: "Ska", 22: "Death Metal", 23: "Pranks", 24: "Soundtrack", 25: "Euro-Techno",
    26: "Ambient", 27: "Trip-Hop", 28: "Vocal", 29: "Jazz+Funk", 30: "Fusion",
    31: "Trance", 32: "Classical", 33: "Instrumental", 34: "Acid", 35: "House",
    36: "Game", 37: "Sound Clip", 38: "Gospel", 39: "Noise", 40: "AlternRock",
    41: "Bass", 42: "Soul", 43: "Punk", 44: "Space", 45: "Meditative",
    46: "Instrumental Pop", 47: "Instrumental Rock", 48: "Ethnic", 49: "Gothic", 50: "Darkwave",
    51: "Techno-Industrial", 52: "Electronic", 53: "Pop-Folk", 54: "Eurodance", 55: "Dream",
    56: "Southern Rock", 57: "Comedy", 58: "Cult", 59: "Gangsta", 60: "Top 40",
    61: "Christian Rap", 62: "Pop/Funk", 63: "Jungle", 64: "Native American", 65: "Cabaret",
    66: "New Wave", 67: "Psychadelic", 68: "Rave", 69: "Showtunes", 70: "Trailer",
    71: "Lo-Fi", 72: "Tribal", 73: "Acid Punk", 74: "Acid Jazz", 75: "Polka",
    76: "Retro", 77: "Musical", 78: "Rock & Roll", 79: "Hard Rock", 80: "Folk",
    81: "Folk-Rock", 82: "National Folk", 83: "Swing", 84: "Fast Fusion", 85: "Bebob",
    86: "Latin", 87: "Revival", 88: "Celtic", 89: "Bluegrass", 90: "Avantgarde",
    91: "Gothic Rock", 92: "Progressive Rock", 93: "Psychedelic Rock", 94: "Symphonic Rock", 95: "Slow Rock",
    96: "Big Band", 97: "Chorus", 98: "Easy Listening", 99: "Acoustic", 100: "Humour",
    101: "Speech", 102: "Chanson", 103: "Opera", 104: "Chamber Music", 105: "Sonata",
    106: "Symphony", 107: "Booty Brass", 108: "Primus", 109: "Porn Groove", 110: "Satire",
    111: "Slow Jam", 112: "Club", 113: "Tango", 114: "Samba", 115: "Folklore",
    116: "Ballad", 117: "Poweer Ballad", 118: "Rhytmic Soul", 119: "Freestyle", 120: "Duet",
    121: "Punk Rock", 122: "Drum Solo", 123: "A Capela", 124: "Euro-House", 125: "Dance Hall",
    126: "Unknown", 127: "Unknown",
}
