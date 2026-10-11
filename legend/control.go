package legend

import (
	"reflect"

	"github.com/qlustra/conduit/control"
	saillegend "github.com/qlustra/sail/legend"
)

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

type ControlLegend struct {
	saillegend.Legend
	Lane struct {
		saillegend.Lane
		ProjectsSpot saillegend.MultiSpot[ProjectLandmark] `sail:"spot" multiplicity:"any"`
	} `sail:"lane"`
}

func (ControlLegend) Mark() string { return "control" }

// ----

type ProjectLandmark struct {
	saillegend.Region
	Lane ScopesLane `sail:"lane"`
}

func (ProjectLandmark) Family() string { return "project" }
func (ProjectLandmark) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[control.ProjectCarrier]()
}

// ----

type DocumentLandmark struct {
	saillegend.Landmark
}

func (DocumentLandmark) Family() string { return "document" }
func (DocumentLandmark) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[control.DocumentCarrier]()
}

// ----

type ScopeRegion struct {
	saillegend.Region
	Lane ScopesLane `sail:"lane"`
}

func (ScopeRegion) Family() string { return "scope" }
func (ScopeRegion) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[control.ScopeCarrier]()
}

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
	return saillegend.AcceptContract[control.CollectionCarrier]()
}

// ====

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
