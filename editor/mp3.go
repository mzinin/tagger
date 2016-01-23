package editor

import (
    "bytes"
    "errors"
    "io/ioutil"
    "strconv"
    "strings"
    "unicode/utf16"
    "unicode/utf8"
)

const (
    id3v2HeaderSize int = 10
    id3v2FrameHeaderSize int = 10
)

type Mp3TagEditor struct {
    file []byte
}

func (editor *Mp3TagEditor) ReadTag(path string) (Tag, error) {
    err := editor.readFile(path)
    if err != nil {
        return Tag{}, err
    }

    tag10, _ := editor.parseID3v10Tag()
    tag23, _ := editor.parseID3v23Tag()
    tag24, _ := editor.parseID3v24Tag()

    if tag10.Empty() && tag23.Empty() && tag24.Empty() {
        return Tag{}, errors.New("no ID3 tag")
    }

    aggregateTag(&tag23, tag24)
    aggregateTag(&tag23, tag10)

    return tag23, nil
}

func (editor *Mp3TagEditor) WriteTag(src, dst string, tag Tag) error {
    return nil
}

func (editor *Mp3TagEditor) readFile(path string) error {
    var err error
    editor.file, err = ioutil.ReadFile(path)
    return err
}

func (editor *Mp3TagEditor) parseID3v23Tag() (Tag, error) {
    tagData := editor.findIDv2Data(3, editor.file)
    if len(tagData) == 0 {
        return Tag{}, errors.New("no ID3v23 tag")
    }
    var tag Tag
    for len(tagData) > id3v2FrameHeaderSize {
        frameId := string(tagData[0:4])
        if frameId == "\x00\x00\x00\x00" {
            break
        }
        frameSize := readSize(tagData[4:8])
        editor.parseID3v2Frame(&tag, frameId, tagData[id3v2FrameHeaderSize:(id3v2FrameHeaderSize + frameSize)])
        tagData = tagData[id3v2FrameHeaderSize + frameSize:]
    }
    return tag, nil
}

func (editor *Mp3TagEditor) parseID3v24Tag() (Tag, error) {
    tagData := editor.findIDv2Data(4, editor.file)
    if len(tagData) == 0 {
        return Tag{}, errors.New("no ID3v24 tag")
    }
    var tag Tag
    for len(tagData) > id3v2FrameHeaderSize {
        frameId := string(tagData[0:4])
        if frameId == "\x00\x00\x00\x00" {
            break
        }
        frameSize := readSyncSize(tagData[4:8])
        editor.parseID3v2Frame(&tag, frameId, tagData[id3v2FrameHeaderSize:(id3v2FrameHeaderSize + frameSize)])
        tagData = tagData[id3v2FrameHeaderSize + frameSize:]
    }
    return tag, nil
}

func (editor *Mp3TagEditor) parseID3v2Frame(tag *Tag, frameId string, frameData []byte) {
    switch frameId {
    case "APIC":
        tag.Cover = readID3v2Cover(frameData)
    case "COMM":
        tag.Comment = readID3v2Text(frameData)
    case "TALB":
        tag.Album = readID3v2Text(frameData)
    case "TCON":
        tag.Genre = readID3v2Text(frameData)
    case "TIT2":
        tag.Title = readID3v2Text(frameData)
    case "TPE1":
        tag.Artist = readID3v2Text(frameData)
    case "TRCK":
        tag.Track, _ = strconv.Atoi(readID3v2Text(frameData))
    case "TYER":
        tag.Year, _ = strconv.Atoi(readID3v2Text(frameData))
    }
}

func (editor *Mp3TagEditor) findIDv2Data(version byte, data []byte) []byte {
    if len(data) < id3v2HeaderSize {
        return nil
    }
    if string(data[0:3]) != "ID3" {
        return nil
    }

    size := readSyncSize(data[6:10])
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
            extendedHeaderSize = readSize(data[11:15]) + 4
        case 4:
            extendedHeaderSize = readSyncSize(data[11:15])
        }
    }

    return data[(id3v2HeaderSize + extendedHeaderSize):(id3v2HeaderSize + extendedHeaderSize + size)]
}

func (editor *Mp3TagEditor) parseID3v10Tag() (Tag, error) {
    if len(editor.file) < 128 {
        return Tag{}, errors.New("no ID3v10 tag")
    }

    tagData := editor.file[len(editor.file)-128:]

    if string(tagData[:3]) != "TAG" {
        return Tag{}, errors.New("no ID3v10 tag")
    }

    var tag Tag
    tag.Title = strings.Trim(string(tagData[3:33]), " ")
    tag.Artist = strings.Trim(string(tagData[33:63]), " ")
    tag.Album = strings.Trim(string(tagData[63:93]), " ")
    tag.Year, _ = strconv.Atoi(string(tagData[93:97]))
    switch tagData[125] {
    case 0:
        tag.Comment = strings.Trim(string(tagData[97:125]), " ")
        tag.Track = int(tagData[126])
    default:
        tag.Comment = strings.Trim(string(tagData[97:127]), " ")
    }
    tag.Genre = genreCodeToString[int(tagData[127])]
    return tag, nil
}

func readSize(data []byte) int {
    return ((int(data[0]) * 0x100 + int(data[1])) * 0x100 + int(data[2])) * 0x100 + int(data[3])
}

func readSyncSize(data []byte) int {
    return ((int(data[0]) * 0x80 + int(data[1])) * 0x80 + int(data[2])) * 0x80 + int(data[3])
}

func readID3v2Text(data []byte) string {
    if len(data) == 0 {
        return ""
    }
    encoding := data[0]
    return decodeText(encoding, data[1:])
}

func decodeText(encoding byte, data []byte) string {
    switch encoding {
    case 0, 3:
        return strings.Trim(string(data), " \x00")
    case 1:
        if len(data) < 2 {
            return ""
        }
        data = data[2:]
        if len(data) % 2 != 0 {
            data = data[:len(data) - 1]
        }
        u16slice := make([]uint16, 1)
        ret := &bytes.Buffer{}
        b8buffer := make([]byte, 4)
        for i := 0; i < len(data); i += 2 {
            u16slice[0] = uint16(data[i]) + (uint16(data[i + 1]) << 8)
    		runes := utf16.Decode(u16slice)
            size := utf8.EncodeRune(b8buffer, runes[0])
    		ret.Write(b8buffer[:size])
        }
        return strings.Trim(ret.String(), " \x00")
    case 2:
        if len(data) < 2 {
            return ""
        }
        data = data[2:]
        if len(data) % 2 != 0 {
            data = data[:len(data) - 1]
        }
        u16slice := make([]uint16, 1)
        ret := &bytes.Buffer{}
        b8buffer := make([]byte, 4)
        for i := 0; i < len(data); i += 2 {
            u16slice[0] = uint16(data[i + 1]) + (uint16(data[i]) << 8)
    		runes := utf16.Decode(u16slice)
            size := utf8.EncodeRune(b8buffer, runes[0])
    		ret.Write(b8buffer[:size])
        }
        return strings.Trim(ret.String(), " \x00")
    }
    return ""
}

func readID3v2Cover(data []byte) Cover {
    if len(data) == 0 {
        return Cover{}
    }

    var cover Cover
    encoding := data[0]
    pos := bytes.IndexByte(data[1:], 0)
    if pos != -1 {
        cover.Mime = decodeText(encoding, data[1:pos + 1])
        data = data[pos + 2:]
    }

    if len(data) < 2 {
        return Cover{}
    }
    cover.Type = imageType[data[0]]

    pos = bytes.IndexByte(data[1:], 0)
    if pos != -1 {
        cover.Description = decodeText(encoding, data[1:pos + 1])
        data = data[pos + 2:]
    }

    cover.Data = make([]byte, len(data))
    copy(cover.Data, data)

    return cover
}

func aggregateTag(dst *Tag, src Tag) {
    if len(dst.Title) == 0 {
        dst.Title = src.Title
    }
    if len(dst.Artist) == 0 {
        dst.Artist = src.Artist
    }
    if len(dst.Album) == 0 {
        dst.Album = src.Album
    }
    if dst.Track == 0 {
        dst.Track = src.Track
    }
    if dst.Year == 0 {
        dst.Year = src.Year
    }
    if len(dst.Comment) == 0 {
        dst.Comment = src.Comment
    }
    if len(dst.Genre) == 0 {
        dst.Genre = src.Genre
    }
    if dst.Cover.Empty() {
        dst.Cover = src.Cover
    }
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

var imageType = map[byte]string {
    0: "Other", 1:  "32x32 file icon", 2: "Other file icon", 3: "Cover (front)", 4: "Cover (back)",
    5: "Leaflet page", 6: "Media", 7: "Lead artist/lead performer/soloist", 8: "Artist/performer",  9: "Conductor",
    10: "Band/Orchestra", 11: "Composer", 12: "Lyricist/text writer", 13: "Recording Location", 14: "During recording",
    15: "During performance", 16: "Movie/video screen capture", 17: "A bright coloured fish", 18: "Illustration", 19: "Band/artist logotype",
    20: "Publisher/Studio logotype",
}
