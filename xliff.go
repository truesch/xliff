// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/

package xliff

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
)

type DocumentExport struct {
	*Document
	XMLName        xml.Name `xml:"xliff"`
	Xmlns          string   `xml:"xmlns,attr"`
	Xsi            string   `xml:"xmlns:xsi,attr"`
	SchemaLocation string   `xml:"xsi:schemaLocation,attr"`
}

type Document struct {
	Version string `xml:"version,attr"`
	Files   []File `xml:"file"`
}

type File struct {
	Original       string `xml:"original,attr"`
	SourceLanguage string `xml:"source-language,attr"`
	Datatype       string `xml:"datatype,attr"`
	TargetLanguage string `xml:"target-language,attr"`
	Header         Header `xml:"header"`
	Body           Body   `xml:"body"`
}

type Header struct {
	Tool Tool `xml:"tool"`
}

type Tool struct {
	ToolID      string `xml:"tool-id,attr"`
	ToolName    string `xml:"tool-name,attr"`
	ToolVersion string `xml:"tool-version,attr"`
	BuildNum    string `xml:"build-num,attr"`
}

type Body struct {
	TransUnits []TransUnit `xml:"trans-unit"`
}

// Returns a new, empty xliff file.
// datatype will always be "plaintext" and version will always be "1.2"
func NewDocument(sl string, tl string) *Document {
	file := File{
		Datatype:       "plaintext",
		SourceLanguage: sl,
		TargetLanguage: tl,
		Header:         Header{},
		Body:           Body{},
	}
	return &Document{
		Version: "1.2",
		Files:   []File{file},
	}
}

// Reads XLIFF Document from disk
func FromFile(path string) (*Document, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return &Document{}, err
	}

	var document Document
	if err := xml.Unmarshal(data, &document); err != nil {
		return &Document{}, err
	}

	return &document, nil
}

// Writes XLIFF Document to disk
func (d *Document) ToFile(path string) error {
	xliff := &DocumentExport{
		Document:       d,
		Xmlns:          "urn:oasis:names:tc:xliff:document:1.2",
		Xsi:            "http://www.w3.org/2001/XMLSchema-instance",
		SchemaLocation: "urn:oasis:names:tc:xliff:document:1.2 http://docs.oasis-open.org/xliff/v1.2/os/xliff-core-1.2-strict.xsd",
	}

	data, err := xml.Marshal(xliff)
	if err != nil {
		return err
	}
	data = []byte(xml.Header + string(data))

	err = ioutil.WriteFile(path, data, 0664)
	if err != nil {
		return err
	}

	return nil
}

// Returns true if the document passes some basic consistency checks.
func (d *Document) Validate() []ValidationError {
	var errors []ValidationError

	// Make sure the document is a version we understand
	if d.Version != "1.2" {
		errors = append(errors, ValidationError{
			Code:    UnsupportedVersion,
			Message: fmt.Sprintf("Version %s is not supported", d.Version),
		})
	}

	// Make sure all files have the attributes we need
	for idx, file := range d.Files {
		if file.Original == "" {
			errors = append(errors, ValidationError{
				Code:    MissingOriginalAttribute,
				Message: fmt.Sprintf("File #%d is missing 'original' attribute", idx),
			})
		}
		if file.SourceLanguage == "" {
			errors = append(errors, ValidationError{
				Code:    MissingSourceLanguage,
				Message: fmt.Sprintf("File '%s' is missing 'source-language' attribute", file.Original),
			})
		}
		if file.TargetLanguage == "" {
			errors = append(errors, ValidationError{
				Code:    MissingTargetLanguage,
				Message: fmt.Sprintf("File '%s' is missing 'target-language' attribute", file.Original),
			})
		}
		if file.Datatype != "plaintext" {
			errors = append(errors, ValidationError{
				Code: UnsupportedDatatype,
				Message: fmt.Sprintf("File '%s' has unsupported 'datatype' attribute with value '%s'",
					file.Original, file.Datatype),
			})
		}
	}

	// Make sure all files are consistent with source and target language
	sourceLanguage, targetLanguage := d.Files[0].SourceLanguage, d.Files[0].TargetLanguage
	for _, file := range d.Files {
		if file.SourceLanguage != sourceLanguage {
			errors = append(errors, ValidationError{
				Code: InconsistentSourceLanguage,
				Message: fmt.Sprintf("File '%s' has inconsistent 'source-language' attribute '%s'",
					file.Original, file.SourceLanguage),
			})
		}
		if file.TargetLanguage != targetLanguage {
			errors = append(errors, ValidationError{
				Code: InconsistentTargetLanguage,
				Message: fmt.Sprintf("File '%s' has inconsistent 'target-language' attribute '%s'",
					file.Original, file.TargetLanguage),
			})
		}
	}

	// Make sure all trans units have the attributes and children we expect
	for _, file := range d.Files {
		for idx, transUnit := range file.Body.TransUnits {
			if transUnit.ID == "" {
				errors = append(errors, ValidationError{
					Code: MissingTransUnitID,
					Message: fmt.Sprintf("Translation unit #%d in file '%s' is missing 'id' attribute",
						idx, file.Original),
				})
			}
			if transUnit.Source == "" {
				errors = append(errors, ValidationError{
					Code: MissingTransUnitSource,
					Message: fmt.Sprintf("Translation unit '%s' in file '%s' is missing 'source' attribute",
						transUnit.ID, file.Original),
				})
			}
			if transUnit.Target == "" {
				errors = append(errors, ValidationError{
					Code: MissingTransUnitTarget,
					Message: fmt.Sprintf("Translation unit '%s' in file '%s' is missing 'target' attribute",
						transUnit.ID, file.Original),
				})
			}
		}
	}

	return errors
}

// Returns true if all translation units in all files have both a
// non-empty source and target.
func (d *Document) IsComplete() bool {
	for _, file := range d.Files {
		for _, transUnit := range file.Body.TransUnits {
			if transUnit.Source == "" || transUnit.Target == "" {
				return false
			}
		}
	}
	return true
}

// finds a specific File within a document
func (d *Document) File(original string) (File, bool) {
	for _, file := range d.Files {
		if file.Original == original {
			return file, true
		}
	}
	return File{}, false
}

// Adds a TransUnit to the last File within a Document
//
// The first optional argument is the target, the second optional argument is a note
func (d *Document) AddTransUnit(source string, opts ...func(*TransUnit)) error {
	if len(d.Files) == 0 {
		return errors.New("document does not contain a file")
	}

	lastId, ok := d.lastId()
	if !ok {
		return errors.New("could not parse last TransUnit ID")
	}

	numId, err := strconv.Atoi(lastId)
	if err != nil {
		return errors.New(fmt.Sprint("last TransUnit ID is not a number:", lastId))
	}

	tu := TransUnit{
		ID:     strconv.Itoa(numId + 1),
		Source: source,
	}

	for _, opt := range opts {
		opt(&tu)
	}

	file := &d.Files[len(d.Files)-1]
	file.Body.TransUnits = append(file.Body.TransUnits, tu)

	return nil
}

// Returns the last ID of the last File
func (d *Document) lastId() (string, bool) {
	if len(d.Files) == 0 {
		return "", false
	}
	file := d.Files[len(d.Files)-1]
	if len(file.Body.TransUnits) == 0 {
		return "-1", true
	}
	last := file.Body.TransUnits[len(file.Body.TransUnits)-1]

	return last.ID, true
}

type ValidationErrorCode int

const (
	UnsupportedVersion ValidationErrorCode = iota
	MissingOriginalAttribute
	MissingSourceLanguage
	MissingTargetLanguage
	UnsupportedDatatype
	InconsistentSourceLanguage
	InconsistentTargetLanguage
	MissingTransUnitID
	MissingTransUnitSource
	MissingTransUnitTarget
)

type ValidationError struct {
	Code    ValidationErrorCode
	Message string
}

func (ve ValidationError) Error() string {
	code := "Unknown"
	switch ve.Code {
	case UnsupportedVersion:
		code = "UnsupportedVersion"
	case MissingOriginalAttribute:
		code = "MissingOriginalAttribute"
	case MissingSourceLanguage:
		code = "MissingSourceLanguage"
	case MissingTargetLanguage:
		code = "MissingTargetLanguage"
	case UnsupportedDatatype:
		code = "UnsupportedDatatype"
	case InconsistentSourceLanguage:
		code = "InconsistentSourceLanguage"
	case InconsistentTargetLanguage:
		code = "InconsistentTargetLanguage"
	case MissingTransUnitID:
		code = "MissingTransUnitID"
	case MissingTransUnitSource:
		code = "MissingTransUnitSource"
	case MissingTransUnitTarget:
		code = "MissingTransUnitTarget"
	}
	return fmt.Sprintf("%s: %s", code, ve.Message)
}

type TransUnit struct {
	ID     string `xml:"id,attr"`
	Source string `xml:"source"`
	Target string `xml:"target"`
	Note   string `xml:"note"`
}

func WithNote(note string) func(*TransUnit) {
	return func(t *TransUnit) {
		t.Note = note
	}
}

func WithTarget(target string) func(*TransUnit) {
	return func(t *TransUnit) {
		t.Target = target
	}
}
