package editor

import (
    "errors"
    "io/ioutil"
    "os"

    "github.com/mzinin/tagger/utils"
)

const (
    flacHeaderMagic string = "fLaC"
    commentBlockType byte = 4
    pictureBlockType byte = 6
    lastMetaBlockFlag byte = 0x80
)

type FlacTagEditor struct {
    file []byte
}

func (editor *FlacTagEditor) ReadTag(path string) (Tag, error) {
    err := editor.readFile(path)
    if err != nil {
        return Tag{}, err
    }

    commentBlock, pictureBlock, _, _, _ := editor.splitFileData()

    var tag Tag
    editor.parseCommentBlock(commentBlock, &tag)
    editor.parsePictureBlock(pictureBlock, &tag.Cover)
    
    if tag.Empty() {
        return Tag{}, errors.New("no tag")
    }

    return tag, nil
}

func (editor *FlacTagEditor) WriteTag(src, dst string, tag Tag) error {
    err := editor.readFile(src)
    if err != nil {
        return err
    }

    commentBlock, pictureBlock, prefix, infix, suffix := editor.splitFileData()

    cover := tag.Cover
    tag.Cover = Cover{}
    newCommentBlock := editor.makeNewCommentBlock(tag, commentBlock)
    newPictureBlock := editor.makeNewPictureBlock(cover)

    // set last meta block flag on new picture block if nessesary
    if editor.findAndRemoveLastMetaBlockFlag(commentBlock) ||
       editor.findAndRemoveLastMetaBlockFlag(pictureBlock) ||
       editor.findAndRemoveLastMetaBlockFlag(prefix[len(flacHeaderMagic):]) {
        newPictureBlock[0] |= lastMetaBlockFlag
    }

    newData := make([]byte, len(prefix) + len(newCommentBlock) + len(newPictureBlock) + len(infix) + len(suffix))
    copy(newData, prefix)
    copy(newData[len(prefix):], newCommentBlock)
    copy(newData[len(prefix) + len(newCommentBlock):], newPictureBlock)
    copy(newData[len(prefix) + len(newCommentBlock) + len(newPictureBlock):], infix)
    copy(newData[len(prefix) + len(newCommentBlock) + len(newPictureBlock) + len(infix):], suffix)

    return ioutil.WriteFile(dst, newData, os.ModePerm)
}

func (editor *FlacTagEditor) readFile(path string) error {
    var err error
    editor.file, err = ioutil.ReadFile(path)
    return err
}

func (editor *FlacTagEditor) splitFileData() ([]byte, []byte, []byte, []byte, []byte) {
    if len(editor.file) < 8 || string(editor.file[0:4]) != flacHeaderMagic {
        return nil, nil, nil, nil, nil
    }

    commentPosition, commentSize, picturePosition, pictureSize := editor.getSizesAndPositions()

    if commentPosition == 0 && picturePosition == 0 {
        return nil, nil, editor.file, nil, nil
    }
    if commentPosition == 0 {
        return nil, editor.file[picturePosition : picturePosition + pictureSize], editor.file[:picturePosition], nil, editor.file[picturePosition + pictureSize:]
    }
    if picturePosition == 0 {
        return editor.file[commentPosition : commentPosition + commentSize], nil, editor.file[:commentPosition], nil, editor.file[commentPosition + commentSize:]
    }

    firstPosition := commentPosition
    firstSize := commentSize
    secondPosition := picturePosition
    secondSize := pictureSize
    if commentPosition > picturePosition {
        firstPosition = picturePosition
        firstSize = pictureSize
        secondPosition = commentPosition
        secondSize = commentSize
    }

    return editor.file[commentPosition : commentPosition + commentSize],
           editor.file[picturePosition : picturePosition + pictureSize],
           editor.file[:firstPosition],
           editor.file[firstPosition + firstSize : secondPosition],
           editor.file[secondPosition + secondSize:]
}

func (editor *FlacTagEditor) getSizesAndPositions() (int, int, int, int) {
    commentPosition := 0
    commentSize := 0
    picturePosition := 0
    pictureSize := 0

    data := editor.file[4:]
    lastBlock := false
    currentPosition := 4

    for len(data) > 3 && !lastBlock {
        blockType := data[0] & (^lastMetaBlockFlag)
        blockSize := utils.ReadInt24Be(data[1:4]) + 4

        switch blockType {
        case commentBlockType:
            commentPosition = currentPosition
            commentSize = blockSize
        case pictureBlockType:
            if blockSize > pictureSize {
                picturePosition = currentPosition
                pictureSize = blockSize
            }
        }

        lastBlock = data[0] & lastMetaBlockFlag == lastMetaBlockFlag
        currentPosition += blockSize
        data = data[blockSize:]
    }

    return commentPosition, commentSize, picturePosition, pictureSize
}

func (editor *FlacTagEditor) parseCommentBlock(data []byte, tag *Tag) error {
    if len(data) < 8 {
        return errors.New("no flac comment block to parse")
    }

    vendorSize := utils.ReadInt32Le(data[4:8])
    return parseVorbisTags(data[8 + vendorSize:], tag)
}

func (editor *FlacTagEditor) parsePictureBlock(data []byte, cover *Cover) error {
    if len(data) == 0 {
        return errors.New("no flac picture block to parse")
    }

    return parseVorbisTagPictureField(data[4:], cover)
}

func (editor *FlacTagEditor) makeNewCommentBlock(tag Tag, existingCommentBlock []byte) []byte {
    var vendorData []byte = nil
    var unsupportedTagData []byte = nil
    var unsupportedFields int = 0

    if len(existingCommentBlock) >= 8 {
        vendorSize := utils.ReadInt32Le(existingCommentBlock[4:8])
        vendorData = existingCommentBlock[4 : 8 + vendorSize]
        unsupportedTagData, unsupportedFields = getUnsupportedVorbisTags(existingCommentBlock[8 + vendorSize:])
    }

    newCommentData, totalFields := serializeVorbisTag(tag, unsupportedFields)

    newCommentBlock := make([]byte, 4 + len(vendorData) + 4 + len(newCommentData) + len(unsupportedTagData))
    newCommentBlock[0] = commentBlockType
    utils.WriteInt24Be(len(newCommentBlock) - 4, newCommentBlock[1:4])
    copy(newCommentBlock[4:], vendorData)
    utils.WriteInt32Le(totalFields, newCommentBlock[4 + len(vendorData) : 8 + len(vendorData)])
    copy(newCommentBlock[8 + len(vendorData):], newCommentData)
    copy(newCommentBlock[8 + len(vendorData) + len(newCommentData):], unsupportedTagData)

    return newCommentBlock
}

func (editor *FlacTagEditor) makeNewPictureBlock(cover Cover) []byte {
    if cover.Empty() {
        return nil
    }

    pictureData := serializeVorbisTagPictureField(cover)

    typeData := make([]byte, 4)
    typeData[0] = pictureBlockType
    utils.WriteInt24Be(len(pictureData), typeData[1:4])

    return append(typeData, pictureData ...)
}

func (editor *FlacTagEditor) findAndRemoveLastMetaBlockFlag(blocks []byte) bool {
    for len(blocks) > 3 {
        if blocks[0] & lastMetaBlockFlag == lastMetaBlockFlag {
            return true
        }
        blockSize := utils.ReadInt24Be(blocks[1:4]) + 4
        blocks = blocks[blockSize:]
    }
    return false
}
