package composer

import (
	"fmt"
	"reflect"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail"
)

// validateMemberCarrier checks complete carrier APIs at collection roots,
// which enter through a Layout marker rather than a tagged carrier landmark.
// Ordinary composite member declarations have no carrier witness to validate.
func validateMemberCarrier(carrierType reflect.Type) error {
	typ := sail.ResolvePointerChain(carrierType)
	if typ == nil {
		return fmt.Errorf("compose: cannot resolve member carrier %v", carrierType)
	}
	// Contract selection is declaration metadata. Use a fresh zero-valued
	// root without allocating participating fields or inspecting live members.
	prototype := reflect.New(typ)
	contract, known := layout.CarrierContract(prototype.Interface())
	if known && !prototype.Type().Implements(contract) {
		return fmt.Errorf("compose: member carrier %v requires complete carrier API %v", carrierType, contract)
	}
	return nil
}
