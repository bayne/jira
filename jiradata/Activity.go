package jiradata

import (
	"encoding/xml"
)

// Define the root element 'feed' with the necessary namespaces
type Feed struct {
	XMLName  xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	ID       string   `xml:"id"`
	Link     []Link   `xml:"link"`
	Title    string   `xml:"title"`
	Timezone string   `xml:"http://streams.atlassian.com/syndication/general/1.0 timezone-offset"`
	Updated  string   `xml:"updated"`
	Entries  []Entry  `xml:"entry"`
}

// Link type to handle multiple link elements with different rel attributes
type Link struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr,omitempty"` // Type is optional and will only be set if the attribute exists
}

// Entry type that corresponds to the entry elements in the feed
type Entry struct {
	ID        string         `xml:"id"`
	Title     Title          `xml:"title"`
	Author    Author         `xml:"author"`
	Published string         `xml:"published"`
	Updated   string         `xml:"updated"`
	Category  Category       `xml:"category"`
	Links     []Link         `xml:"link"`
	Generator Generator      `xml:"generator"`
	Object    ActivityObject `xml:"http://activitystrea.ms/spec/1.0/ object"`
}

// Title type to handle the type attribute in title elements
type Title struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}

// Author type to handle nested elements within author
type Author struct {
	Name       string `xml:"name"`
	Email      string `xml:"email"`
	URI        string `xml:"uri"`
	Links      []Link `xml:"link"`
	Username   string `xml:"http://streams.atlassian.com/syndication/username/1.0 username"`
	ObjectType string `xml:"http://activitystrea.ms/spec/1.0/ object-type"`
}

// Category type for category elements
type Category struct {
	Term string `xml:"term,attr"`
}

// Generator type to capture information about the feed generator
type Generator struct {
	URI string `xml:"uri,attr"`
}

// ActivityObject type to capture details about an activity object
type ActivityObject struct {
	ID         string `xml:"id"`
	Title      Title  `xml:"title"`
	Summary    Title  `xml:"summary"` // Reusing Title type to capture optional type attribute in summary
	Link       Link   `xml:"link"`
	ObjectType string `xml:"http://activitystrea.ms/spec/1.0/ object-type"`
}
