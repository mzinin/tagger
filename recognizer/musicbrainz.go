package recognizer

import (
    "bytes"
    "compress/gzip"
    "encoding/json"
    "io/ioutil"
    "net/http"
    "path/filepath"
    "strconv"
    "sync"
    "time"

    "github.com/mzinin/tagger/editor"
    "github.com/mzinin/tagger/utils"
)

const (
    appKey string = "jouNIpYIoz"
    musizBrainzDelay time.Duration = 350 * time.Millisecond
)

var (
    musicBrainzMutex sync.Mutex
    lastMusicBrainzRequestTime time.Time = time.Unix(0, 0)
)

func Recognize(path string) (editor.Tag, error) {
    fingerPrint, duration, err := getFingerPrint(path)
    if err != nil {
        utils.Log(utils.ERROR, "Failed to get finger print for file '%v': %v", path, err)
        return editor.Tag{}, err
    }

    return askMusicBrainz(fingerPrint, duration)
}

func askMusicBrainz(fingerPrint string, duration int) (editor.Tag, error) {
    waitIfNeeded()

    reply, err := lookupByFingerPrint(fingerPrint, duration)
    if err != nil {
        utils.Log(utils.ERROR, "Failed to lookup by finger print: %v", err)
        return editor.Tag{}, err
    }

    tag, releaseId := parseAcousticIdReply(reply)
    if len(releaseId) > 0 {
        tag.Cover = askCoverArtArchive(releaseId)
    }

    return tag, nil
}

func waitIfNeeded() {
    musicBrainzMutex.Lock()
    defer musicBrainzMutex.Unlock()

    sinceLastRequest := time.Now().Sub(lastMusicBrainzRequestTime)
    if sinceLastRequest < musizBrainzDelay {
        time.Sleep(musizBrainzDelay - sinceLastRequest)
    }

    lastMusicBrainzRequestTime = time.Now()
}

func lookupByFingerPrint(fingetPrint string, duration int) (string, error) {
    data := "client=" + appKey + "&meta=releases+tracks+compress&duration=" + strconv.Itoa(duration) + "&fingerprint=" + fingetPrint
    var zippedData bytes.Buffer
    zipper := gzip.NewWriter(&zippedData)
    zipper.Write([]byte(data))
    zipper.Close()

    request, err := http.NewRequest("POST", "http://api.acoustid.org/v2/lookup", bytes.NewReader(zippedData.Bytes()))
    if err != nil {
        utils.Log(utils.ERROR, "Failed to make new http request: %v", err)
        return "", err
    }
    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    request.Header.Add("Content-Encoding", "gzip")

    response, err := (&http.Client{}).Do(request)
    if err != nil {
        utils.Log(utils.ERROR, "Failed to send http request: %v", err)
        return "", err
    }

    reply, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
        utils.Log(utils.ERROR, "Failed to read http response: %v", err)
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
    release := getFirstRelease(releases)

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

func getFirstRelease(releases []interface{}) map[string]interface{} {
    var year int = 9999
    var index int = 0

    for i, r := range releases {
        release := r.(map[string]interface{})
        if release["date"] == nil {
            continue
        }
        date := release["date"].(map[string]interface{})
        y := int(date["year"].(float64))

        if y < year {
            year = y
            index = i
        }
    }

    return releases[index].(map[string]interface{})
}

func askCoverArtArchive(releaseId string) editor.Cover {
    response, err := http.Get("http://coverartarchive.org/release/" + releaseId)
    if err != nil {
        utils.Log(utils.ERROR, "Failed to send http request '%v' and get response: %v", "http://coverartarchive.org/release/" + releaseId, err)
        return editor.Cover{}
    }
    if response.StatusCode != 200 {
        return editor.Cover{}
    }

    reply, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
        utils.Log(utils.ERROR, "Failed to read http response: %v", err)
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
