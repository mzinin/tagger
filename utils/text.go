package utils

import (
    "bytes"
    "strings"
    "unicode/utf16"
    "unicode/utf8"
)

func Utf16LeToUtf8(data []byte) string {
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
}

func Utf16BeToUtf8(data []byte) string {
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

func Utf8ToUtf16Le(text string) []byte {
    if len(text) == 0 {
        return nil
    }
    result := make([]byte, 2*len(text))
    counter := 0

    for _, char := range(text) {
        result[2*counter] = byte(char)
        result[2*counter+1] = byte(char >> 8)
        counter++
    }

    return result[:2*counter]
}

