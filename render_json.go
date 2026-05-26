package gometagen

import _ "embed"

// // // // // // // // // //

//go:embed templates/render_json.tmpl
var jsonTemplateText string

// //

func renderJSON(dataObj templateDataObj) ([]byte, error) {
	return executePreparedTemplate(jsonTemplateObj, dataObj)
}
