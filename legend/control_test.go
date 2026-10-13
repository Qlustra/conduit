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
	if project.Layout.Get() != reflect.TypeFor[programLayout]() || chart.Lane.ProjectsSpot.Value[1].Layout.Get() != reflect.TypeFor[otherLayout]() {
		t.Fatal("project declarations lost their controlled layout types")
	}
	if len(project.Lane.DocumentsSpot.Value) != 1 || len(project.Lane.CollectionsSpot.Value) != 1 || len(project.Lane.ScopesSpot.Value) != 1 {
		t.Fatalf("project children = documents %d, collections %d, scopes %d, want one each",
			len(project.Lane.DocumentsSpot.Value), len(project.Lane.CollectionsSpot.Value), len(project.Lane.ScopesSpot.Value))
	}
	if len(project.Lane.ScopesSpot.Value[0].Lane.DocumentsSpot.Value) != 1 {
		t.Fatal("nested scope did not retain its document declaration")
	}
	if project.Lane.ScopesSpot.Value[0].Layout.Get() != reflect.TypeFor[commandLayout]() {
		t.Fatal("nested scope declaration lost its controlled layout type")
	}

	// The external authored wrappers retain role contracts and exact type metadata.
	var surface programControl
	var projectCarrier control.ProjectCarrier = surface.Program
	var scopeCarrier control.ScopeCarrier = surface.Program
	var collectionCarrier control.CollectionCarrier = surface.Program.Commands
	if projectCarrier.LayoutCarrierType() != reflect.TypeFor[programLayout]() {
		t.Fatal("project carrier lost its layout declaration type")
	}
	if scopeCarrier.LayoutCarrierType() != projectCarrier.LayoutCarrierType() {
		t.Fatal("project's scope contract lost its layout declaration type")
	}
	if collectionCarrier.LayoutCarrierType() != reflect.TypeFor[layout.Slot[*commandLayout]]() || collectionCarrier.MemberControlType() != reflect.TypeFor[commandControl]() {
		t.Fatal("collection carrier lost its source or member control type")
	}
}

func TestControlLegendPopulatesIndependentWholeAndFocusedScopes(t *testing.T) {
	type standalone struct {
		Whole   control.ScopeCtrl[programLayout] `control:"scope"`
		Command commandControl                   `control:"scope"`
	}
	charter, err := sail.CompileLegend[legend.ControlLegend]()
	if err != nil {
		t.Fatal(err)
	}
	chart, err := sail.Populate[standalone](charter)
	if err != nil {
		t.Fatal(err)
	}
	if len(chart.Lane.ProjectsSpot.Value) != 0 || len(chart.Lane.ScopesSpot.Value) != 2 {
		t.Fatalf("root controls = projects %d, scopes %d, want 0 and 2",
			len(chart.Lane.ProjectsSpot.Value), len(chart.Lane.ScopesSpot.Value))
	}
	if len(chart.Lane.ScopesSpot.Value[1].Lane.DocumentsSpot.Value) != 1 {
		t.Fatal("independent focused scope lost its document declaration")
	}
	if chart.Lane.ScopesSpot.Value[0].Layout.Get() != reflect.TypeFor[programLayout]() || chart.Lane.ScopesSpot.Value[1].Layout.Get() != reflect.TypeFor[commandLayout]() {
		t.Fatal("whole and focused scope declarations lost their controlled layout types")
	}
	var surface standalone
	if surface.Whole.LayoutCarrierType() != reflect.TypeFor[programLayout]() || surface.Command.LayoutCarrierType() != reflect.TypeFor[commandLayout]() {
		t.Fatal("whole and focused scopes lost their controlled layout types")
	}
}

func TestControlLegendAcceptsProjectThroughScopeContract(t *testing.T) {
	type surface struct {
		Program projectControl `control:"scope"`
	}
	charter, err := sail.CompileLegend[legend.ControlLegend]()
	if err != nil {
		t.Fatal(err)
	}
	chart, err := sail.Populate[surface](charter)
	if err != nil {
		t.Fatal(err)
	}
	if len(chart.Lane.ScopesSpot.Value) != 1 || len(chart.Lane.ProjectsSpot.Value) != 0 {
		t.Fatal("project was not interpreted through its declared scope role")
	}
	if chart.Lane.ScopesSpot.Value[0].Layout.Get() != reflect.TypeFor[programLayout]() {
		t.Fatal("project's scope declaration lost its controlled layout type")
	}
	members := chart.Lane.ScopesSpot.Value[0].Lane
	if len(members.DocumentsSpot.Value) != 1 || len(members.CollectionsSpot.Value) != 1 || len(members.ScopesSpot.Value) != 1 {
		t.Fatal("project's scope view lost its exposed members")
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
	t.Run("root rejects document peers", func(t *testing.T) {
		type invalid struct {
			Project  control.ProjectCtrl[programLayout]               `control:"project"`
			Document control.DocumentCtrl[formats.YAMLFile[manifest]] `control:"document"`
		}
		requireRejectedControl[invalid](t, charter)
	})

	t.Run("space is not a developing scoping role", func(t *testing.T) {
		type invalid struct {
			Space control.ScopeCtrl[programLayout] `control:"space"`
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
