package recognizer

import (
    "github.com/mzinin/tagger/editor"
    "strings"
)

func pickRelease(releases []interface{}, existingTag ... editor.Tag) map[string]interface{} {
    var tag editor.Tag
    if len(existingTag) > 0 {
        tag.Title = strings.ToUpper(existingTag[0].Title)
        tag.Artist = strings.ToUpper(existingTag[0].Artist)
        tag.Album = strings.ToUpper(existingTag[0].Album)
    }

    var best int = 0
    for i := 1; i < len(releases); i++ {
        if isMoreSuitableRelease(releases[i].(map[string]interface{}), releases[best].(map[string]interface{}), tag) {
            best = i
        }
    }

    return releases[best].(map[string]interface{})
}

func isMoreSuitableRelease(r1, r2 map[string]interface{}, tag editor.Tag) bool {
    data1 := getReleaseDate(r1)
    artist1 := strings.ToUpper(getReleaseArtist(r1))
    album1 := strings.ToUpper(getReleaseAlbum(r1))
    title1 := strings.ToUpper(getReleaseTitle(r1))

    data2 := getReleaseDate(r2)
    artist2 := strings.ToUpper(getReleaseArtist(r2))
    album2 := strings.ToUpper(getReleaseAlbum(r2))
    title2 := strings.ToUpper(getReleaseTitle(r2))

    // either there is no existing tag or both releases have the same RIGHT data - choose by date only
    if tag.Empty() ||
       len(tag.Artist) > 0 && len(tag.Album) > 0 && len(tag.Title) > 0 &&
       artist1 == tag.Artist && album1 == tag.Album && title1 == tag.Title &&
       artist2 == tag.Artist && album2 == tag.Album && title2 == tag.Title {
        return data1 != 0 && data1 < data2 || data2 == 0 && data1 != 0
    }

    // if 1st release has RIGHT data (and the 2nd one has not)
    if len(tag.Artist) > 0 && len(tag.Album) > 0 && len(tag.Title) > 0 &&
       artist1 == tag.Artist && album1 == tag.Album && title1 == tag.Title {
        return true
    }

    // if 2nd release has RIGHT data (and the 1st one has not)
    if len(tag.Artist) > 0 && len(tag.Album) > 0 && len(tag.Title) > 0 &&
       artist2 == tag.Artist && album2 == tag.Album && title2 == tag.Title {
        return false
    }


    // if both releases have the same RIGHT artist and title
    if len(tag.Artist) > 0 && len(tag.Title) > 0 &&
       artist1 == tag.Artist && title1 == tag.Title &&
       artist2 == tag.Artist && title2 == tag.Title {
        if len(album1) > 0 && len(album2) == 0 {
            return true
        }
        if len(album1) == 0 && len(album2) > 0 {
            return false
        }
        return data1 != 0 && data1 < data2 || data2 == 0 && data1 != 0
    }

    // if 1st release has RIGHT title and artist (and the 2nd one has not)
    if len(tag.Artist) > 0 && len(tag.Title) > 0 &&
       artist1 == tag.Artist && title1 == tag.Title {
        return true
    }

    // if 2nd release has RIGHT title and artist (and the 1st one has not)
    if len(tag.Artist) > 0 && len(tag.Title) > 0 &&
       artist2 == tag.Artist && title2 == tag.Title {
        return false
    }

    // prefer singles and albums over collections
    various1 := strings.Contains(artist1, "VARIOUS")
    various2 := strings.Contains(artist2, "VARIOUS")

    // if both releases have the same RIGHT album and title
    if len(tag.Album) > 0 && len(tag.Title) > 0 &&
       album1 == tag.Album && title1 == tag.Title &&
       album2 == tag.Album && title2 == tag.Title {
        if len(artist1) > 0 && len(artist2) == 0 || !various1 && various2 {
            return true
        }
        if len(artist1) == 0 && len(artist2) > 0 || various1 && !various2 {
            return false
        }
        return data1 != 0 && data1 < data2 || data2 == 0 && data1 != 0
    }

    // if 1st release has RIGHT album and title (and the 2nd one has not)
    if len(tag.Album) > 0 && len(tag.Title) > 0 &&
       album1 == tag.Album && title1 == tag.Title {
        return true
    }

    // if 2nd release has RIGHT album and title (and the 1st one has not)
    if len(tag.Album) > 0 && len(tag.Title) > 0 &&
       album2 == tag.Album && title2 == tag.Title {
        return false
    }

    // if both releases have the same RIGHT album and artist
    if len(tag.Album) > 0 && len(tag.Artist) > 0 &&
       album1 == tag.Album && artist1 == tag.Artist &&
       album2 == tag.Album && artist2 == tag.Artist {
        return data1 != 0 && data1 < data2 || data2 == 0 && data1 != 0
    }

    // if 1st release has RIGHT album and artist (and the 2nd one has not)
    if len(tag.Album) > 0 && len(tag.Artist) > 0 &&
       album1 == tag.Album && artist1 == tag.Artist {
        return true
    }

    // if 2nd release has RIGHT album and artist (and the 1st one has not)
    if len(tag.Album) > 0 && len(tag.Artist) > 0 &&
       album2 == tag.Album && artist2 == tag.Artist {
        return false
    }


    // if both releases have the same RIGHT title
    if len(tag.Title) > 0 && title1 == tag.Title && title2 == tag.Title {
        if len(artist1) > 0 && len(artist2) == 0 || !various1 && various2 || len(album1) > 0 && len(album2) == 0 {
            return true
        }
        if len(artist1) == 0 && len(artist2) > 0 || various1 && !various2 || len(album1) == 0 && len(album2) > 0 {
            return false
        }
        return data1 != 0 && data1 < data2 || data2 == 0 && data1 != 0
    }

    // if 1st release has RIGHT title (and the 2nd one has not)
    if len(tag.Title) > 0 && title1 == tag.Title {
        return true
    }

    // if 2nd release has RIGHT title (and the 1st one has not)
    if len(tag.Title) > 0 && title2 == tag.Title {
        return false
    }

    // if both releases have the same RIGHT artist
    if len(tag.Artist) > 0 && artist1 == tag.Artist && artist2 == tag.Artist {
        if len(title1) > 0 && len(title2) == 0 || len(album1) > 0 && len(album2) == 0 {
            return true
        }
        if len(title1) == 0 && len(title2) > 0 || len(album1) == 0 && len(album2) > 0 {
            return false
        }
        return data1 != 0 && data1 < data2 || data2 == 0 && data1 != 0
    }

    // if 1st release has RIGHT artist (and the 2nd one has not)
    if len(tag.Artist) > 0 && artist1 == tag.Artist {
        return true
    }

    // if 2nd release has RIGHT artist (and the 1st one has not)
    if len(tag.Artist) > 0 && artist2 == tag.Artist {
        return false
    }

    // if both releases have the same RIGHT album
    if len(tag.Album) > 0 && album1 == tag.Album && album2 == tag.Album {
        if len(artist1) > 0 && len(artist2) == 0 || !various1 && various2 || len(title1) > 0 && len(title2) == 0 {
            return true
        }
        if len(artist1) == 0 && len(artist2) > 0 || various1 && !various2 || len(title1) == 0 && len(title2) > 0 {
            return false
        }
        return data1 != 0 && data1 < data2 || data2 == 0 && data1 != 0
    }

    // if 1st release has RIGHT album (and the 2nd one has not)
    if len(tag.Album) > 0 && album1 == tag.Album {
        return true
    }

    // if 2nd release has RIGHT album (and the 1st one has not)
    if len(tag.Album) > 0 && album2 == tag.Album {
        return false
    }


    if !various1 && various2 {
        return true
    }
    if various1 && !various2 {
        return false
    }
    return data1 != 0 && data1 < data2 || data2 == 0 && data1 != 0
}

func getReleaseDate(release map[string]interface{}) int {
    if release["date"] != nil {
        date := release["date"].(map[string]interface{})
        return int(date["year"].(float64))
    }
    return 0
}

func getReleaseArtist(release map[string]interface{}) string {
    if release["artists"] != nil {
        artists := release["artists"].([]interface{})
        if len(artists) != 0 {
            artist := artists[0].(map[string]interface{})
            if artist["name"] != nil {
                return artist["name"].(string)
            }
        }
    }
    return ""
}

func getReleaseAlbum(release map[string]interface{}) string {
    if release["mediums"] != nil {
        mediums := release["mediums"].([]interface{})
        if len(mediums) != 0 {
            medium := mediums[0].(map[string]interface{})

            if medium["title"] != nil {
                return medium["title"].(string)
            } else if release["title"] != nil {
                return release["title"].(string)
            }
        }
    }
    return ""
}

func getReleaseTitle(release map[string]interface{}) string {
    if release["mediums"] != nil {
        mediums := release["mediums"].([]interface{})
        if len(mediums) != 0 {
            medium := mediums[0].(map[string]interface{})
            if  medium["tracks"] != nil {
                tracks := medium["tracks"].([]interface{})
                if len(tracks) != 0 {
                    track := tracks[0].(map[string]interface{})
                    if track["title"] != nil {
                        return track["title"].(string)
                    }
                }
            }
        }
    }
    return ""
}

func getReleaseTrack(release map[string]interface{}) int {
    if release["mediums"] != nil {
        mediums := release["mediums"].([]interface{})
        if len(mediums) != 0 {
            medium := mediums[0].(map[string]interface{})
            if  medium["tracks"] != nil {
                tracks := medium["tracks"].([]interface{})
                if len(tracks) != 0 {
                    track := tracks[0].(map[string]interface{})
                    if track["position"] != nil {
                        return int(track["position"].(float64))
                    }
                }
            }
        }
    }
    return 0
}