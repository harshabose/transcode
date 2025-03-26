package transcode

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aler9/gomavlib"
	"github.com/aler9/gomavlib/pkg/dialects/ardupilotmega"
	"github.com/aler9/gomavlib/pkg/dialects/common"
	"github.com/asticode/go-astiav"
)

type Propeller string

func (prop Propeller) String() string {
	return string(prop)
}

const (
	PropellerOne   Propeller = "propeller0"
	PropellerTwo   Propeller = "propeller1"
	PropellerThree Propeller = "propeller2"
	PropellerFour  Propeller = "propeller3"
	PropellerFive  Propeller = "propeller4"
	PropellerSix   Propeller = "propeller5"
)

type Updator interface {
	Start(*Filter)
}

func WithUpdateFilter(updator Updator) FilterOption {
	return func(filter *Filter) error {
		filter.updators = append(filter.updators, updator)
		return nil
	}
}

type notch struct {
	prop        Propeller
	harmonics   uint8
	frequencies []float32
	nBlades     uint8
}

func createNotch(prop Propeller, fundamental float32, harmonics, nBlades uint8) *notch {
	n := &notch{
		prop:        prop,
		harmonics:   harmonics,
		frequencies: make([]float32, harmonics),
		nBlades:     nBlades,
	}

	for i := uint8(0); i < n.harmonics; i++ {
		n.frequencies[i] = fundamental * float32(i+1)
	}

	return n
}

func (notch *notch) update(rpm float32) {
	fundamental := rpm * float32(notch.nBlades) / 60.0
	for i := uint8(0); i < notch.harmonics; i++ {
		notch.frequencies[i] = (notch.frequencies[i] + fundamental*float32(i+1)) / 2.0
	}
}

type PropNoiseFilterUpdator struct {
	notches  []*notch
	node     *gomavlib.Node
	interval time.Duration
	mux      sync.RWMutex
	flags    astiav.FilterCommandFlags
	ctx      context.Context
}

func CreatePropNoiseFilterUpdator(ctx context.Context, mavlinkSerial string, baudrate int, interval time.Duration) (*PropNoiseFilterUpdator, error) {
	updater := &PropNoiseFilterUpdator{
		notches:  make([]*notch, 0),
		flags:    astiav.NewFilterCommandFlags(astiav.FilterCommandFlagFast, astiav.FilterCommandFlagOne),
		interval: interval,
		ctx:      ctx,
	}

	config := gomavlib.NodeConf{
		Endpoints: []gomavlib.EndpointConf{
			gomavlib.EndpointSerial{
				Device: mavlinkSerial,
				Baud:   baudrate,
			},
		},
		Dialect:     ardupilotmega.Dialect,
		OutVersion:  gomavlib.V2,
		OutSystemID: 10,
	}

	node, err := gomavlib.NewNode(config)
	if err != nil {
		return nil, err
	}
	updater.node = node

	return updater, nil
}

func (update *PropNoiseFilterUpdator) AddNotchFilter(id Propeller, frequency float32, harmonics uint8, nBlades uint8) {
	update.mux.Lock()

	// rpm nBlades will have RPM to rpm conversion with number of blades (Nb / 60)
	update.notches = append(update.notches, createNotch(id, frequency, harmonics, nBlades))

	update.mux.Unlock()
}

func (update *PropNoiseFilterUpdator) loop1() {
	ticker := time.NewTicker(update.interval)
	defer ticker.Stop()

	for {
		select {
		case <-update.ctx.Done():
			return
		case <-ticker.C:
			update.node.WriteMessageAll(&ardupilotmega.MessageCommandLong{
				TargetSystem:    1,
				TargetComponent: 0,
				Command:         common.MAV_CMD_REQUEST_MESSAGE,
				Confirmation:    0,
				Param1:          float32((&ardupilotmega.MessageEscTelemetry_1To_4{}).GetID()),
				Param2:          0,
				Param3:          0,
				Param4:          0,
				Param5:          0,
				Param6:          0,
				Param7:          0,
			})

			update.node.WriteMessageAll(&ardupilotmega.MessageCommandLong{
				TargetSystem:    1,
				TargetComponent: 0,
				Command:         common.MAV_CMD_REQUEST_MESSAGE,
				Confirmation:    0,
				Param1:          float32((&ardupilotmega.MessageEscTelemetry_5To_8{}).GetID()),
				Param2:          0,
				Param3:          0,
				Param4:          0,
				Param5:          0,
				Param6:          0,
				Param7:          0,
			})
		}
	}
}

func (update *PropNoiseFilterUpdator) loop2() {
	eventChan := update.node.Events()

loop:
	for {
		select {
		case <-update.ctx.Done():
			return
		case event, ok := <-eventChan:
			if !ok {
				return
			}

			if frm, ok := event.(*gomavlib.EventFrame); ok {
				switch msg := frm.Message().(type) {
				case *ardupilotmega.MessageEscTelemetry_1To_4:
					update.mux.Lock()

					length := min(len(update.notches), 4)
					if length <= 0 {
						continue loop
					}
					for i := 0; i < length; i++ {
						update.notches[i].update(float32(msg.Rpm[i]))
					}

					update.mux.Unlock()
				case *ardupilotmega.MessageEscTelemetry_5To_8:
					update.mux.Lock()

					length := min(len(update.notches)-4, 4)
					if length <= 0 {
						continue loop
					}
					for i := 0; i < length; i++ {
						update.notches[i+4].update(float32(msg.Rpm[i]))
					}

					update.mux.Unlock()
				}
			}
		}
	}
}

func (update *PropNoiseFilterUpdator) loop3(filter *Filter) {
	ticker := time.NewTicker(update.interval)
	defer ticker.Stop()

	for {
		select {
		case <-update.ctx.Done():
			return
		case <-ticker.C:
			if err := update.update(filter); err != nil {
				fmt.Printf("Error updating notch filter: %v\n", err)
			}
		}
	}
}

func (update *PropNoiseFilterUpdator) Start(filter *Filter) {
	go update.loop1()
	go update.loop2()
	go update.loop3(filter)
}

func (update *PropNoiseFilterUpdator) update(filter *Filter) error {
	if filter == nil {
		return errors.New("filter is nil")
	}
	filter.mux.Lock()
	defer filter.mux.Unlock()

	for index, notch := range update.notches {
		target := fmt.Sprintf("%s%d", notch.prop.String(), index)
		for _, frequency := range notch.frequencies {
			if _, err := filter.graph.SendCommand(target, "frequency", fmt.Sprintf("%.2f", frequency), update.flags); err != nil {
				return err
			}
		}
	}

	return nil
}
