package recognizer

import (
    "bytes"
    "compress/gzip"
    "encoding/json"
    "io/ioutil"
    "net/http"
    "path/filepath"
    "strconv"

    "github.com/mzinin/tagger/editor"
)

type UpdateStrategyType int

const (
    Always UpdateStrategyType = iota
    IfEmpty
    IfNoTitle
    IfNoTitleArtist
    IfNoTitleArtistAlbum
    IfNoCover
)

const (
    appKey string = "jouNIpYIoz"
)

func UpdateTag(tag editor.Tag, path string, strategy ... UpdateStrategyType) (editor.Tag, error) {
    s := IfNoTitle
    if len(strategy) > 0 {
        s = strategy[0]
    }

    if !needUpdate(tag, s) {
        return tag, nil
    }

    newTag, err := Recognize(path)
    if err != nil {
        return tag, nil
    }

    if s == IfNoCover {
        tag.Cover = newTag.Cover
        return tag, nil
    }
        
    newTag.MergeWith(tag)
    return newTag, nil
}

func Recognize(path string) (editor.Tag, error) {
    fingerPrint, duration, err := getFingerPrint(path)
    if err != nil {
        return editor.Tag{}, err
    }

    return askMusicBrainz(fingerPrint, duration)
}

func askMusicBrainz(fingerPrint string, duration int) (editor.Tag, error) {
    reply, err := lookupByFingerPrint(fingerPrint, duration)
    if err != nil {
        return editor.Tag{}, err
    }

    tag, releaseId := parseAcousticIdReply(reply)
    if len(releaseId) > 0 {
        tag.Cover = askCoverArtArchive(releaseId)
    }

    return tag, nil
}

func lookupByFingerPrint(fingetPrint string, duration int) (string, error) {
    data := "client=" + appKey + "&meta=releases+tracks+compress&duration=" + strconv.Itoa(duration) + "&fingerprint=" + fingetPrint
    var zippedData bytes.Buffer
    zipper := gzip.NewWriter(&zippedData)
    zipper.Write([]byte(data))
    zipper.Close()

    request, err := http.NewRequest("POST", "http://api.acoustid.org/v2/lookup", bytes.NewReader(zippedData.Bytes()))
    if err != nil {
        return "", err
    }
    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    request.Header.Add("Content-Encoding", "gzip")

    var response *http.Response
    response, err = (&http.Client{}).Do(request)
    if err != nil {
        return "", err
    }

    var reply []byte
    reply, err = ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return "", err
	}
    return string(reply), nil
}

func parseAcousticIdReply(reply string) (editor.Tag, string) {
    var fields map[string]interface{} 
    err := json.Unmarshal([]byte(reply), &fields)

    if err != nil || fields["status"] != "ok" {
        return editor.Tag{}, ""
    }

    if fields["results"] == nil {
        return editor.Tag{}, ""
    }
    results := fields["results"].([]interface{})
    if len(results) == 0 {
        return editor.Tag{}, ""
    }
    result := results[0].(map[string]interface{})

    if result["releases"] == nil {
        return editor.Tag{}, ""
    }
    releases := result["releases"].([]interface{})
    if len(releases) == 0 {
        return editor.Tag{}, ""
    }
    release := releases[0].(map[string]interface{})

    var tag editor.Tag

    if release["date"] != nil {
        date := release["date"].(map[string]interface{})
        tag.Year = int(date["year"].(float64))
    }
    
    if release["artists"] != nil {
        artists := release["artists"].([]interface{})
        if len(artists) != 0 {
            artist := artists[0].(map[string]interface{})
            if artist["name"] != nil {
                tag.Artist = artist["name"].(string)
            }
        }
    }

    if release["mediums"] != nil {
        mediums := release["mediums"].([]interface{})
        if len(mediums) != 0 {
            medium := mediums[0].(map[string]interface{})

            if medium["title"] != nil {
                tag.Album = medium["title"].(string)
            } else if release["title"] != nil {
                tag.Album = release["title"].(string)
            }

            if  medium["tracks"] != nil {
                tracks := medium["tracks"].([]interface{})
                if len(tracks) != 0 {
                    track := tracks[0].(map[string]interface{})
                    if track["title"] != nil {
                        tag.Title = track["title"].(string)
                    }
                    if track["position"] != nil {
                        tag.Track = int(track["position"].(float64))
                    }
                }
            }
        }
    }

    return tag, release["id"].(string)
}

func askCoverArtArchive(releaseId string) editor.Cover {
    response, err := http.Get("http://coverartarchive.org/release/" + releaseId)
    if err != nil || response.StatusCode != 200 {
        return editor.Cover{}
    }

    var reply []byte
    reply, err = ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return editor.Cover{}
	}

    imageUrl := parseCoverArtArchiveReply(string(reply))
    return getCover(imageUrl)
}

func parseCoverArtArchiveReply(reply string) string {
    var fields map[string]interface{} 
    err := json.Unmarshal([]byte(reply), &fields)
    if err != nil {
        return ""
    }

    if fields["images"] == nil {
        return ""
    }
    images := fields["images"].([]interface{})
    if len(images) == 0 {
        return ""
    }
    image := images[0].(map[string]interface{})

    var imageUrl string

    if image["thumbnails"] != nil {
        thumbnails := image["thumbnails"].(map[string]interface{})
        if thumbnails["large"] != nil {
            imageUrl = thumbnails["large"].(string)
        } else if thumbnails["small"] != nil {
            imageUrl = thumbnails["small"].(string)
        }
    }

    if len(imageUrl) == 0 && image["image"] != nil {
        imageUrl = image["image"].(string)
    }

    return imageUrl
}

func getCover(url string) editor.Cover {
    if len(url) == 0 {
        return editor.Cover{}
    }

    response, err := http.Get(url)
    if err != nil || response.StatusCode != 200 {
        return editor.Cover{}
    }

    var cover editor.Cover
    cover.Data, err = ioutil.ReadAll(response.Body)
    response.Body.Close()
	if err != nil {
		return editor.Cover{}
	}

    switch filepath.Ext(url) {
    case ".jpg", ".jpeg":
        cover.Mime = "image/jpeg"
    case ".png":
        cover.Mime = "image/png"
    case ".tif", ".tiff":
        cover.Mime = "image/tiff"
    }

    // TODO parse out from json
    cover.Type = "Cover (front)"

    return cover
}

func needUpdate(tag editor.Tag, strategy UpdateStrategyType) bool {
    switch strategy {
    case Always:
        return true
    case IfNoTitle:
        return len(tag.Title) == 0
    case IfNoTitleArtist:
        return len(tag.Title) == 0 && len(tag.Artist) == 0
    case IfNoTitleArtistAlbum:
        return len(tag.Title) == 0 && len(tag.Artist) == 0 && len(tag.Album) == 0
    case IfNoCover:
        return tag.Cover.Empty()
    }
    return false
}
