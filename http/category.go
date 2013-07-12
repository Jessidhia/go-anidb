package httpapi

import (
	"encoding/xml"
	"strings"
)

// Separate from regular Category because of different XML structure.
type CLCategory struct {
	ID       int  `xml:"id,attr"`       // Category ID
	ParentID int  `xml:"parentid,attr"` // ID of the parent category
	R18      bool `xml:"ishentai,attr"` // Whether the category is associated with porn or not

	Name        string `xml:"name"`        // Category name
	Description string `xml:"description"` // Category description
}

type CategoryList struct {
	Error      string       `xml:",chardata"`
	Categories []CLCategory `xml:"category"`
}

func GetCategoryList() (cl CategoryList, err error) {
	if res, err := doRequest("categorylist", reqMap{}); err != nil {
		return cl, err
	} else {
		dec := xml.NewDecoder(res.Body)
		err = dec.Decode(&cl)
		res.Body.Close()

		cl.Error = strings.TrimSpace(cl.Error)

		return cl, err
	}
}
