package utils


func ReadInt32Be(data []byte) int {
    return ((int(data[0]) * 0x100 + int(data[1])) * 0x100 + int(data[2])) * 0x100 + int(data[3])
}

func ReadInt32Le(data []byte) int {
    return ((int(data[3]) * 0x100 + int(data[2])) * 0x100 + int(data[1])) * 0x100 + int(data[0])
}

func WriteInt32Be(size int, dst []byte) {
    if len(dst) < 4 {
        return
    }
    for i := 3; i >= 0; i-- {
        dst[i] = byte(size % 0x100)
        size = size / 0x100
    }
}

