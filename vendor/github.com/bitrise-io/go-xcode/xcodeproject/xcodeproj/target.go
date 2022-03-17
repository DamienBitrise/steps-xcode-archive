package xcodeproj

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
)

// TargetType ...
type TargetType string

// TargetTypes
const (
	NativeTargetType    TargetType = "PBXNativeTarget"
	AggregateTargetType TargetType = "PBXAggregateTarget"
	LegacyTargetType    TargetType = "PBXLegacyTarget"
)

const appClipProductType = "com.apple.product-type.application.on-demand-install-capable"

// Target ...
type Target struct {
	Type                   TargetType
	ID                     string
	Name                   string
	BuildConfigurationList ConfigurationList
	Dependencies           []TargetDependency
	ProductReference       ProductReference
	ProductType            string
	buildPhaseIDs          []string
}

// DependentTargets ...
func (t Target) DependentTargets() []Target {
	var targets []Target
	for _, targetDependency := range t.Dependencies {
		childTarget := targetDependency.Target
		targets = append(targets, childTarget)

		childDependentTargets := childTarget.DependentTargets()
		targets = append(targets, childDependentTargets...)
	}

	return targets
}

// DependesOn ...
func (t Target) DependesOn(targetID string) bool {
	for _, targetDependency := range t.Dependencies {
		childTarget := targetDependency.Target
		if childTarget.ID == targetID {
			return true
		}
	}
	return false
}

// DependentExecutableProductTargets ...
func (t Target) DependentExecutableProductTargets() []Target {
	var targets []Target
	for _, targetDependency := range t.Dependencies {
		childTarget := targetDependency.Target
		if !childTarget.IsExecutableProduct() {
			continue
		}

		targets = append(targets, childTarget)

		childDependentTargets := childTarget.DependentExecutableProductTargets()
		targets = append(targets, childDependentTargets...)
	}

	return targets
}

// IsAppProduct ...
func (t Target) IsAppProduct() bool {
	return filepath.Ext(t.ProductReference.Path) == ".app"
}

// IsAppExtensionProduct ...
func (t Target) IsAppExtensionProduct() bool {
	return filepath.Ext(t.ProductReference.Path) == ".appex"
}

// IsExecutableProduct ...
func (t Target) IsExecutableProduct() bool {
	return t.IsAppProduct() || t.IsAppExtensionProduct()
}

// IsTest identifies test targets
// Based on https://github.com/CocoaPods/Xcodeproj/blob/907c81763a7660978fda93b2f38f05de0cbb51ad/lib/xcodeproj/project/object/native_target.rb#L470
func (t Target) IsTest() bool {
	return t.IsTestProduct() ||
		t.IsUITestProduct() ||
		t.ProductType == "com.apple.product-type.bundle" // OCTest bundle
}

// IsTestProduct ...
func (t Target) IsTestProduct() bool {
	return filepath.Ext(t.ProductType) == ".unit-test"
}

// IsUITestProduct ...
func (t Target) IsUITestProduct() bool {
	return filepath.Ext(t.ProductType) == ".ui-testing"
}

func (t Target) isAppClipProduct() bool {
	return t.ProductType == appClipProductType
}

// CanExportAppClip ...
func (t Target) CanExportAppClip() bool {
	deps := t.DependentTargets()
	for _, target := range deps {
		if target.isAppClipProduct() {
			return true
		}
	}

	return false
}

func parseTarget(id string, objects serialized.Object) (Target, error) {
	rawTarget, err := objects.Object(id)
	if err != nil {
		return Target{}, err
	}

	isa, err := rawTarget.String("isa")
	if err != nil {
		return Target{}, err
	}

	var targetType TargetType
	switch isa {
	case "PBXNativeTarget":
		targetType = NativeTargetType
	case "PBXAggregateTarget":
		targetType = AggregateTargetType
	case "PBXLegacyTarget":
		targetType = LegacyTargetType
	default:
		return Target{}, fmt.Errorf("unknown target type: %s", isa)
	}

	name, err := rawTarget.String("name")
	if err != nil {
		return Target{}, err
	}

	productType, err := rawTarget.String("productType")
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return Target{}, err
	}

	buildConfigurationListID, err := rawTarget.String("buildConfigurationList")
	if err != nil {
		return Target{}, err
	}
	logger.Infof("[mattrob] xcodeproj - parseConfigurationList(buildConfigurationListID, objects) start")
	buildConfigurationList, err := parseConfigurationList(buildConfigurationListID, objects)
	logger.Infof("[mattrob] xcodeproj - parseConfigurationList(buildConfigurationListID, objects) end")
	if err != nil {
		return Target{}, err
	}

	//dependencyIDs, err := rawTarget.StringSlice("dependencies")
	//if err != nil {
	//	return Target{}, err
	//}

	var dependencies []TargetDependency
	logger.Infof("[mattrob] xcodeproj - TargetDependency start")
	//for _, dependencyID := range dependencyIDs {
	//	logger.Infof("[mattrob] xcodeproj - parseTargetDependency start")
	//	dependency, err := parseTargetDependency(dependencyID, objects)
	//	logger.Infof("[mattrob] xcodeproj - parseTargetDependency end")
	//	if err != nil {
	//		// KeyNotFoundError can be only raised if the 'target' property not found on the raw target dependency object
	//		// we only care about target dependency, which points to a target
	//		if serialized.IsKeyNotFoundError(err) {
	//			continue
	//		} else {
	//			return Target{}, err
	//		}
	//	}
	//	dependencies = append(dependencies, dependency)
	//}
	logger.Infof("[mattrob] xcodeproj - TargetDependency end")

	var productReference ProductReference
	logger.Infof("[mattrob] xcodeproj - ProductReference start")
	productReferenceID, err := rawTarget.String("productReference")
	if err != nil {
		if !serialized.IsKeyNotFoundError(err) {
			return Target{}, err
		}
	} else {
		productReference, err = parseProductReference(productReferenceID, objects)
		if err != nil {
			return Target{}, err
		}
	}
	logger.Infof("[mattrob] xcodeproj - ProductReference end")

	buildPhaseIDs, err := rawTarget.StringSlice("buildPhases")
	if err != nil {
		return Target{}, err
	}

	return Target{
		Type:                   targetType,
		ID:                     id,
		Name:                   name,
		BuildConfigurationList: buildConfigurationList,
		Dependencies:           dependencies,
		ProductReference:       productReference,
		ProductType:            productType,
		buildPhaseIDs:          buildPhaseIDs,
	}, nil
}
