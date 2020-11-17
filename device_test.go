package plc

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDevice(t *testing.T) {
	_, err := NewDevice("", 0)
	assert.NoError(t, err)
}

func newTestDevice(rd rawDevice) Device {
	return Device{rawDevice: rd}
}

const testTagName = "TEST_TAG"

func TestReadTag(t *testing.T) {
	spy := RawDeviceFake{DeviceFake{}}
	dev := newTestDevice(&spy)

	spy.DeviceFake[testTagName] = int(7)

	var result int
	err := dev.ReadTag(testTagName, &result)
	assert.NoError(t, err)

	assert.Equal(t, 7, result)
}

func TestWriteTag(t *testing.T) {
	spy := RawDeviceFake{DeviceFake{}}
	dev := newTestDevice(&spy)

	var value = 9
	err := dev.WriteTag(testTagName, value)
	assert.NoError(t, err)

	assert.Equal(t, 9, spy.DeviceFake[testTagName])
}

// RawDeviceFake adds lower APIs to a DeviceFake
type RawDeviceFake struct {
	DeviceFake
}

func (dev RawDeviceFake) Close() error {
	return nil
}

func (dev RawDeviceFake) GetList(listName, prefix string) ([]Tag, []string, error) {
	return nil, nil, nil
}

type DeviceFake map[string]interface{}

func (df DeviceFake) ReadTag(name string, value interface{}) error {
	v, ok := df[name]
	if !ok {
		return fmt.Errorf("")
	}

	in := reflect.ValueOf(v)
	out := reflect.Indirect(reflect.ValueOf(value))

	switch {
	case !out.CanSet():
		return fmt.Errorf("cannot set %s", out.Type().Name())
	case out.Kind() != in.Kind():
		return fmt.Errorf("cannot set %s to %s", out.Type().Name(), in.Type().Name())
	}

	out.Set(in)
	return nil
}

func (df DeviceFake) WriteTag(name string, value interface{}) error {
	df[name] = value
	return nil
}