package server

import (
	"html/template"
	"strconv"
	"strings"
)

func reorderList(text string) ([]template.HTML, []string) {
	listItemsString := ""
	for _, lineString := range strings.Split(text, "\n") {
		if len(lineString) > 1 {
			if string(lineString[0]) != "-" {
				listItemsString += "- " + lineString + "\n"
			} else {
				listItemsString += lineString + "\n"
			}
		}
	}

	// get ordering of template.HTML for rendering
	renderedListString := MarkdownToHtml(listItemsString)
	listItems := []template.HTML{}
	endItems := []template.HTML{}
	for _, lineString := range strings.Split(renderedListString, "\n") {
		if len(lineString) > 1 {
			if strings.Contains(lineString, "<del>") || strings.Contains(lineString, "</ul>") {
				endItems = append(endItems, template.HTML(lineString))
			} else {
				listItems = append(listItems, template.HTML(lineString))
			}
		}
	}

	// get ordering of strings for deleting
	listItemsStringArray := []string{}
	endItemsStringArray := []string{}
	for _, lineString := range strings.Split(listItemsString, "\n") {
		if len(lineString) > 1 {
			if strings.Contains(lineString, "~~") {
				endItemsStringArray = append(endItemsStringArray, lineString)
			} else {
				listItemsStringArray = append(listItemsStringArray, lineString)
			}
		}
	}
	return append(listItems, endItems...), append(listItemsStringArray, endItemsStringArray...)
}

func renderList(currentRawText string) []template.HTML {
	listItems, _ := reorderList(currentRawText)
	for i := range listItems {
		newHTML := strings.Replace(string(listItems[i]), "</a>", "</a>"+`<span id="`+strconv.Itoa(i)+`" class="deletable">`, -1)
		newHTML = strings.Replace(newHTML, "<a href=", "</span><a href=", -1)
		newHTML = strings.Replace(newHTML, "<li>", "<li>"+`<span id="`+strconv.Itoa(i)+`" class="deletable">`, -1)
		newHTML = strings.Replace(newHTML, "</li>", "</span></li>", -1)
		newHTML = strings.Replace(newHTML, "<li>"+`<span id="`+strconv.Itoa(i)+`" class="deletable"><del>`, "<li><del>"+`<span id="`+strconv.Itoa(i)+`" class="deletable">`, -1)
		newHTML = strings.Replace(newHTML, "</del></span></li>", "</span></del></li>", -1)
		listItems[i] = template.HTML([]byte(newHTML))
	}
	return listItems
}
