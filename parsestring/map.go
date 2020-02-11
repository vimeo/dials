package parsestring

import (
	"fmt"
	"reflect"
)

// ParseMap converts a string representation of a map with concrete values as
// keys and vals into a reflect.Value representing that map.
func ParseMap(s string, mapType reflect.Type) (reflect.Value, error) {
	m := reflect.MakeMap(mapType)
	keyType := mapType.Key()
	valType := mapType.Elem()

	splitErr := splitMap(s,
		func(newKeyStr, newValStr string) error {
			newKeyCast, err := ParseString(newKeyStr, keyType)
			if err != nil {
				return fmt.Errorf("Error casting map key")
			}

			val := m.MapIndex(newKeyCast.Elem())
			if val.IsValid() {
				return fmt.Errorf("duplicate key %q, already has value %q", newKeyCast.Elem(), val)
			}

			newValCast, err := ParseString(newValStr, valType)
			if err != nil {
				return fmt.Errorf("Error casting map val")
			}

			m.SetMapIndex(newKeyCast.Elem(), newValCast.Elem())

			return nil
		})

	if splitErr != nil {
		return reflect.Value{}, splitErr
	}

	return m, nil
}
