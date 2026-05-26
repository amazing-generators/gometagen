package gometagen

import _ "embed"

// // // // // // // // // //

//go:embed templates/render_yaml.tmpl
var yamlTemplateText string

// //

func renderYAML(dataObj templateDataObj) ([]byte, error) {
	return executePreparedTemplate(yamlTemplateObj, dataObj)
}
