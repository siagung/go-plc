package plc

/*
#include <stdlib.h>
#include "./libplctag.h"
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"
)

// libplctagDevice is an instance of the rawDevice interface.
// It communicates with a PLC over the network by using the libplctag C library.
type libplctagDevice struct {
	conConf string
	ids     map[string]C.int32_t
	timeout C.int
}

// newLibplctagDevice creates a new libplctagDevice.
// The conConf string provides IP and other connection configuration (see libplctag for options).
func newLibplctagDevice(conConf string, timeout int) (libplctagDevice, error) {
	dev := libplctagDevice{
		conConf: conConf,
		ids:     make(map[string]C.int32_t),
		timeout: C.int(timeout),
	}

	return dev, nil
}

// Close should be called on the libplctagDevice to clean up its resources.
func (dev *libplctagDevice) Close() error {
	for _, id := range dev.ids {
		err := newError(C.plc_tag_destroy(id))
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	noOffset         = C.int(0)
	stringDataOffset = 4
	stringMaxLength  = 82 // Size according to libplctag. Seems like an underlying protocol thing.
)

func (dev *libplctagDevice) getID(tagName string) (C.int32_t, error) {
	id, ok := dev.ids[tagName]
	if ok {
		return id, nil
	}

	cattrib_str := C.CString(dev.conConf + "&name=" + tagName) // can also specify elem_size=1&elem_count=1
	defer C.free(unsafe.Pointer(cattrib_str))

	id = C.plc_tag_create(cattrib_str, dev.timeout)
	if id < 0 {
		return id, newError(id)
	}
	dev.ids[tagName] = id
	return id, nil
}

// ReadTag reads the requested tag into the provided value.
func (dev *libplctagDevice) ReadTag(name string, value interface{}) (err error) {
	id, err := dev.getID(name)
	if err != nil {
		return err
	}

	if err = newError(C.plc_tag_read(id, dev.timeout)); err != nil {
		return err
	}

	switch val := value.(type) {
	case *bool:
		result := C.plc_tag_get_uint8(id, noOffset)
		*val = uint8(result) > 0
	case *uint8:
		result := C.plc_tag_get_uint8(id, noOffset)
		*val = uint8(result)
	case *uint16:
		result := C.plc_tag_get_uint16(id, noOffset)
		*val = uint16(result)
	case *uint32:
		result := C.plc_tag_get_uint32(id, noOffset)
		*val = uint32(result)
	case *uint64:
		result := C.plc_tag_get_uint64(id, noOffset)
		*val = uint64(result)
	case *int8:
		result := C.plc_tag_get_int8(id, noOffset)
		*val = int8(result)
	case *int16:
		result := C.plc_tag_get_int16(id, noOffset)
		*val = int16(result)
	case *int32:
		result := C.plc_tag_get_int32(id, noOffset)
		*val = int32(result)
	case *int64:
		result := C.plc_tag_get_int64(id, noOffset)
		*val = int64(result)
	case *float32:
		result := C.plc_tag_get_float32(id, noOffset)
		*val = float32(result)
	case *float64:
		result := C.plc_tag_get_float64(id, noOffset)
		*val = float64(result)
	case *string:
		// We only lock in this context because it's ok if we get the results of someone else's update to the cache
		err = newError(C.plc_tag_lock(id))
		if err != nil {
			return err
		}
		defer func() {
			lockErr := newError(C.plc_tag_unlock(id))
			if lockErr != nil {
				err = fmt.Errorf("error locking %w and other error %s", lockErr, err.Error())
			}
		}()

		bytes := make([]byte, 0, stringMaxLength)
		str_len := int(C.plc_tag_get_int32(id, noOffset))
		for str_index := 0; str_index < str_len; str_index++ {
			bytes[str_index] = byte(C.plc_tag_get_uint8(id, C.int(stringDataOffset+str_index)))
		}
		*val = string(bytes)
	default:
		return fmt.Errorf("Type %T is unknown and can't be read (%v)", val, val)
	}

	return nil
}

// WriteTag writes the provided tag and value.
func (dev *libplctagDevice) WriteTag(name string, value interface{}) (err error) {
	id, err := dev.getID(name)
	if err != nil {
		return err
	}

	err = newError(C.plc_tag_lock(id))
	if err != nil {
		return err
	}
	defer func() {
		lockErr := newError(C.plc_tag_unlock(id))
		if lockErr != nil {
			err = fmt.Errorf("error locking %w and other error %s", lockErr, err.Error())
		}
	}()

	switch val := value.(type) {
	case bool:
		b := C.uint8_t(0)
		if val {
			b = C.uint8_t(255)
		}
		err = newError(C.plc_tag_set_uint8(id, noOffset, b))
	case uint8:
		err = newError(C.plc_tag_set_uint8(id, noOffset, C.uint8_t(val)))
	case uint16:
		err = newError(C.plc_tag_set_uint16(id, noOffset, C.uint16_t(val)))
	case uint32:
		err = newError(C.plc_tag_set_uint32(id, noOffset, C.uint32_t(val)))
	case uint64:
		err = newError(C.plc_tag_set_uint64(id, noOffset, C.uint64_t(val)))
	case int8:
		err = newError(C.plc_tag_set_int8(id, noOffset, C.int8_t(val)))
	case int16:
		err = newError(C.plc_tag_set_int16(id, noOffset, C.int16_t(val)))
	case int32:
		err = newError(C.plc_tag_set_int32(id, noOffset, C.int32_t(val)))
	case int64:
		err = newError(C.plc_tag_set_int64(id, noOffset, C.int64_t(val)))
	case float32:
		err = newError(C.plc_tag_set_float32(id, noOffset, C.float(val)))
	case float64:
		err = newError(C.plc_tag_set_float64(id, noOffset, C.double(val)))
	case string: // TODO this should lock the tag until the write is done
		// write the string length
		err = newError(C.plc_tag_set_int32(id, noOffset, C.int32_t(len(val))))
		if err != nil {
			return err
		}

		// copy the data
		for str_index := 0; str_index < stringMaxLength; str_index++ {
			byt := byte(0) // pad with zeroes after the string ended
			if str_index < len(val) {
				byt = val[str_index]
			}

			err = newError(C.plc_tag_set_uint8(id, C.int(stringDataOffset+str_index), C.uint8_t(byt)))
			if err != nil {
				return err
			}
		}
	default:
		err = fmt.Errorf("Type %T is unknown and can't be written (%v)", val, val)
	}
	if err != nil {
		return err
	}

	// Read. If non-zero, value is true. Otherwise, it's false.
	if err = newError(C.plc_tag_write(id, dev.timeout)); err != nil {
		return err
	}

	return nil
}

func (dev *libplctagDevice) GetList(listName, prefix string) ([]Tag, []string, error) {
	if listName == "" {
		listName += "@tags"
	} else {
		listName += ".@tags"
	}

	id, err := dev.getID(listName)
	if err != nil {
		return nil, nil, err
	}

	if err := newError(C.plc_tag_read(id, dev.timeout)); err != nil {
		return nil, nil, err
	}

	tags := []Tag{}
	programNames := []string{}

	offset := C.int(0)
	for {
		tag := Tag{}
		offset += 4

		tag.tagType = C.plc_tag_get_uint16(id, offset)
		offset += 2

		tag.elementSize = C.plc_tag_get_uint16(id, offset)
		offset += 2

		tag.addDimension(int(C.plc_tag_get_uint32(id, offset)))
		offset += 4
		tag.addDimension(int(C.plc_tag_get_uint32(id, offset)))
		offset += 4
		tag.addDimension(int(C.plc_tag_get_uint32(id, offset)))
		offset += 4

		nameLength := int(C.plc_tag_get_uint16(id, offset))
		offset += 2

		tagBytes := make([]byte, nameLength)
		for i := 0; i < nameLength; i++ {
			tagBytes[i] = byte(C.plc_tag_get_int8(id, offset))
			offset++
		}

		if prefix != "" {
			tag.name = prefix + "." + string(tagBytes)
		} else {
			tag.name = string(tagBytes)
		}

		if strings.HasPrefix(tag.name, "Program:") {
			programNames = append(programNames, tag.name)
		} else if (tag.tagType & SystemTagBit) == SystemTagBit {
			// Do nothing for system tags
		} else {
			numDimensions := int((tag.tagType & TagDimensionMask) >> 13)
			if numDimensions != len(tag.dimensions) {
				return nil, nil, fmt.Errorf("Tag '%s' claims to have %d dimensions but has %d", tag.name, numDimensions, len(tag.dimensions))
			}

			tags = append(tags, tag)
		}

		if offset >= C.plc_tag_get_size(id) {
			break
		}
	}

	return tags, programNames, nil
}