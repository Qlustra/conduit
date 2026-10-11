package layout_test

import (
	"reflect"
	"testing"

	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
)

type apiManifest struct{ Name string }
type apiContext struct{ Name string }
type apiMember struct{ Root layout.Dir }

type apiFile struct{ layout.File }
type apiDirectory = layout.Dir
type apiDir struct{ apiDirectory }
type apiExec struct{ layout.Exec }
type apiFormat struct{ formats.YAMLFile[apiManifest] }
type apiTemplate struct {
	layout.TextTemplate[apiContext]
}
type apiFileLink struct{ layout.FileLink }
type apiDirLink struct{ layout.DirLink }
type apiSlot struct{ layout.Slot[*apiMember] }
type apiFileSlot struct{ layout.FileSlot[*apiFormat] }
type apiLinkSlot struct {
	layout.LinkSlot[*layout.FileLink]
}

var (
	_ layout.FileAPI                       = (*apiFile)(nil)
	_ layout.DirAPI                        = (*apiDir)(nil)
	_ layout.ExecAPI                       = (*apiExec)(nil)
	_ layout.FormatAPI[apiManifest]        = (*apiFormat)(nil)
	_ layout.TextTemplateAPI[apiContext]   = (*apiTemplate)(nil)
	_ layout.FileLinkAPI                   = (*apiFileLink)(nil)
	_ layout.DirLinkAPI                    = (*apiDirLink)(nil)
	_ layout.SlotAPI[*apiMember]           = (*apiSlot)(nil)
	_ layout.FileSlotAPI[*apiFormat]       = (*apiFileSlot)(nil)
	_ layout.LinkSlotAPI[*layout.FileLink] = (*apiLinkSlot)(nil)
	_ layout.ContentCarrier                = (*apiFormat)(nil)
	_ layout.CollectionCarrier             = (*apiSlot)(nil)
)

func TestCarrierAPIsCoverCompleteExportedMethodSets(t *testing.T) {
	cases := []struct {
		name         string
		api, carrier reflect.Type
	}{
		{"File", reflect.TypeFor[layout.FileAPI](), reflect.TypeFor[*layout.File]()},
		{"Dir", reflect.TypeFor[layout.DirAPI](), reflect.TypeFor[*layout.Dir]()},
		{"Exec", reflect.TypeFor[layout.ExecAPI](), reflect.TypeFor[*layout.Exec]()},
		{"Format", reflect.TypeFor[layout.FormatAPI[apiManifest]](), reflect.TypeFor[*layout.Format[apiManifest, formats.JSONCodec[apiManifest]]]()},
		{"TextTemplate", reflect.TypeFor[layout.TextTemplateAPI[apiContext]](), reflect.TypeFor[*layout.TextTemplate[apiContext]]()},
		{"Link", reflect.TypeFor[layout.LinkAPI](), reflect.TypeFor[*layout.Link]()},
		{"FileLink", reflect.TypeFor[layout.FileLinkAPI](), reflect.TypeFor[*layout.FileLink]()},
		{"DirLink", reflect.TypeFor[layout.DirLinkAPI](), reflect.TypeFor[*layout.DirLink]()},
		{"Slot", reflect.TypeFor[layout.SlotAPI[*apiMember]](), reflect.TypeFor[*layout.Slot[*apiMember]]()},
		{"FileSlot", reflect.TypeFor[layout.FileSlotAPI[*apiFormat]](), reflect.TypeFor[*layout.FileSlot[*apiFormat]]()},
		{"LinkSlot", reflect.TypeFor[layout.LinkSlotAPI[*layout.FileLink]](), reflect.TypeFor[*layout.LinkSlot[*layout.FileLink]]()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.carrier.Implements(tc.api) {
				t.Fatalf("%v does not implement %v", tc.carrier, tc.api)
			}
			if tc.api.NumMethod() != tc.carrier.NumMethod() {
				t.Fatalf("%v has %d methods; %v exposes %d: update the native API", tc.api, tc.api.NumMethod(), tc.carrier, tc.carrier.NumMethod())
			}
		})
	}
}

type apiBadFile struct{ layout.File }

func (*apiBadFile) WriteBytes(string, layout.Context) error { return nil }

type apiBadDir struct{ apiDirectory }

func (*apiBadDir) List() string { return "" }

type apiDirectDir struct{ layout.Dir }

type apiBadExec struct{ layout.Exec }

func (*apiBadExec) EnsureExecutable() error { return nil }

type apiBadFormat struct{ formats.YAMLFile[apiManifest] }

func (*apiBadFormat) Get() string { return "" }

type apiBadTemplate struct {
	layout.TextTemplate[apiContext]
}

func (*apiBadTemplate) SetContext(string) {}

type apiBadLink struct{ layout.Link }

func (*apiBadLink) SetTarget(int) {}

type apiBadFileLink struct{ layout.FileLink }

func (*apiBadFileLink) TargetFile() string { return "" }

type apiBadDirLink struct{ layout.DirLink }

func (*apiBadDirLink) TargetDir() string { return "" }

type apiBadSlot struct{ layout.Slot[*apiMember] }

func (*apiBadSlot) ComposeMember(string) (string, error) { return "", nil }

type apiBadFileSlot struct{ layout.FileSlot[*apiFormat] }

func (*apiBadFileSlot) Get(string) (string, bool) { return "", false }

type apiBadLinkSlot struct {
	layout.LinkSlot[*layout.FileLink]
}

func (*apiBadLinkSlot) Put(string, string) {}

func TestCarrierAPIsRejectIncompatibleOuterOverrides(t *testing.T) {
	content, collections := layout.ContentFamily{}, layout.CollectionFamily{}
	cases := []struct {
		name string
		node any
		api  reflect.Type
	}{
		{"File", new(apiBadFile), reflect.TypeFor[layout.FileAPI]()},
		{"Dir", new(apiBadDir), reflect.TypeFor[layout.DirAPI]()},
		{"DirFieldShadowsDirMethod", new(apiDirectDir), reflect.TypeFor[layout.DirAPI]()},
		{"Exec", new(apiBadExec), reflect.TypeFor[layout.ExecAPI]()},
		{"Format", new(apiBadFormat), content.ContractOf(new(apiBadFormat))},
		{"TextTemplate", new(apiBadTemplate), content.ContractOf(new(apiBadTemplate))},
		{"Link", new(apiBadLink), content.ContractOf(new(apiBadLink))},
		{"FileLink", new(apiBadFileLink), content.ContractOf(new(apiBadFileLink))},
		{"DirLink", new(apiBadDirLink), content.ContractOf(new(apiBadDirLink))},
		{"Slot", new(apiBadSlot), collections.ContractOf(new(apiBadSlot))},
		{"FileSlot", new(apiBadFileSlot), collections.ContractOf(new(apiBadFileSlot))},
		{"LinkSlot", new(apiBadLinkSlot), collections.ContractOf(new(apiBadLinkSlot))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reflect.TypeOf(tc.node).Implements(tc.api) {
				t.Fatalf("incompatible outer override implements %v", tc.api)
			}
		})
	}
}

func TestCarrierFamilyWitnessesAreZeroSafeAndKeepTypedAPIs(t *testing.T) {
	content, collections := layout.ContentFamily{}, layout.CollectionFamily{}
	contentCases := []struct {
		node layout.ContentCarrier
		want reflect.Type
	}{
		{(*layout.Format[apiManifest, formats.JSONCodec[apiManifest]])(nil), reflect.TypeFor[layout.FormatAPI[apiManifest]]()},
		{new(apiFormat), reflect.TypeFor[layout.FormatAPI[apiManifest]]()},
		{(*layout.TextTemplate[apiContext])(nil), reflect.TypeFor[layout.TextTemplateAPI[apiContext]]()},
		{(*layout.Link)(nil), reflect.TypeFor[layout.LinkAPI]()},
		{(*layout.FileLink)(nil), reflect.TypeFor[layout.FileLinkAPI]()},
		{(*layout.DirLink)(nil), reflect.TypeFor[layout.DirLinkAPI]()},
	}
	for _, tc := range contentCases {
		if got := content.ContractOf(tc.node); got != tc.want {
			t.Fatalf("%T contract = %v; want %v", tc.node, got, tc.want)
		}
		if !reflect.TypeOf(tc.node).Implements(tc.want) {
			t.Fatalf("%T does not implement %v", tc.node, tc.want)
		}
	}
	collectionCases := []struct {
		node layout.CollectionCarrier
		want reflect.Type
	}{
		{(*layout.Slot[*apiMember])(nil), reflect.TypeFor[layout.SlotAPI[*apiMember]]()},
		{new(apiSlot), reflect.TypeFor[layout.SlotAPI[*apiMember]]()},
		{(*layout.FileSlot[*apiFormat])(nil), reflect.TypeFor[layout.FileSlotAPI[*apiFormat]]()},
		{(*layout.LinkSlot[*layout.FileLink])(nil), reflect.TypeFor[layout.LinkSlotAPI[*layout.FileLink]]()},
	}
	for _, tc := range collectionCases {
		if got := collections.ContractOf(tc.node); got != tc.want {
			t.Fatalf("%T contract = %v; want %v", tc.node, got, tc.want)
		}
		if !reflect.TypeOf(tc.node).Implements(tc.want) {
			t.Fatalf("%T does not implement %v", tc.node, tc.want)
		}
	}
	json := new(formats.JSONFile[apiManifest])
	yaml := new(formats.YAMLFile[apiManifest])
	if content.ContractOf(json) != content.ContractOf(yaml) {
		t.Fatal("content API depends on codec")
	}
}

type apiSpoofFormat struct{ formats.YAMLFile[apiManifest] }

func (*apiSpoofFormat) contentContract() reflect.Type { return reflect.TypeFor[any]() }

type apiSpoofSlot struct{ layout.Slot[*apiMember] }

func (*apiSpoofSlot) collectionContract() reflect.Type { return reflect.TypeFor[any]() }

func TestCarrierFamilyWitnessesCannotBeOverriddenOutsideLayout(t *testing.T) {
	if got := (layout.ContentFamily{}).ContractOf(new(apiSpoofFormat)); got != reflect.TypeFor[layout.FormatAPI[apiManifest]]() {
		t.Fatalf("external method replaced content contract: %v", got)
	}
	if got := (layout.CollectionFamily{}).ContractOf(new(apiSpoofSlot)); got != reflect.TypeFor[layout.SlotAPI[*apiMember]]() {
		t.Fatalf("external method replaced collection contract: %v", got)
	}
}

type apiBadContentPath struct{ formats.YAMLFile[apiManifest] }

func (*apiBadContentPath) Path() int { return 0 }

type apiBadCollectionEnter struct{ layout.Slot[*apiMember] }

func (*apiBadCollectionEnter) Enter() string { return "" }

func TestCarrierContractSurvivesIncompatibleOperationalOverrides(t *testing.T) {
	cases := []struct {
		name string
		node any
		want reflect.Type
	}{
		{"File", new(apiBadFile), reflect.TypeFor[layout.FileAPI]()},
		{"Dir", new(apiDirectDir), reflect.TypeFor[layout.DirAPI]()},
		{"Exec", new(apiBadExec), reflect.TypeFor[layout.ExecAPI]()},
		{"Format", new(apiBadContentPath), reflect.TypeFor[layout.FormatAPI[apiManifest]]()},
		{"Collection", new(apiBadCollectionEnter), reflect.TypeFor[layout.SlotAPI[*apiMember]]()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := layout.CarrierContract(tc.node)
			if !ok || got != tc.want {
				t.Fatalf("%T metadata contract = %v, %v; want %v", tc.node, got, ok, tc.want)
			}
			if reflect.TypeOf(tc.node).Implements(got) {
				t.Fatal("incompatible outer API was accepted")
			}
		})
	}
	if reflect.TypeFor[*apiBadContentPath]().Implements(reflect.TypeFor[layout.ContentCarrier]()) {
		t.Fatal("bad Path unexpectedly satisfies ContentCarrier")
	}
	if reflect.TypeFor[*apiBadCollectionEnter]().Implements(reflect.TypeFor[layout.CollectionCarrier]()) {
		t.Fatal("bad Enter unexpectedly satisfies CollectionCarrier")
	}
	if contract, ok := layout.CarrierContract(new(struct{})); ok || contract != nil {
		t.Fatal("unrelated type has a carrier contract")
	}
	if contract, ok := layout.CarrierContract(nil); ok || contract != nil {
		t.Fatal("nil has a carrier contract")
	}
}
