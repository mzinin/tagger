package editor


import (
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

func (tag Tag) Empty() bool {
    return len(tag.Title) == 0 &&
           len(tag.Artist) == 0 &&
           len(tag.Album) == 0 &&
           tag.Track == 0 && tag.Year == 0 &&
           len(tag.Comment) == 0 &&
           len(tag.Genre) == 0 &&
           tag.Cover.Empty()
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

func (cover Cover) Empty() bool {
    return len(cover.Mime) == 0 &&
           len(cover.Type) == 0 &&
           len(cover.Description) == 0 &&
           len(cover.Data) == 0
}
