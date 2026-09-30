package release

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type signPathConfiguration struct {
	XMLName    xml.Name            `xml:"artifact-configuration"`
	Parameters []signPathParameter `xml:"parameters>parameter"`
	Root       signPathZIP         `xml:"zip-file"`
}

type signPathParameter struct {
	Name     string `xml:"name,attr"`
	Required bool   `xml:"required,attr"`
}

type signPathZIP struct {
	Path string        `xml:"path,attr"`
	ZIPs []signPathZIP `xml:"zip-file"`
	PEs  []signPathPE  `xml:"pe-file"`
}

type signPathPE struct {
	Path             string               `xml:"path,attr"`
	ProductName      string               `xml:"product-name,attr"`
	ProductVersion   string               `xml:"product-version,attr"`
	FileVersion      string               `xml:"file-version,attr"`
	CompanyName      string               `xml:"company-name,attr"`
	Copyright        string               `xml:"copyright,attr"`
	OriginalFilename string               `xml:"original-filename,attr"`
	Sign             signPathAuthenticode `xml:"authenticode-sign"`
}

type signPathAuthenticode struct {
	HashAlgorithm  string `xml:"hash-algorithm,attr"`
	Description    string `xml:"description,attr"`
	DescriptionURL string `xml:"description-url,attr"`
}

func TestSignPathArtifactConfiguration(t *testing.T) {
	configurationPath := filepath.Join("..", "..", ".signpath", "artifact-configuration.xml")
	content, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatal(err)
	}
	var configuration signPathConfiguration
	if err := xml.Unmarshal(content, &configuration); err != nil {
		t.Fatalf("invalid SignPath XML: %v", err)
	}
	wantParameters := []signPathParameter{{Name: "version", Required: true}, {Name: "pe-version", Required: true}}
	if !reflect.DeepEqual(configuration.Parameters, wantParameters) {
		t.Fatalf("unexpected SignPath parameters: %#v", configuration.Parameters)
	}
	if configuration.Root.Path != "" || len(configuration.Root.ZIPs) != 2 {
		t.Fatalf("expected one root ZIP containing two Windows ZIPs: %#v", configuration.Root)
	}
	wantZIPs := []string{"arc_v${version}_windows_amd64.zip", "arc_v${version}_windows_arm64.zip"}
	for index, archive := range configuration.Root.ZIPs {
		if archive.Path != wantZIPs[index] || len(archive.PEs) != 1 {
			t.Fatalf("unexpected nested ZIP at index %d: %#v", index, archive)
		}
		pe := archive.PEs[0]
		if pe.Path != windowsOriginalName || pe.ProductName != windowsProductName ||
			pe.ProductVersion != "${pe-version}" || pe.FileVersion != "${pe-version}" ||
			pe.CompanyName != windowsCompanyName || pe.Copyright != windowsCopyright ||
			pe.OriginalFilename != windowsOriginalName {
			t.Fatalf("unexpected PE restrictions for %s: %#v", archive.Path, pe)
		}
		if pe.Sign.HashAlgorithm != "sha256" || pe.Sign.Description != windowsDescription ||
			pe.Sign.DescriptionURL != "https://github.com/EnzoCaetano015/Archbase" {
			t.Fatalf("unexpected Authenticode directive for %s: %#v", archive.Path, pe.Sign)
		}
	}
}
