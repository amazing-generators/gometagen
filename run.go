package gometagen

// // // // // // // // // //

func Run(config ConfigObj) (*ResultObj, error) {
	normalizedConfig, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}

	manifestObj, metaObj, manifestPath, hashModeValue, hashPointValue, err := collectNormalized(*normalizedConfig)
	if err != nil {
		return nil, err
	}

	dataArr, err := renderNormalized(*normalizedConfig, manifestObj, metaObj, manifestPath, hashModeValue, hashPointValue)
	if err != nil {
		return nil, err
	}

	dataArr = ensureTrailingNewline(dataArr)

	if !normalizedConfig.Stdout {
		if err = writeFileAtomically(normalizedConfig.OutputFile, dataArr, normalizedConfig.Force); err != nil {
			return nil, err
		}
	}

	return &ResultObj{
		OutputFile: normalizedConfig.OutputFile,
		Meta:       metaObj,
		Data:       dataArr,
	}, nil
}
