package rules

import (
	"bytes"
	"errors"
)

// InspectDocument reuses the exporter's ownership and canonical metadata
// validators without writing files or interpreting rule prose.
func InspectDocument(relative string, content []byte) (id string, owned bool, err error) {
	id, owned, err = ownedExportRuleID(content)
	if err != nil || !owned {
		return
	}
	m, err := readDocument(relative, content)
	if err != nil {
		return "", true, err
	}
	return m.ID, true, nil
}

// InspectIndex checks only the owned block. External bytes are neither
// interpreted nor rewritten; links never establish document ownership.
func InspectIndex(format Format, content []byte, installed map[string][]byte) (active bool, err error) {
	start, end, active, err := indexRange(content)
	if err != nil || !active {
		return active, err
	}
	if format == FormatCursor {
		if err = validateCursor(content); err != nil {
			return active, err
		}
	}
	documents := map[string]documentMetadata{}
	for relative, data := range installed {
		m, e := readDocument(relative, data)
		if e != nil {
			return active, e
		}
		if _, duplicate := documents[m.ID]; duplicate {
			return active, errors.New("duplicate owned document")
		}
		documents[m.ID] = m
	}
	normalize := func(data []byte) []byte { return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")) }
	if !bytes.Equal(normalize(content[start:end]), normalize(renderIndex(format, documents))) {
		return active, errors.New("managed references differ from installed owned documents")
	}
	return active, nil
}
