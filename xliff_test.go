// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/

package xliff_test

import (
	"bytes"
	"encoding/xml"
	"os"
	"strings"
	"testing"

	"github.com/truesch/xliff"
)

func TestParse(t *testing.T) {
	if _, err := xliff.FromFile("testdata/focus-ios-ar.xliff"); err != nil {
		t.Error("Could not parse testdata/focus-ios-ar.xliff:", err)
	}
	if _, err := xliff.FromFile("testdata/focus-ios-it.xliff"); err != nil {
		t.Error("Could not parse testdata/focus-ios-it.xliff:", err)
	}
}

func Test_Save(t *testing.T) {
	doc, err := xliff.FromFile("testdata/focus-ios-ar.xliff")
	if err != nil {
		t.Error("Could not parse testdata/focus-ios-ar.xliff:", err)
	}

	// save file back to disk
	if err := doc.ToFile("testdata/focus-ios-ar-duplicate.xliff"); err != nil {
		t.Error("Could not save document to testdata/focus-ios-ar-duplicate.xliff:", err)
	}
}

func Test_ParseNonExistentFile(t *testing.T) {
	if _, err := xliff.FromFile("testdata/doesnotexist.xliff"); !os.IsNotExist(err) {
		t.Error("Unexpected error when opening testdata/doesnotexist.xliff:", err)
	}
}

func Test_ParseBadXMLFile(t *testing.T) {
	_, err := xliff.FromFile("testdata/badxml.xliff")
	if _, ok := err.(*xml.SyntaxError); !ok {
		t.Error("Unexpected error when opening testdata/badxml.xliff:", err)
	}
}

func Test_ValidateGood(t *testing.T) {
	doc, err := xliff.FromFile("testdata/good.xliff")
	if err != nil {
		t.Error("Could not parse testdata/good.xliff:", err)
	}

	if errors := doc.Validate(); errors != nil {
		t.Error("Unexpected error from Validate()")
	}
}

func Test_ValidateGoodSave(t *testing.T) {
	doc, err := xliff.FromFile("testdata/good.xliff")
	if err != nil {
		t.Error("Could not parse testdata/good.xliff:", err)
	}

	// save file back to disk
	if err := doc.ToFile("testdata/good-duplicate.xliff"); err != nil {
		t.Error("Could not save document to testdata/good-duplicate.xliff:", err)
	}

	// re-read duplicate
	duplicate, err := xliff.FromFile("testdata/good-duplicate.xliff")
	if err != nil {
		t.Error("Could not parse testdata/good-duplicate.xliff:", err)
	}

	// compare if original and duplicate are identical
	o, _ := xml.Marshal(doc)
	d, _ := xml.Marshal(duplicate)

	bytes.Compare(o, d)
}

func containsValidationError(t *testing.T, errors []xliff.ValidationError, code xliff.ValidationErrorCode) bool {
	for _, err := range errors {
		if err.Code == code {
			if strings.HasPrefix(err.Error(), "Unknown: ") {
				t.Error("Error has no good message: ", err)
			}
			return true
		}
	}
	return false
}

func Test_ValidateErrors(t *testing.T) {
	doc, err := xliff.FromFile("testdata/errors.xliff")
	if err != nil {
		t.Error("Could not parse testdata/errors.xliff:", err)
	}

	errors := doc.Validate()
	if len(errors) == 0 {
		t.Error("Expected error from Validate()")
	}

	if !containsValidationError(t, errors, xliff.UnsupportedVersion) {
		t.Error("Expected validation to fail with UnsupportedVersion")
	}

	if !containsValidationError(t, errors, xliff.MissingOriginalAttribute) {
		t.Error("Expected validation to fail with MissingOriginalAttribute")
	}

	if !containsValidationError(t, errors, xliff.MissingSourceLanguage) {
		t.Error("Expected validation to fail with MissingSourceLanguage")
	}

	if !containsValidationError(t, errors, xliff.MissingTargetLanguage) {
		t.Error("Expected validation to fail with MissingTargetLanguage")
	}

	if !containsValidationError(t, errors, xliff.UnsupportedDatatype) {
		t.Error("Expected validation to fail with UnsupportedDatatype")
	}

	if !containsValidationError(t, errors, xliff.InconsistentSourceLanguage) {
		t.Error("Expected validation to fail with InconsistentSourceLanguage")
	}

	if !containsValidationError(t, errors, xliff.InconsistentTargetLanguage) {
		t.Error("Expected validation to fail with InconsistentTargetLanguage")
	}

	if !containsValidationError(t, errors, xliff.MissingTransUnitID) {
		t.Error("Expected validation to fail with MissingTransUnitID")
	}

	if !containsValidationError(t, errors, xliff.MissingTransUnitSource) {
		t.Error("Expected validation to fail with MissingTransUnitSource")
	}

	if !containsValidationError(t, errors, xliff.MissingTransUnitTarget) {
		t.Error("Expected validation to fail with MissingTransUnitTarget")
	}
}

func Test_IsComplete(t *testing.T) {
	doc, err := xliff.FromFile("testdata/complete.xliff")
	if err != nil {
		t.Error("Could not parse testdata/complete.xliff:", err)
	}

	if !doc.IsComplete() {
		t.Error("Unexpected result from doc.IsComplete(). Got false, expected true")
	}
}

func Test_IsInComplete(t *testing.T) {
	doc, err := xliff.FromFile("testdata/incomplete.xliff")
	if err != nil {
		t.Error("Could not parse testdata/incomplete.xliff:", err)
	}

	if doc.IsComplete() {
		t.Error("Unexpected result from doc.IsComplete(). Got true, expected false")
	}
}

func Test_File(t *testing.T) {
	doc, err := xliff.FromFile("testdata/complete.xliff")
	if err != nil {
		t.Error("Could not parse testdata/complete.xliff:", err)
	}

	if _, found := doc.File("One.strings"); found != true {
		t.Error("Unexpected result from doc.File(One.strings)")
	}
	if _, found := doc.File("Unknown.strings"); found != false {
		t.Error("Unexpected result from doc.File(Unknown.strings)")
	}
}

func Test_CreateEmptyXLIFF(t *testing.T) {
	file := xliff.File{
		Datatype:       "plaintext",
		SourceLanguage: "de",
		TargetLanguage: "en",
		Header:         xliff.Header{},
		Body:           xliff.Body{},
	}
	doc := xliff.Document{
		Version: "1.2",
		Files:   []xliff.File{file},
	}

	doc.Validate()
}

func Test_CreateXLIFF(t *testing.T) {
	file := xliff.File{
		Datatype:       "plaintext",
		SourceLanguage: "de",
		TargetLanguage: "en",
		Header:         xliff.Header{},
		Body:           xliff.Body{},
	}
	doc := xliff.Document{
		Version: "1.2",
		Files:   []xliff.File{file},
	}

	tu := xliff.TransUnit{
		ID:     "0",
		Source: "Hallo Welt",
		Target: "Hello World",
		Note:   "Some Comment",
	}

	tu2 := xliff.TransUnit{
		ID:     "1",
		Source: "Auf Wiedersehen, Welt",
		Target: "Goodbye World",
		Note:   "Some Comment",
	}

	doc.Files[0].Body.TransUnits = []xliff.TransUnit{tu, tu2}

	doc.ToFile("testdata/test.xliff")
}

func Test_CreateXLIFFBuiltIn(t *testing.T) {
	doc := xliff.NewDocument("de", "en")
	doc.AddTransUnit("Hallo Welt")

	doc.Validate()

	doc.AddTransUnit("Wie geht es dir?", xliff.WithNote("This is a test."), xliff.WithTarget("How are you?"))

	doc.ToFile("testdata/test_built_in.xliff")
}
