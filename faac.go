package faac

/*

#include "faac.h"
#include <stdio.h>

// "int32_t" makes cgo crazy, "signed int" works ...
int wrappedFaacEncEncode(faacEncHandle hEncoder, signed int *inputBuffer, unsigned int samplesInput, unsigned char *outputBuffer, unsigned int bufferSize) {
  return faacEncEncode(hEncoder, inputBuffer, samplesInput, outputBuffer, bufferSize);
}

#cgo LDFLAGS: -lfaac
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

type Encoder struct {
	handle C.faacEncHandle

	inputSamples   int
	maxOutputBytes int

	sampleWidth int
}

const (
	InputFloat  = int(C.FAAC_INPUT_FLOAT)
	Input16bits = int(C.FAAC_INPUT_16BIT)
	Input32bits = int(C.FAAC_INPUT_32BIT)

	Main = int(C.MAIN)
)

func Open(sampleRate int, channelCount int) *Encoder {
	var inputSamples C.ulong
	var maxOutputBytes C.ulong

	handle := C.faacEncOpen(
		C.ulong(sampleRate),
		C.uint(channelCount),
		&inputSamples,
		&maxOutputBytes)

	encoder := &Encoder{
		handle:         handle,
		inputSamples:   int(inputSamples),
		maxOutputBytes: int(maxOutputBytes),
	}

	runtime.SetFinalizer(encoder, finalizeEncoder)

	return encoder
}

type EncoderConfiguration struct {
	// QuantizerQuality int
	BitRate     int
	InputFormat int
	ObjectType  int
	UseLFE      bool
}

func (encoder *Encoder) Configuration() *EncoderConfiguration {
	config := C.faacEncGetCurrentConfiguration(encoder.handle)

	return &EncoderConfiguration{
		//	QuantizerQuality: int(config.quantqual),
		BitRate:     int(config.bitRate),
		InputFormat: int(config.inputFormat),
		ObjectType:  int(config.aacObjectType),
		UseLFE:      (config.useLfe == 1),
	}
}

func (encoder *Encoder) SetConfiguration(configuration *EncoderConfiguration) error {
	config := C.faacEncGetCurrentConfiguration(encoder.handle)

	// config.quantqual = C.ulong(configuration.QuantizerQuality)
	config.bitRate = C.ulong(configuration.BitRate)
	config.inputFormat = C.uint(configuration.InputFormat)
	config.aacObjectType = C.uint(configuration.ObjectType)

	if configuration.UseLFE {
		config.useLfe = 1
	} else {
		config.useLfe = 0
	}

	switch {
	case configuration.InputFormat == Input16bits:
		encoder.sampleWidth = 2
	case configuration.InputFormat == Input32bits:
		encoder.sampleWidth = 4
	}

	if C.faacEncSetConfiguration(encoder.handle, config) == 0 {
		return errors.New("Can't configure Faac encoder")
	}

	return nil
}

func (encoder *Encoder) InputSamples() int {
	return encoder.inputSamples
}

func (encoder *Encoder) MaxOutputBytes() int {
	return encoder.maxOutputBytes
}

func (encoder *Encoder) OutputBuffer() []byte {
	return make([]byte, encoder.maxOutputBytes)
}

func (encoder *Encoder) EncodeFloats(samples []float32, output []byte) int {
	encodedByteCount := C.wrappedFaacEncEncode(encoder.handle,
		(*C.int)(unsafe.Pointer(&samples[0])),
		C.uint(len(samples)),
		(*C.uchar)(unsafe.Pointer(&output[0])),
		C.uint(len(output)))
	return int(encodedByteCount)
}

func (encoder *Encoder) EncodeBytes(samples []byte, output []byte) int {
	encodedByteCount := C.wrappedFaacEncEncode(encoder.handle,
		(*C.int)(unsafe.Pointer(&samples[0])),
		C.uint(len(samples)/encoder.sampleWidth),
		(*C.uchar)(unsafe.Pointer(&output[0])),
		C.uint(len(output)))
	return int(encodedByteCount)
}

func (encoder *Encoder) Close() {
	if encoder.handle != nil {
		C.faacEncClose(encoder.handle)
		encoder.handle = nil
	}
}

func finalizeEncoder(encoder *Encoder) {
	encoder.Close()
}
