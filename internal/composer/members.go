package composer

import (
	"fmt"

	"github.com/qlustra/conduit/internal/contracts"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
)

type memberComposer struct {
	engine *Engine
	chart  *legend.Layout
}

func (e *Engine) prepareMembers(target any, marker participantMarker, source layout.CollectionCarrier) (contracts.MemberComposer, error) {
	if e == nil || e.charter == nil {
		return nil, fmt.Errorf("compose: collection at %v requires an engine", marker.CarrierPath())
	}
	_, present, err := sail.Bind[contracts.MemberCompositionAware](target, marker)
	if err != nil {
		return nil, fmt.Errorf("compose: collection at %v: %w", marker.CarrierPath(), err)
	}
	if !present {
		return nil, fmt.Errorf("compose: collection at %v must accept a member composer", marker.CarrierPath())
	}
	members, err := source.Enter()
	if err != nil {
		return nil, fmt.Errorf("compose: collection at %v member source: %w", marker.CarrierPath(), err)
	}
	chart, err := sail.PopulateMembers(e.charter, members)
	if err != nil {
		return nil, fmt.Errorf("compose: collection at %v member Chart: %w", marker.CarrierPath(), err)
	}
	if err := ValidateChart(chart); err != nil {
		return nil, fmt.Errorf("compose: collection at %v member Chart: %w", marker.CarrierPath(), err)
	}
	if err := validateMemberCarrier(chart.CarrierType()); err != nil {
		return nil, fmt.Errorf("compose: collection at %v member Chart: %w", marker.CarrierPath(), err)
	}
	return &memberComposer{engine: e, chart: chart}, nil
}

func (c *memberComposer) ComposeMember(target any, path, composeBase string) error {
	if c == nil {
		return fmt.Errorf("compose: a member composer is required")
	}
	p, err := c.engine.planMember(target, c.chart, path, composeBase)
	if err != nil {
		return err
	}
	return p.apply(target)
}
