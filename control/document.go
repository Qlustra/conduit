package control

import "fmt"

// Layout returns the exact content carrier controlled by this document,
// preserving its concrete codec or template API.
func (d *DocumentCtrl[L]) Layout() *L { return d.layout }

func (d *DocumentCtrl[L]) BindDocument(tree any) error {
	live, ok := tree.(*L)
	if !ok || live == nil {
		return fmt.Errorf("document: expected *%T, got %T", *new(L), tree)
	}
	d.layout = live
	return nil
}
