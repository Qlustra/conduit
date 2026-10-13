package legend_test

import (
	"testing"

	"github.com/qlustra/conduit/control"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
)

type selectedFile struct {
	control.SelectedDocumentCtrl[programLayout, manifest]
}

func (*selectedFile) Control(*programLayout) *manifest { panic("selection must not run at population") }

type selectedScope struct {
	control.SelectedScopeCtrl[programLayout, commandLayout]
}

func (*selectedScope) Control(*programLayout) *commandLayout {
	panic("selection must not run at population")
}

type selectedProject struct {
	control.SelectedProjectCtrl[otherLayout, programLayout]
}

func (*selectedProject) Control(*otherLayout) *programLayout {
	panic("selection must not run at population")
}

type selectedCollection struct {
	control.SelectedCollectionCtrl[programLayout, commandControl, layout.Slot[*commandLayout]]
}

func (*selectedCollection) Control(*programLayout) *layout.Slot[*commandLayout] {
	panic("selection must not run at population")
}

type familySurface struct {
	Program struct {
		selectedProject
		File     selectedFile       `control:"document"`
		Scope    selectedScope      `control:"scope"`
		Commands selectedCollection `control:"collection"`
	} `control:"project"`
}

func TestControlFamiliesAcceptInstantiatedHooksWithoutInvokingThem(t *testing.T) {
	charter, err := sail.CompileLegend[legend.ControlLegend]()
	if err != nil {
		t.Fatal(err)
	}
	chart, err := sail.Populate[familySurface](charter)
	if err != nil {
		t.Fatal(err)
	}
	if len(chart.Lane.ProjectsSpot.Value) != 1 {
		t.Fatal("missing selected project")
	}
	lane := chart.Lane.ProjectsSpot.Value[0].Lane
	if len(lane.DocumentsSpot.Value) != 1 || len(lane.ScopesSpot.Value) != 1 || len(lane.CollectionsSpot.Value) != 1 {
		t.Fatal("shared legend omitted selected carrier roles")
	}
}

type wrongInput struct {
	control.SelectedDocumentCtrl[programLayout, manifest]
}

func (*wrongInput) Control(*otherLayout) *manifest { return nil }

type wrongOutput struct {
	control.SelectedDocumentCtrl[programLayout, manifest]
}

func (*wrongOutput) Control(*programLayout) *otherLayout { return nil }

type variadicSelection struct {
	control.SelectedDocumentCtrl[programLayout, manifest]
}

func (*variadicSelection) Control(...*programLayout) *manifest { return nil }

func TestControlFamiliesRejectMissingAndMismatchedHooks(t *testing.T) {
	charter, err := sail.CompileLegend[legend.ControlLegend]()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("missing hook", func(t *testing.T) {
		type surface struct {
			Program struct {
				control.ScopeCtrl[programLayout]
				File control.SelectedDocumentCtrl[programLayout, manifest] `control:"document"`
			} `control:"scope"`
		}
		requireRejectedControl[surface](t, charter)
	})
	t.Run("wrong input", func(t *testing.T) {
		type surface struct {
			Program struct {
				control.ScopeCtrl[programLayout]
				File wrongInput `control:"document"`
			} `control:"scope"`
		}
		requireRejectedControl[surface](t, charter)
	})
	t.Run("wrong output", func(t *testing.T) {
		type surface struct {
			Program struct {
				control.ScopeCtrl[programLayout]
				File wrongOutput `control:"document"`
			} `control:"scope"`
		}
		requireRejectedControl[surface](t, charter)
	})
	t.Run("variadic input", func(t *testing.T) {
		type surface struct {
			Program struct {
				control.ScopeCtrl[programLayout]
				File variadicSelection `control:"document"`
			} `control:"scope"`
		}
		requireRejectedControl[surface](t, charter)
	})
}
