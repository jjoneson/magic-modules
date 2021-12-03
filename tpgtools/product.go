package main

import (
	"log"
	"strings"

	"github.com/golang/glog"
	"github.com/nasa9084/go-openapi"
)

type Filepath string

type DCLPackageName string

func (d DCLPackageName) lowercase() string {
	return string(d)
}

type ProductMetadata struct {
	// PackagePath is the path to the package relative to the dcl
	PackagePath Filepath
	// PackageName is the namespace of the package within the dcl
	// the PackageName is normally a lowercase variant of ProductName
	PackageName DCLPackageName
	// ProductName is the case accounted (snake case) name of the product
	// that the resource belongs to.
	ProductName SnakeCaseProductName
}

var productOverrides map[Filepath]Overrides = make(map[Filepath]Overrides, 0)

func GetProductMetadataFromDocument(document *openapi.Document, packagePath Filepath) *ProductMetadata {
	// load overrides for product
	if _, ok := productOverrides[packagePath]; !ok {
		productOverrides[packagePath] = loadOverrides(packagePath, "tpgtools_product.yaml")
	}
	title := getProductTitle(document.Info.Title, packagePath)
	productMetadata := NewProductMetadata(packagePath, title)
	return productMetadata
}

func NewProductMetadata(packagePath Filepath, productName string) *ProductMetadata {
	packageName := strings.Split(string(packagePath), "/")[0]
	return &ProductMetadata{
		PackagePath: packagePath,
		PackageName: DCLPackageName(packageName),
		ProductName: SnakeCaseProductName(jsonToSnakeCase(productName)),
	}
}

func (pm *ProductMetadata) ShouldWriteProductBasePath() bool {
	bp := pm.ProductBasePathDetails()
	if bp == nil {
		return true
	}
	return !bp.Skip
}

type BasePathOverrideNameSnakeCase string

func (b BasePathOverrideNameSnakeCase) snakecase() string {
	return string(b)
}

func (b BasePathOverrideNameSnakeCase) ToUpper() string {
	return strings.ToUpper(string(b))
}

func (b BasePathOverrideNameSnakeCase) ToTitle() string {
	title := snakeToTitleCase(b).titlecase()
	// Got to special case the capitalization of "OS" in "OSConfig", for base paths specifically,
	// because of interop with MMv1.
	if strings.HasPrefix(string(b), "os") {
		return "OS" + title[2:]
	}
	return title
}

func (s SnakeCaseProductName) ToTitle() string {
	return snakeToTitleCase(s).titlecase()
}

func (pm *ProductMetadata) BasePathIdentifier() BasePathOverrideNameSnakeCase {
	bp := pm.ProductBasePathDetails()
	if bp != nil && bp.BasePathIdentifier != "" {
		return BasePathOverrideNameSnakeCase(bp.BasePathIdentifier)
	}
	return BasePathOverrideNameSnakeCase(pm.ProductName)
}

func (pm *ProductMetadata) ProductBasePathDetails() *ProductBasePathDetails {
	overrides, ok := productOverrides[pm.PackagePath]
	if !ok {
		glog.Fatalf("product overrides should be loaded already for packagePath %s", pm.PackagePath)
	}
	bp := ProductBasePathDetails{}
	bpOk, err := overrides.ProductOverrideWithDetails(ProductBasePath, &bp)
	if err != nil {
		log.Fatalln("error - failed to decode base path details")
	}

	if !bpOk {
		return nil
	}

	return &bp
}

// getProductTitle is used internally to get the product title
// or case sensitve product name from the product definition
// we will also check if there is an override for the product title
// and utilize that if avalible and set.
func getProductTitle(documentTitle string, packagePath Filepath) string {
	overrides, ok := productOverrides[packagePath]
	if !ok {
		glog.Fatalf("product overrides should be loaded already for packagePath %s", packagePath)
	}

	pt := ProductTitleDetails{}
	ptOk, err := overrides.ProductOverrideWithDetails(ProductTitle, &pt)
	if err != nil {
		glog.Fatalln("error - failed to decode base path details")
	}

	if ptOk {
		if pt.Title == "" {
			glog.Fatalf("error - product title override defined but got empty value for %", packagePath)
		}
		title := pt.Title
		return title
	}

	titleParts := strings.Split(documentTitle, "/")
	if len(titleParts) < 0 {
		glog.Exitf("could not find product information for %s", packagePath)
	}
	title := titleParts[0]
	return title
}

func (pm *ProductMetadata) DocsSection() miscellaneousNameLowercase {
	overrides, ok := productOverrides[pm.PackagePath]
	if !ok {
		glog.Fatalf("product overrides should be loaded already for packagePath %s", pm.PackagePath)
	}
	pt := ProductDocsSectionDetails{}
	ptOk, err := overrides.ProductOverrideWithDetails(ProductDocsSection, &pt)
	if err != nil {
		glog.Fatalf("could not parse override %v", err)
	}
	if ptOk {
		return miscellaneousNameLowercase(pt.DocsSection)
	}

	return miscellaneousNameLowercase(pm.PackageName)
}

type DCLPackage string

// DCLPackage is the package name of the DCL client library to use for this
// resource. For example, the Package "access_context_manager" would have a
// DCLPackage of "accesscontextmanager"
func (pm *ProductMetadata) DCLPackage() DCLPackage {
	return DCLPackage(strings.Replace(string(pm.PackagePath), "_", "", -1))
}
