package editor

import (
    "bytes"
    "errors"
    "io/ioutil"
    "os"

    "github.com/mzinin/tagger/utils"
)

const (
    oggPageMagic string = "OggS"
    oggPageHeaderSize int = 27
    oggHeaderMagic = "vorbis"
    headerTypeContinue byte = 1
    maxFrameDataSize int = 65025 // 65307 - 282
)

type OggTagEditor struct {
    file []byte
}

func (editor *OggTagEditor) ReadTag(path string) (Tag, error) {
    err := editor.readFile(path)
    if err != nil {
        return Tag{}, err
    }

    _, commentPages, _ := editor.splitFileData(editor.file)
    if len(commentPages) == 0 {
        return Tag{}, errors.New("no tag")
    }

    _, tagData, _ := editor.splitCommentPages(commentPages)

    var tag Tag
    return tag, parseVorbisTags(tagData, &tag)
}

func (editor *OggTagEditor) WriteTag(src, dst string, tag Tag) error {
    err := editor.readFile(src)
    if err != nil {
        return err
    }

    idPage, commentPages, restData := editor.splitFileData(editor.file)
    newCommentPages, newSetupPages, numberOfPages := editor.makeNewPages(commentPages, tag)

    newPrefix := make([]byte, len(idPage) + len(newCommentPages) + len(newSetupPages))
    copy(newPrefix, idPage)
    copy(newPrefix[len(idPage):], newCommentPages)
    copy(newPrefix[len(idPage) + len(newCommentPages):], newSetupPages)

    // fix page numbers and CRCs
    sequence := numberOfPages + 1
    data := restData
    for len(data) > oggPageHeaderSize {
        pageSize := editor.getPageSize(data)

        utils.WriteInt32Le(sequence, data[18:22]) // number
        utils.WriteUint32Le(0, data[22:26]) // zero CRC
        utils.WriteUint32Le(editor.crc(data[:pageSize]), data[22:26]) // CRC

        sequence++
        data = data[pageSize:]
    }

    return ioutil.WriteFile(dst, append(newPrefix, restData ...), os.ModePerm)
}

func (editor *OggTagEditor) readFile(path string) error {
    var err error
    editor.file, err = ioutil.ReadFile(path)
    return err
}

func (editor *OggTagEditor) splitFileData(data []byte) ([]byte, []byte, []byte) {
    firstPageSize := editor.getPageSize(data)
    secondPageSize := editor.getPageSize(data[firstPageSize:])

    for secondPageSize >= oggPageHeaderSize && data[firstPageSize + 5] & headerTypeContinue != 0 {
        firstPageSize += secondPageSize
        secondPageSize = editor.getPageSize(data[firstPageSize:])
    }

    thirdPageSize := editor.getPageSize(data[firstPageSize + secondPageSize:])
    for thirdPageSize >= oggPageHeaderSize && data[firstPageSize + secondPageSize + 5] & headerTypeContinue != 0 {
        secondPageSize += thirdPageSize
        thirdPageSize = editor.getPageSize(data[firstPageSize + secondPageSize:])
    }

    return data[:firstPageSize], data[firstPageSize : firstPageSize + secondPageSize], data[firstPageSize + secondPageSize:]
}

func (editor *OggTagEditor) getPageSize(page []byte) int {
    if len(page) < oggPageHeaderSize {
        return 0
    }
    if string(page[0:4]) != oggPageMagic {
        return 0
    }

    pageSegmentsNumber := int(page[oggPageHeaderSize - 1])
    headerSize := oggPageHeaderSize + pageSegmentsNumber
    if (len(page) < headerSize) {
        return 0
    }

    pageSegmentsSize := 0
    for _, byteValue := range page[oggPageHeaderSize : headerSize] {
        pageSegmentsSize += int(byteValue)
    }

    return headerSize + pageSegmentsSize
}

func (editor *OggTagEditor) splitCommentPages(pages []byte) ([]byte, []byte, []byte) {
    var commentHeader []byte = nil
    var tagData []byte = nil
    var setupHeader []byte = nil

    for len(pages) > oggPageHeaderSize {
        pageHeaderSize := oggPageHeaderSize + int(pages[oggPageHeaderSize - 1])

        // it's the 1st page, the one with comment header
        if pages[5] & headerTypeContinue == 0 {
            if len(pages) < pageHeaderSize + 5 + len(oggHeaderMagic) {
                return nil, nil, nil
            }
            commentHeaderSize := utils.ReadInt32Le(pages[pageHeaderSize + 1 + len(oggHeaderMagic) : pageHeaderSize + 5 + len(oggHeaderMagic)])
            if len(pages) < pageHeaderSize + 5 + len(oggHeaderMagic) + commentHeaderSize {
                return nil, nil, nil
            }
            commentHeader = pages[pageHeaderSize : pageHeaderSize + 5 + len(oggHeaderMagic) + commentHeaderSize]
            pageHeaderSize += 5 + len(oggHeaderMagic) + commentHeaderSize
        }

        pageSize := editor.getPageSize(pages)
        tagData = append(tagData, pages[pageHeaderSize : pageSize] ...)
        pages = pages[pageSize:]
    }

    pos := bytes.Index(tagData, []byte(oggHeaderMagic))
    if pos != -1 {
        setupHeader = tagData[pos - 1:]
        tagData = tagData[:pos - 1]
    }

    return commentHeader, tagData, setupHeader
}

func (editor *OggTagEditor) makeNewPages(existingCommentPages []byte, tag Tag) ([]byte, []byte, int) {
    // bitstream number
    var bitstream int = 31013
    if len(existingCommentPages) > oggPageHeaderSize {
        bitstream = utils.ReadInt32Le(existingCommentPages[14 : 18])
    }

    commentHeader, existingTagData, setupHeader := editor.splitCommentPages(existingCommentPages)
    if len(commentHeader) == 0 {
        commentHeader = make([]byte, 1 + len(oggHeaderMagic) + 4)
        commentHeader[0] = 3
        copy(commentHeader[1 : 1 + len(oggHeaderMagic)], oggHeaderMagic)
        utils.WriteInt32Le(0, commentHeader[len(oggHeaderMagic) : len(oggHeaderMagic) + 4])
    }

    unsupportedTagData, unsupportedFields := getUnsupportedVorbisTags(existingTagData)
    newTagData, totalFields := serializeVorbisTag(tag, unsupportedFields)

    tagData := make([]byte, len(commentHeader) + 4 + len(newTagData) + len(unsupportedTagData) + 1)
    copy(tagData, commentHeader)
    utils.WriteInt32Le(totalFields, tagData[len(commentHeader) : len(commentHeader) + 4])
    copy(tagData[len(commentHeader) + 4:], newTagData)
    copy(tagData[len(commentHeader) + 4 + len(newTagData):], unsupportedTagData)
    tagData[len(tagData) - 1] = 1

    newCommentHeader, commentPages := editor.packTagDataIntoFrames(bitstream, 1, tagData)
    newSetupHeader, setupPages := editor.packTagDataIntoFrames(bitstream, commentPages + 1, setupHeader)

    return newCommentHeader, newSetupHeader, commentPages + setupPages
}

func (editor *OggTagEditor) packTagDataIntoFrames(bitstream, sequence int, tagData []byte) ([]byte, int) {
    totalTagSize := len(tagData)
    if totalTagSize == 0 {
        return nil, 0
    }

    tagSize := 0
    size := 0
    pages := 0

    result := make([]byte, totalTagSize + 282 * (totalTagSize / maxFrameDataSize + 1))

    for tagSize < totalTagSize {
        headerSize, dataSize := editor.fillHeader(result[size:], bitstream, sequence + pages, totalTagSize - tagSize)
        pages++
        if tagSize != 0 {
            result[size + len(oggPageMagic) + 1] = headerTypeContinue
        }
        size += headerSize

        copy(result[size : size + dataSize], tagData[:dataSize])
        size += dataSize
        tagSize += dataSize
        tagData = tagData[dataSize:]

        checksum := editor.crc(result[size - headerSize - dataSize : size])
        utils.WriteUint32Le(checksum, result[size - headerSize - dataSize + 22 : size - headerSize - dataSize + 26])
    }

    return result[:size], pages
}

func (editor *OggTagEditor) fillHeader(dst []byte, bitstream, sequence, totalSize int) (int, int) {
    copy(dst[0:4], oggPageMagic) // magic
    dst[4] = 0 // version
    dst[5] = 0 // header type
    dst[6] = 0 // granule position
    dst[7] = 0
    dst[8] = 0
    dst[9] = 0
    dst[10] = 0
    dst[11] = 0
    dst[12] = 0
    dst[13] = 0
    utils.WriteInt32Le(bitstream, dst[14:18]) // bitstream serial number
    utils.WriteInt32Le(sequence, dst[18:22]) // page sequence number
    dst[22] = 0 // CRC checksum
    dst[23] = 0
    dst[24] = 0
    dst[25] = 0

    dataSize := totalSize
    if dataSize > maxFrameDataSize {
        dataSize = maxFrameDataSize
    }

    dst[26] = byte(dataSize / 255)
    if dataSize % 255 != 0 {
        dst[26]++
    }

    headerSize := 27 + int(dst[26])
    for i := 27; i < headerSize; i++ {
        dst[i] = 0xFF
    }
    if dataSize % 255 != 0 {
        dst[headerSize - 1] = byte(dataSize % 255)
    }

    return headerSize, dataSize
}

func (editor *OggTagEditor) crc(data []byte) uint32 {
    var crc uint32 = 0
    for i := range data {
        crc = oggCrcTable[byte(crc>>24) ^ data[i]] ^ (crc << 8)
    }
    return crc
}

var oggCrcTable = []uint32 {
	0x00000000, 0x04c11db7, 0x09823b6e, 0x0d4326d9,	0x130476dc, 0x17c56b6b, 0x1a864db2, 0x1e475005,
	0x2608edb8, 0x22c9f00f, 0x2f8ad6d6, 0x2b4bcb61,	0x350c9b64, 0x31cd86d3, 0x3c8ea00a, 0x384fbdbd,
	0x4c11db70, 0x48d0c6c7, 0x4593e01e, 0x4152fda9,	0x5f15adac, 0x5bd4b01b, 0x569796c2, 0x52568b75,
	0x6a1936c8, 0x6ed82b7f, 0x639b0da6, 0x675a1011,	0x791d4014, 0x7ddc5da3, 0x709f7b7a, 0x745e66cd,
	0x9823b6e0, 0x9ce2ab57, 0x91a18d8e, 0x95609039,	0x8b27c03c, 0x8fe6dd8b, 0x82a5fb52, 0x8664e6e5,
	0xbe2b5b58, 0xbaea46ef, 0xb7a96036, 0xb3687d81,	0xad2f2d84, 0xa9ee3033, 0xa4ad16ea, 0xa06c0b5d,
	0xd4326d90, 0xd0f37027, 0xddb056fe, 0xd9714b49,	0xc7361b4c, 0xc3f706fb, 0xceb42022, 0xca753d95,
	0xf23a8028, 0xf6fb9d9f, 0xfbb8bb46, 0xff79a6f1,	0xe13ef6f4, 0xe5ffeb43, 0xe8bccd9a, 0xec7dd02d,
	0x34867077, 0x30476dc0, 0x3d044b19, 0x39c556ae,	0x278206ab, 0x23431b1c, 0x2e003dc5, 0x2ac12072,
	0x128e9dcf, 0x164f8078, 0x1b0ca6a1, 0x1fcdbb16,	0x018aeb13, 0x054bf6a4, 0x0808d07d, 0x0cc9cdca,
	0x7897ab07, 0x7c56b6b0, 0x71159069, 0x75d48dde,	0x6b93dddb, 0x6f52c06c, 0x6211e6b5, 0x66d0fb02,
	0x5e9f46bf, 0x5a5e5b08, 0x571d7dd1, 0x53dc6066,	0x4d9b3063, 0x495a2dd4, 0x44190b0d, 0x40d816ba,
	0xaca5c697, 0xa864db20, 0xa527fdf9, 0xa1e6e04e,	0xbfa1b04b, 0xbb60adfc, 0xb6238b25, 0xb2e29692,
	0x8aad2b2f, 0x8e6c3698, 0x832f1041, 0x87ee0df6,	0x99a95df3, 0x9d684044, 0x902b669d, 0x94ea7b2a,
	0xe0b41de7, 0xe4750050, 0xe9362689, 0xedf73b3e,	0xf3b06b3b, 0xf771768c, 0xfa325055, 0xfef34de2,
	0xc6bcf05f, 0xc27dede8, 0xcf3ecb31, 0xcbffd686,	0xd5b88683, 0xd1799b34, 0xdc3abded, 0xd8fba05a,
	0x690ce0ee, 0x6dcdfd59, 0x608edb80, 0x644fc637,	0x7a089632, 0x7ec98b85, 0x738aad5c, 0x774bb0eb,
	0x4f040d56, 0x4bc510e1, 0x46863638, 0x42472b8f,	0x5c007b8a, 0x58c1663d, 0x558240e4, 0x51435d53,
	0x251d3b9e, 0x21dc2629, 0x2c9f00f0, 0x285e1d47,	0x36194d42, 0x32d850f5, 0x3f9b762c, 0x3b5a6b9b,
	0x0315d626, 0x07d4cb91, 0x0a97ed48, 0x0e56f0ff,	0x1011a0fa, 0x14d0bd4d, 0x19939b94, 0x1d528623,
	0xf12f560e, 0xf5ee4bb9, 0xf8ad6d60, 0xfc6c70d7,	0xe22b20d2, 0xe6ea3d65, 0xeba91bbc, 0xef68060b,
	0xd727bbb6, 0xd3e6a601, 0xdea580d8, 0xda649d6f,	0xc423cd6a, 0xc0e2d0dd, 0xcda1f604, 0xc960ebb3,
	0xbd3e8d7e, 0xb9ff90c9, 0xb4bcb610, 0xb07daba7,	0xae3afba2, 0xaafbe615, 0xa7b8c0cc, 0xa379dd7b,
	0x9b3660c6, 0x9ff77d71, 0x92b45ba8, 0x9675461f,	0x8832161a, 0x8cf30bad, 0x81b02d74, 0x857130c3,
	0x5d8a9099, 0x594b8d2e, 0x5408abf7, 0x50c9b640,	0x4e8ee645, 0x4a4ffbf2, 0x470cdd2b, 0x43cdc09c,
	0x7b827d21, 0x7f436096, 0x7200464f, 0x76c15bf8,	0x68860bfd, 0x6c47164a, 0x61043093, 0x65c52d24,
	0x119b4be9, 0x155a565e, 0x18197087, 0x1cd86d30,	0x029f3d35, 0x065e2082, 0x0b1d065b, 0x0fdc1bec,
	0x3793a651, 0x3352bbe6, 0x3e119d3f, 0x3ad08088,	0x2497d08d, 0x2056cd3a, 0x2d15ebe3, 0x29d4f654,
	0xc5a92679, 0xc1683bce, 0xcc2b1d17, 0xc8ea00a0,	0xd6ad50a5, 0xd26c4d12, 0xdf2f6bcb, 0xdbee767c,
	0xe3a1cbc1, 0xe760d676, 0xea23f0af, 0xeee2ed18,	0xf0a5bd1d, 0xf464a0aa, 0xf9278673, 0xfde69bc4,
	0x89b8fd09, 0x8d79e0be, 0x803ac667, 0x84fbdbd0,	0x9abc8bd5, 0x9e7d9662, 0x933eb0bb, 0x97ffad0c,
	0xafb010b1, 0xab710d06, 0xa6322bdf, 0xa2f33668,	0xbcb4666d, 0xb8757bda, 0xb5365d03, 0xb1f740b4,
}