package editor


import (
    "math"
    "strconv"
)

type Tag struct {
    Title string
    Artist string
    Album string
    Track int
    Year int
    Comment string
    Genre string
    Cover Cover
}

func (tag Tag) String() string {
    return "Title: " + tag.Title + "\n" +
           "Artist: " + tag.Artist + "\n" +
           "Album: " + tag.Album + "\n" +
           "Track: " + strconv.Itoa(tag.Track) + "\n" +
           "Year: " + strconv.Itoa(tag.Year) + "\n" +
           "Comment: " + tag.Comment + "\n" +
           "Genre: " + tag.Genre + "\n" +
           "Cover: " + tag.Cover.String()
}

func (tag Tag) Size() int {
    return len(tag.Title) +
           len(tag.Artist) +
           len(tag.Album) +
           int(math.Log10(float64(tag.Track))) +
           int(math.Log10(float64(tag.Year))) + 
           len(tag.Comment) +
           len(tag.Genre) +
           tag.Cover.Size()
}

func (tag Tag) Empty() bool {
    return len(tag.Title) == 0 &&
           len(tag.Artist) == 0 &&
           len(tag.Album) == 0 &&
           tag.Track == 0 && tag.Year == 0 &&
           len(tag.Comment) == 0 &&
           len(tag.Genre) == 0 &&
           tag.Cover.Empty()
}

func (tag *Tag) MergeWith(src Tag) {
    if len(tag.Title) == 0 {
        tag.Title = src.Title
    }
    if len(tag.Artist) == 0 {
        tag.Artist = src.Artist
    }
    if len(tag.Album) == 0 {
        tag.Album = src.Album
    }
    if tag.Track == 0 {
        tag.Track = src.Track
    }
    if tag.Year == 0 {
        tag.Year = src.Year
    }
    if len(tag.Comment) == 0 {
        tag.Comment = src.Comment
    }
    if len(tag.Genre) == 0 {
        tag.Genre = src.Genre
    }
    if tag.Cover.Empty() {
        tag.Cover = src.Cover
    }
}

type Cover struct {
    Mime string
    Type string
    Description string
    Data []byte
}

func (cover Cover) String() string {
    return "Mime: " + cover.Mime + "\n" +
           "Type: " + cover.Type + "\n" +
           "Description: " + cover.Description
}

func (cover Cover) Size() int {
    return len(cover.Mime) + len(cover.Type) + len(cover.Description) + len(cover.Data)
}

func (cover Cover) Empty() bool {
    return cover.Size() == 0
}
