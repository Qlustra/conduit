package legend

import (
	"reflect"

	"github.com/qlustra/conduit/control"
	saillegend "github.com/qlustra/sail/legend"
)

// Earlier metadata contracts and Control are retained alongside the developing
// carrier-based ControlLegend during shaping.

// LayoutDeclaration identifies the storage layout of a Project, Space, or Scope.
// This declaration-time method must be safe on a zero-valued receiver.
type LayoutDeclaration interface {
	LayoutType() reflect.Type
}

// DocumentDeclaration identifies a Document's content type independently of
// its concrete node or codec. The method must be safe on a zero-valued receiver.
type DocumentDeclaration interface {
	ValueType() reflect.Type
}

// CollectionDeclaration identifies a Collection's operational member-view type,
// such as Scope[NotebookLayout]. The method must be safe on a zero-valued receiver.
type CollectionDeclaration interface {
	MemberType() reflect.Type
}

// Control describes the supplied control Subject. Each member has its own
// explicit live binding; declaration nesting never establishes ownership.
type Control struct {
	saillegend.Legend
	Members Members `sail:"lane"`
}

func (Control) Mark() string { return "control" }

// ====

// ControlLegend describes project-led control surfaces and independently
// usable scopes. A project's additional meaning does not change its scope
// members. Documents and collections expose controls over layout carriers.
type ControlLegend struct {
	saillegend.Legend
	Lane struct {
		saillegend.Lane
		ProjectsSpot saillegend.MultiSpot[ProjectLandmark] `sail:"spot" multiplicity:"any"`
		ScopesSpot   saillegend.MultiSpot[ScopeRegion]     `sail:"spot" multiplicity:"any"`
	} `sail:"lane"`
}

func (ControlLegend) Mark() string { return "control" }

// ----

// ProjectLandmark describes a scope with additional project meaning.
// Recognition is a live operation; declaration interpretation does not detect
// or compose a project.
type ProjectLandmark struct {
	saillegend.Region
	Layout saillegend.Barrel[reflect.Type, control.ProjectCarrier] `sail:"barrel" populate:"LayoutCarrierType" barrel:"required"`
	Lane   ScopesLane                                              `sail:"lane"`
}

func (ProjectLandmark) Family() string { return "project" }
func (ProjectLandmark) Accept() saillegend.Acceptance {
	return saillegend.AcceptContractFamily[control.ProjectFamily]()
}

// ----

type DocumentLandmark struct {
	saillegend.Landmark
}

func (DocumentLandmark) Family() string { return "document" }
func (DocumentLandmark) Accept() saillegend.Acceptance {
	return saillegend.AcceptContractFamily[control.DocumentFamily]()
}

// ----

// ScopeRegion describes control over a chosen layout segment. It can appear
// independently or within another scope. Deep operation extent comes from the
// controlled Layout, rather than the exposed control members.
type ScopeRegion struct {
	saillegend.Region
	Layout saillegend.Barrel[reflect.Type, control.ScopeCarrier] `sail:"barrel" populate:"LayoutCarrierType" barrel:"required"`
	Lane   ScopesLane                                            `sail:"lane"`
}

func (ScopeRegion) Family() string { return "scope" }
func (ScopeRegion) Accept() saillegend.Acceptance {
	return saillegend.AcceptContractFamily[control.ScopeFamily]()
}

// ScopesLane exposes the same member vocabulary for projects and scopes.
// Declaration nesting does not itself select storage or establish ownership.
type ScopesLane struct {
	saillegend.Lane
	DocumentsSpot   saillegend.MultiSpot[DocumentLandmark]   `sail:"spot" multiplicity:"any"`
	CollectionsSpot saillegend.MultiSpot[CollectionLandmark] `sail:"spot" multiplicity:"any"`
	ScopesSpot      saillegend.MultiSpot[ScopeRegion]        `sail:"spot" multiplicity:"any"`
}

// ----

type CollectionLandmark struct {
	saillegend.Landmark
}

func (CollectionLandmark) Family() string { return "collection" }
func (CollectionLandmark) Accept() saillegend.Acceptance {
	return saillegend.AcceptContractFamily[control.CollectionFamily]()
}

// ====

// The following vocabulary belongs to the earlier Control. In particular,
// Space is retained here for development context; ControlLegend uses Project
// and Scope for scoping.

// Members is one declaration-order band of control roles. Containers describe
// presentation topology, while Documents and Collections are terminal markers.
type Members struct {
	saillegend.Lane
	Projects    saillegend.MultiSpot[Project]    `sail:"spot" multiplicity:"any"`
	Spaces      saillegend.MultiSpot[Space]      `sail:"spot" multiplicity:"any"`
	Scopes      saillegend.MultiSpot[Scope]      `sail:"spot" multiplicity:"any"`
	Documents   saillegend.MultiSpot[Document]   `sail:"spot" multiplicity:"any"`
	Collections saillegend.MultiSpot[Collection] `sail:"spot" multiplicity:"any"`
}

// Project describes a recognition-capable controller. Recognition itself is a
// live control operation, not a declaration-time capability.
type Project struct {
	saillegend.Region
	Name    saillegend.Feature                                 `sail:"feature" symbol:"name"`
	Layout  saillegend.Barrel[reflect.Type, LayoutDeclaration] `sail:"barrel" populate:"LayoutType" barrel:"required"`
	Members Members                                            `sail:"lane"`
}

func (Project) Family() string { return "project" }
func (Project) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[LayoutDeclaration]()
}

// Space describes a controller with an explicitly chosen management root.
type Space struct {
	saillegend.Region
	Name    saillegend.Feature                                 `sail:"feature" symbol:"name"`
	Layout  saillegend.Barrel[reflect.Type, LayoutDeclaration] `sail:"barrel" populate:"LayoutType" barrel:"required"`
	Members Members                                            `sail:"lane"`
}

func (Space) Family() string { return "space" }
func (Space) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[LayoutDeclaration]()
}

// Scope describes an operational view over explicitly supplied storage.
type Scope struct {
	saillegend.Region
	Name    saillegend.Feature                                 `sail:"feature" symbol:"name"`
	Layout  saillegend.Barrel[reflect.Type, LayoutDeclaration] `sail:"barrel" populate:"LayoutType" barrel:"required"`
	Members Members                                            `sail:"lane"`
}

func (Scope) Family() string { return "scope" }
func (Scope) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[LayoutDeclaration]()
}

// Document describes typed content without traversing its node internals.
type Document struct {
	saillegend.Landmark
	Name  saillegend.Feature                                   `sail:"feature" symbol:"name"`
	Value saillegend.Barrel[reflect.Type, DocumentDeclaration] `sail:"barrel" populate:"ValueType" barrel:"required"`
}

func (Document) Family() string { return "document" }
func (Document) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[DocumentDeclaration]()
}

// Collection describes repeated operational member views without treating the
// source cache as static declaration topology. Conduit owns member binding.
type Collection struct {
	saillegend.Landmark
	Name   saillegend.Feature                                     `sail:"feature" symbol:"name"`
	Member saillegend.Barrel[reflect.Type, CollectionDeclaration] `sail:"barrel" populate:"MemberType" barrel:"required"`
}

func (Collection) Family() string { return "collection" }
func (Collection) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[CollectionDeclaration]()
}
