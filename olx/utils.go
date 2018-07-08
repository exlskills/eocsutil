package olx

import (
	"encoding/xml"
	"fmt"
)

func urlNameToXMLFileName(urlName string) string {
	return fmt.Sprintf("%s.xml", urlName)
}

func urlNameToHTMLFileName(urlName string) string {
	return fmt.Sprintf("%s.html", urlName)
}

func xmlAttrsToMap(attrs []xml.Attr) map[string]string {
	ret := make(map[string]string, len(attrs))
	for _, a := range attrs {
		ret[a.Name.Local] = a.Value
	}
	return ret
}
