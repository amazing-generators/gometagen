package gometagen

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
)

// // // // // // // // // //

//go:embed templates/render_go.tmpl
var goTemplateText string

// //

func renderGo(dataObj templateDataObj) ([]byte, error) {
	dataArr, err := executePreparedTemplate(goTemplateObj, dataObj)
	if err != nil {
		return nil, err
	}

	formattedArr, err := format.Source(dataArr)
	if err != nil {
		return nil, fmt.Errorf("format generated Go source: %w", err)
	}

	return bytes.TrimSpace(formattedArr), nil
}
