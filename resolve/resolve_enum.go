package resolve

import (
	"fmt"
	"reflect"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/rgraphql/magellan/types"
)

var stringTypeRef *ast.Named = &ast.Named{
	Kind: "Named",
	Name: &ast.Name{
		Kind:  "Name",
		Value: "String",
	},
}

type enumResolver struct {
	useName             bool
	possibleValues      map[string]int
	possibleValuesIndex []string

	valueResolver Resolver
	convertTo     reflect.Type
}

func (er *enumResolver) Execute(rc *resolutionContext, value reflect.Value) {
	if !value.IsValid() || (value.Kind() == reflect.Ptr && value.IsNil()) {
		rc.SetValue(nil)
		return
	}

	if er.useName {
		// This is asserted in buildEnumResolver.
		sval := value.Interface().(string)
		_, ok := er.possibleValues[sval]
		if !ok {
			rc.SetError(fmt.Errorf("Enum value %s not in list of possible values.", sval))
			return
		}
	} else {
		if er.convertTo != nil {
			value = value.Convert(er.convertTo)
		}
		ival := value.Interface().(int)
		if ival < 0 || ival > len(er.possibleValuesIndex) {
			rc.SetError(fmt.Errorf("Enum value %d not in list of possible values.", ival))
			return
		}
	}

	go er.valueResolver.Execute(rc, value)
}

func (rt *ResolverTree) buildEnumResolver(value reflect.Type, etyp *ast.EnumDefinition) (Resolver, error) {
	useName := value.Kind() == reflect.String
	needsConvert := false
	intType := types.GraphQLPrimitivesTypes["Int"]
	if !useName && value.Kind() != reflect.Int {
		if value.ConvertibleTo(intType) {
			needsConvert = true
		} else {
			return nil, fmt.Errorf("Enum values can be expressed with an int or string, not %s.", value.String())
		}
	}

	resolver := &enumResolver{useName: useName}
	if needsConvert {
		resolver.convertTo = intType
	}
	if useName {
		resolver.possibleValues = make(map[string]int)
	} else {
		resolver.possibleValuesIndex = make([]string, len(etyp.Values))
	}

	for idx, val := range etyp.Values {
		if useName {
			resolver.possibleValues[val.Name.Value] = idx
		} else {
			resolver.possibleValuesIndex[idx] = val.Name.Value
		}
	}

	vr, err := rt.BuildResolver(TypeResolverPair{
		GqlType:      stringTypeRef,
		ResolverType: value,
	})
	if err != nil {
		return nil, err
	}
	resolver.valueResolver = vr
	return resolver, nil
}
