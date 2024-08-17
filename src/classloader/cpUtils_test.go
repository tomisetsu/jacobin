/*
 * Jacobin VM - A Java virtual machine
 * Copyright (c) 2024 by the Jacobin Authors. All rights reserved.
 * Licensed under Mozilla Public License 2.0 (MPL 2.0)  Consult jacobin.org.
 */

package classloader

import (
	"jacobin/frames"
	"jacobin/globals"
	"jacobin/log"
	"jacobin/types"
	"math"
	"os"
	"testing"
)

func TestMeInfoFromMethRefInvalid(t *testing.T) {
	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.CLASS)

	// set up a class with a constant pool containing entries
	// that will fail the following tests
	klass := CPool{}
	klass.CpIndex = append(klass.CpIndex, CpEntry{})
	klass.CpIndex = append(klass.CpIndex, CpEntry{IntConst, 0})
	klass.CpIndex = append(klass.CpIndex, CpEntry{UTF8, 0})

	klass.IntConsts = append(klass.IntConsts, int32(26))
	klass.Utf8Refs = append(klass.Utf8Refs, "Hello string")

	s1, s2, s3 := GetMethInfoFromCPmethref(&klass, 0)
	if s1 != "" && s2 != "" && s3 != "" {
		t.Errorf("Did not get expected result for pointing to CPentry[0]")
	}

	s1, s2, s3 = GetMethInfoFromCPmethref(&klass, 999)
	if s1 != "" && s2 != "" && s3 != "" {
		t.Errorf("Did not get expected result for pointing to CPentry outside of CP")
	}

	s1, s2, s3 = GetMethInfoFromCPmethref(&klass, 1)
	if s1 != "" && s2 != "" && s3 != "" {
		t.Errorf("Did not get expected result for pointing to CPentry that's not a MethodRef")
	}
}

func TestMeInfoFromMethRefValid(t *testing.T) {
	globals.InitGlobals("test")

	// redirect stderr so as not to pollute the test output with the expected error message
	normalStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	// Initialize classloaders and method area
	err := Init()
	if err != nil {
		t.Errorf("Failure to load classes in TestMeInfoFromMethRefValid")
	}
	LoadBaseClasses() // must follow classloader.Init()

	f := frames.CreateFrame(4)
	f.Meth = append(f.Meth, 0x00)
	f.Meth = append(f.Meth, 0x01) // Go to slot 0x0001 in the CP

	CP := CPool{}
	CP.CpIndex = make([]CpEntry, 10)
	CP.CpIndex[0] = CpEntry{Type: 0, Slot: 0}
	CP.CpIndex[1] = CpEntry{Type: MethodRef, Slot: 0}

	CP.MethodRefs = make([]MethodRefEntry, 1)
	CP.MethodRefs[0] = MethodRefEntry{ClassIndex: 2, NameAndType: 3}

	CP.CpIndex[2] = CpEntry{Type: ClassRef, Slot: 0}
	CP.ClassRefs = make([]uint32, 4)
	CP.ClassRefs[0] = types.ObjectPoolStringIndex

	CP.CpIndex[3] = CpEntry{Type: NameAndType, Slot: 0}
	CP.NameAndTypes = make([]NameAndTypeEntry, 4)
	CP.NameAndTypes[0] = NameAndTypeEntry{
		NameIndex: 4,
		DescIndex: 5,
	}
	CP.CpIndex[4] = CpEntry{Type: UTF8, Slot: 0} // method name
	CP.Utf8Refs = make([]string, 4)
	CP.Utf8Refs[0] = "<init>"

	CP.CpIndex[5] = CpEntry{Type: UTF8, Slot: 1} // method name
	CP.Utf8Refs[1] = "()V"

	f.CP = &CP

	_, s2, s3 := GetMethInfoFromCPmethref(&CP, 1)
	if s2 != "<init>" && s3 != "()V" {
		t.Errorf("Expect to get a method: <init>()V, got %s%s", s2, s2)
	}

	// restore stderr
	_ = w.Close()
	os.Stderr = normalStderr
}

func TestGetClassNameFromCPclassref(t *testing.T) {
	globals.InitGlobals("test")

	// Initialize classloaders and method area
	err := Init()
	if err != nil {
		t.Errorf("Failure to load classes in TestGetClassNameFromCPclassref")
	}
	LoadBaseClasses() // must follow classloader.Init()

	f := frames.CreateFrame(4)
	f.Meth = append(f.Meth, 0x00)
	f.Meth = append(f.Meth, 0x01) // Go to slot 0x0001 in the CP

	CP := CPool{}
	CP.CpIndex = make([]CpEntry, 10)
	CP.CpIndex[0] = CpEntry{Type: 0, Slot: 0}
	CP.CpIndex[1] = CpEntry{Type: ClassRef, Slot: 0}
	CP.CpIndex[2] = CpEntry{Type: ClassRef, Slot: 1}

	CP.ClassRefs = make([]uint32, 4)
	CP.ClassRefs[0] = types.ObjectPoolStringIndex
	CP.ClassRefs[1] = types.InvalidStringIndex

	f.CP = &CP

	s1 := GetClassNameFromCPclassref(&CP, 1)
	if s1 != "java/lang/Object" {
		t.Errorf("Expect class name of 'java/lang/Object', got %s", s1)
	}

	s2 := GetClassNameFromCPclassref(&CP, 0)
	if s2 != "" {
		t.Errorf("Expected empty class name, got %s", s2)
	}
}

func TestFetchCPentry(t *testing.T) {
	globals.InitGlobals("test")

	cp := FetchCPentry(nil, 6)
	if cp.RetType != IS_ERROR {
		t.Errorf("Expect IS_ERROR, got %d", cp.RetType)
	}

	CP := CPool{
		CpIndex:        []CpEntry{},
		ClassRefs:      []uint32{},
		Doubles:        []float64{},
		Dynamics:       []DynamicEntry{},
		Floats:         []float32{},
		IntConsts:      []int32{},
		InterfaceRefs:  []InterfaceRefEntry{},
		InvokeDynamics: []InvokeDynamicEntry{},
		LongConsts:     []int64{},
		MethodHandles:  []MethodHandleEntry{},
		MethodRefs:     []MethodRefEntry{},
		MethodTypes:    []uint16{},
		NameAndTypes:   []NameAndTypeEntry{},
		Utf8Refs:       []string{},
	}

	CP.CpIndex = make([]CpEntry, 20)
	CP.CpIndex[0] = CpEntry{Type: 0, Slot: 0} // mandatory dummy entry
	CP.CpIndex[1] = CpEntry{Type: IntConst, Slot: 0}
	CP.CpIndex[2] = CpEntry{Type: LongConst, Slot: 0}
	CP.CpIndex[3] = CpEntry{Type: StringConst, Slot: 4}
	CP.CpIndex[4] = CpEntry{Type: UTF8, Slot: 0}
	CP.CpIndex[5] = CpEntry{Type: MethodType, Slot: 0}
	CP.CpIndex[6] = CpEntry{Type: FloatConst, Slot: 0}
	CP.CpIndex[7] = CpEntry{Type: DoubleConst, Slot: 0}
	CP.CpIndex[8] = CpEntry{Type: ClassRef, Slot: 0}

	CP.IntConsts = []int32{25}
	cp = FetchCPentry(&CP, 1)
	if cp.RetType != IS_INT64 || cp.IntVal != int64(25) {
		t.Errorf("Expect IS_INT64 with value of 25, got %d with value of %d", cp.RetType, cp.IntVal)
	}

	CP.LongConsts = []int64{250}
	cp = FetchCPentry(&CP, 2)
	if cp.RetType != IS_INT64 || cp.IntVal != int64(250) {
		t.Errorf("Expect IS_INT64 with value of 250, got %d with value of %d", cp.RetType, cp.IntVal)
	}

	CP.Utf8Refs = []string{"Hello from Jacobin JVM!"}
	cp = FetchCPentry(&CP, 3) // get the string const that points to the string in Utf8Refs[0]
	if cp.RetType != IS_STRING_ADDR || *cp.StringVal != "Hello from Jacobin JVM!" {
		t.Errorf("Expect IS_STRING_ADDR pointing to 'Hello from Jacobin JVM!', got %s", *cp.StringVal)
	} // when this passes, we know that StringConst and UTF8 both work

	CP.MethodTypes = []uint16{24}
	cp = FetchCPentry(&CP, 5)
	if cp.RetType != IS_INT64 || cp.IntVal != int64(24) {
		t.Errorf("Expect IS_INT64 with value of 24, got %d with value of %d", cp.RetType, cp.IntVal)
	}

	CP.Floats = []float32{float32(24.100000)}
	cp = FetchCPentry(&CP, 6)
	if cp.RetType != IS_FLOAT64 || math.Abs(24.1-cp.FloatVal) > 0.001 {
		t.Errorf("Expected IS_FLOAT64 with value of 24.1, got %d with value of %f", cp.RetType, cp.FloatVal)
	}

	CP.Doubles = []float64{24.20}
	cp = FetchCPentry(&CP, 7)
	if cp.RetType != IS_FLOAT64 || math.Abs(24.20-cp.FloatVal) > 0.001 {
		t.Errorf("Expected IS_FLOAT64 with value of 24.2, got %d with value of %f", cp.RetType, cp.FloatVal)
	}

	err := Init()
	if err != nil {
		t.Errorf("Failure to load classes in TestFetchCPentry")
	}
	LoadBaseClasses() // must follow classloader.Init()
	CP.ClassRefs = []uint32{types.StringPoolStringIndex}
	cp = FetchCPentry(&CP, 8)
	if cp.RetType != IS_STRING_ADDR || *cp.StringVal != "java/lang/String" {
		t.Errorf("Expected IS_STRING_ADDR pointing to 'java/lang/String', got %d pointing to %s",
			cp.RetType, *cp.StringVal)
	}
}
