package legend_test

import (
	"reflect"
	"testing"

	"github.com/qlustra/conduit/control"
	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
)

type manifest struct{}
type programLayout struct{}
type commandLayout struct{}
type otherLayout struct{}

type commandControl struct {
	control.ScopeCtrl[commandLayout]
	Config control.DocumentCtrl[formats.YAMLFile[manifest]] `control:"document"`
}

type projectControl struct {
	control.ProjectCtrl[programLayout]
	Manifest control.DocumentCtrl[formats.YAMLFile[manifest]]                    `control:"document"`
	Commands control.CollectionCtrl[commandControl, layout.Slot[*commandLayout]] `control:"collection"`
	Scope    commandControl                                                      `control:"scope"`
}

type programControl struct {
	Program projectControl                   `control:"project"`
	Other   control.ProjectCtrl[otherLayout] `control:"project"`
}

func TestControlLegendPopulatesAuthoredProjectScopes(t *testing.T) {
	charter, err := sail.CompileLegend[legend.ControlLegend]()
	if err != nil {
		t.Fatal(err)
	}
	chart, err := sail.Populate[programControl](charter)
	if err != nil {
		t.Fatal(err)
	}
	if len(chart.Lane.ProjectsSpot.Value) != 2 {
		t.Fatalf("projects = %d, want 2", len(chart.Lane.ProjectsSpot.Value))
	}
	project := chart.Lane.ProjectsSpot.Value[0]
	if len(project.Lane.DocumentsSpot.Value) != 1 || len(project.Lane.CollectionsSpot.Value) != 1 || len(project.Lane.ScopesSpot.Value) != 1 {
		t.Fatalf("project children = documents %d, collections %d, scopes %d, want one each",
			len(project.Lane.DocumentsSpot.Value), len(project.Lane.CollectionsSpot.Value), len(project.Lane.ScopesSpot.Value))
	}
	if len(project.Lane.ScopesSpot.Value[0].Lane.DocumentsSpot.Value) != 1 {
		t.Fatal("nested scope did not retain its document declaration")
	}

	// The external authored wrappers retain role contracts and exact type metadata.
	var surface programControl
	var projectCarrier control.ProjectCarrier = surface.Program
	var collectionCarrier control.CollectionCarrier = surface.Program.Commands
	if projectCarrier.LayoutCarrierType() != reflect.TypeFor[programLayout]() {
		t.Fatal("project carrier lost its layout declaration type")
	}
	if collectionCarrier.LayoutCarrierType() != reflect.TypeFor[layout.Slot[*commandLayout]]() || collectionCarrier.MemberControlType() != reflect.TypeFor[commandControl]() {
		t.Fatal("collection carrier lost its source or member control type")
	}
}

type metadataOnly struct{}

func (metadataOnly) LayoutCarrierType() reflect.Type {
	return reflect.TypeFor[programLayout]()
}

func TestControlLegendRejectsWrongRolesAndRootPeers(t *testing.T) {
	charter, err := sail.CompileLegend[legend.ControlLegend]()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("document cannot declare project", func(t *testing.T) {
		type invalid struct {
			Project control.DocumentCtrl[programLayout] `control:"project"`
		}
		requireRejectedControl[invalid](t, charter)
	})
	t.Run("metadata alone does not identify project", func(t *testing.T) {
		type invalid struct {
			Project metadataOnly `control:"project"`
		}
		requireRejectedControl[invalid](t, charter)
	})
	t.Run("document cannot declare nested scope", func(t *testing.T) {
		type invalidProject struct {
			control.ProjectCtrl[programLayout]
			Scope control.DocumentCtrl[commandLayout] `control:"scope"`
		}
		type invalid struct {
			Project invalidProject `control:"project"`
		}
		requireRejectedControl[invalid](t, charter)
	})
	t.Run("root contains only projects", func(t *testing.T) {
		type invalid struct {
			Project  control.ProjectCtrl[programLayout]               `control:"project"`
			Document control.DocumentCtrl[formats.YAMLFile[manifest]] `control:"document"`
		}
		requireRejectedControl[invalid](t, charter)
	})
}

func requireRejectedControl[T any](t *testing.T, charter *sail.Charter[legend.ControlLegend]) {
	t.Helper()
	if _, err := sail.Populate[T](charter); err == nil {
		t.Fatal("invalid control declaration was accepted")
	}
}
