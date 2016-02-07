package editor

import (
    //"bytes"
    "errors"
    "io/ioutil"
    //"os"
    //"strconv"
    //"strings"

    //"fmt"

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
    return nil
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
    if len(data) == 0 {
        return errors.New("no flac comment block to parse")
    }

    vendorSize := utils.ReadInt32Le(data[4:8])
    return parseVorbisTags(data[8 + vendorSize:], tag)
}

func (editor *FlacTagEditor) parsePictureBlock(data []byte, cover *Cover) error {
    if len(data) == 0 {
        return errors.New("no flac picture block to parse")
    }

    return parseVorbisPictureTag(data[4:], cover)
}
