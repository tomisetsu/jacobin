/*
 * Jacobin VM - A Java virtual machine
 * Copyright (c) 2023 by  the Jacobin authors. Consult jacobin.org.
 * Licensed under Mozilla Public License 2.0 (MPL 2.0) All rights reserved.
 */

package gfunction

import (
	"fmt"
	"jacobin/classloader"
	"jacobin/exceptions"
	"jacobin/globals"
	"jacobin/log"
	"jacobin/object"
	"jacobin/shutdown"
	"jacobin/statics"
	"jacobin/stringPool"
	"jacobin/types"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"
)

/*
 Each object or library that has golang methods contains a reference to MethodSignatures,
 which contain data needed to insert the golang method into the MTable of the currently
 executing JVM. MethodSignatures is a map whose key is the fully qualified name and
 type of the method (that is, the method's full signature) and a value consisting of
 a struct of an int (the number of slots to pop off the caller's operand stack when
 creating the new frame and a function). All methods have the same signature, regardless
 of the signature of their Java counterparts. That signature is that it accepts a slice
 of interface{} and returns an interface{}. The accepted slice can be empty and the
 return interface can be nil. This covers all Java functions. (Objects are returned
 as a 64-bit address in this scheme--as they are in the JVM).

 The passed-in slice contains one entry for every parameter passed to the method (which
 could mean an empty slice). Longs and doubles use two parameter entries.
*/

func Load_Lang_System() map[string]GMeth {

	MethodSignatures["java/lang/System.currentTimeMillis()J"] = // get time in ms since Jan 1, 1970, returned as long
		GMeth{
			ParamSlots: 0,
			GFunction:  currentTimeMillis,
		}

	MethodSignatures["java/lang/System.nanoTime()J"] = // get nanoseconds time, returned as long
		GMeth{
			ParamSlots: 0,
			GFunction:  nanoTime,
		}

	MethodSignatures["java/lang/System.exit(I)V"] = // shutdown the app
		GMeth{
			ParamSlots: 1,
			GFunction:  exitI,
		}

	MethodSignatures["java/lang/System.gc()V"] = // for a GC cycle
		GMeth{
			ParamSlots: 0,
			GFunction:  forceGC,
		}

	MethodSignatures["java/lang/System.getProperty(Ljava/lang/String;)Ljava/lang/String;"] =
		GMeth{
			ParamSlots: 1,
			GFunction:  getProperty,
		}

	MethodSignatures["java/lang/System.registerNatives()V"] =
		GMeth{
			ParamSlots: 0,
			GFunction:  justReturn,
		}

	MethodSignatures["java/lang/Thread.registerNatives()V"] =
		GMeth{
			ParamSlots: 0,
			GFunction:  justReturn,
		}

	MethodSignatures["java/lang/System.console()Ljava/io/Console;"] = // get nanoseconds time, returned as long
		GMeth{
			ParamSlots: 0,
			GFunction:  getConsole,
		}

	MethodSignatures["java/lang/System.<clinit>()V"] =
		GMeth{
			ParamSlots: 0,
			GFunction:  clinit,
		}

	return MethodSignatures
}

/*
		 check whether this clinit() has been previously run. If not, have it duplicate the
	   following bytecodes from JDK 17 java/lang/System:
			static {};
			0: invokestatic  #637                // Method registerNatives:()V
			3: aconst_null
			4: putstatic     #640                // Field in:Ljava/io/InputStream;
			7: aconst_null
			8: putstatic     #387                // Field out:Ljava/io/PrintStream;
			11: aconst_null
			12: putstatic     #384                // Field err:Ljava/io/PrintStream;
			15: return
*/
func clinit([]interface{}) interface{} {
	klass := classloader.MethAreaFetch("java/lang/System")
	if klass == nil {
		errMsg := "System <clinit>: Expected java/lang/System to be in the MethodArea, but it was not"
		_ = log.Log(errMsg, log.SEVERE)
		exceptions.ThrowEx(exceptions.ClassNotLoadedException, errMsg, nil)
	}
	if klass.Data.ClInit != types.ClInitRun {
		_ = statics.AddStatic("java/lang/System.in", statics.Static{Type: "GS", Value: os.Stdin})
		_ = statics.AddStatic("java/lang/System.err", statics.Static{Type: "GS", Value: os.Stderr})
		_ = statics.AddStatic("java/lang/System.out", statics.Static{Type: "GS", Value: os.Stdout})
		klass.Data.ClInit = types.ClInitRun
	}
	return nil
}

// arrayCopy copies an array or subarray from one array to another, both of which must exist.
// It is a complex native function in the JDK. Javadoc here:
// docs.oracle.com/en/java/javase/17/docs/api/java.base/java/lang/System.html#arraycopy(java.lang.Object,int,java.lang.Object,int,int)
func arrayCopy(params []interface{}) interface{} {
	if len(params) != 5 {
		errMsg := fmt.Sprintf("java/lang/System.arraycopy: Expected 5 parameters, got %d", len(params))
		return getGErrBlk(exceptions.IllegalArgumentException, errMsg)
	}

	src := params[0].(*object.Object)
	srcPos := params[1].(int64)
	dest := params[2].(*object.Object)
	destPos := params[3].(int64)
	length := params[4].(int64)

	if src == nil || dest == nil {
		errMsg := fmt.Sprintf("java/lang/System.arraycopy: null src or dest")
		return getGErrBlk(exceptions.NullPointerException, errMsg)
	}

	if srcPos < 0 || destPos < 0 || length < 0 {
		errMsg := fmt.Sprintf(
			"java/lang/System.arraycopy: Negative value in: srcPose=%d, destPos=%d, or length=%d", srcPos, destPos, length)
		return getGErrBlk(exceptions.ArrayIndexOutOfBoundsException, errMsg)
	}

	srcType := *(stringPool.GetStringPointer(src.KlassName))
	destType := *(stringPool.GetStringPointer(dest.KlassName))

	if !strings.HasPrefix(srcType, types.Array) || !strings.HasPrefix(destType, types.Array) || srcType != destType {
		errMsg := fmt.Sprintf("java/lang/System.arraycopy: invalid src or dest array")
		return getGErrBlk(exceptions.ArrayStoreException, errMsg)
	}

	srcLen := object.ArrayLength(src)
	destLen := object.ArrayLength(dest)

	if srcPos+length > srcLen || destPos+length > destLen {
		errMsg := fmt.Sprintf("java/lang/System.arraycopy: array + length exceeds array size")
		return getGErrBlk(exceptions.ArrayIndexOutOfBoundsException, errMsg)
	}

	if (src != dest) || ((src == dest) && (srcPos+length < destPos)) {
		// CURR: do the non-overlapping copy
	} else {
		// TODO: do overlapping copy
	}

	return nil
}

// Return the system input console as a *os.File.
func getConsole([]interface{}) interface{} {
	return statics.GetStaticValue("java/lang/System", "in")
}

// Return time in milliseconds, measured since midnight of Jan 1, 1970
func currentTimeMillis([]interface{}) interface{} {
	return time.Now().UnixMilli() // is int64
}

// Return time in nanoseconds. Note that in golang this function has a lower (that is, less good)
// resolution than Java: two successive calls often return the same value.
func nanoTime([]interface{}) interface{} {
	return time.Now().UnixNano() // is int64
}

// Exits the program directly, returning the passed in value
// exit is a static function, so no object ref and exit value is in params[0]
func exitI(params []interface{}) interface{} {
	exitCode := params[0].(int64)
	var exitStatus = int(exitCode)
	shutdown.Exit(exitStatus)
	return 0 // this code is not executed as previous line ends Jacobin
}

// Force a garbage collection cycle.
func forceGC([]interface{}) interface{} {
	runtime.GC()
	return nil
}

// Get a property
func getProperty(params []interface{}) interface{} {
	propObj := params[0].(*object.Object) // string
	propStr := object.GoStringFromStringObject(propObj)

	var value string
	g := globals.GetGlobalRef()
	operSys := runtime.GOOS

	switch propStr {
	case "file.encoding":
		value = g.FileEncoding
	case "file.separator":
		value = string(os.PathSeparator)
	case "java.class.path":
		value = "." // OpenJDK JVM default value
	case "java.compiler": // the name of the JIT compiler (we don't have a JIT)
		value = "no JIT"
	case "java.home":
		value = g.JavaHome
	case "java.library.path":
		value = g.JavaHome
	case "java.vendor":
		value = "Jacobin"
	case "java.vendor.url":
		value = "https://jacobin.org"
	case "java.vendor.version":
		value = g.Version
	case "java.version":
		value = strconv.Itoa(g.MaxJavaVersion)
	// case "java.version.date":
	// 	need to get this
	case "java.vm.name":
		value = fmt.Sprintf(
			"Jacobin VM v. %s (Java %d) 64-bit VM", g.Version, g.MaxJavaVersion)
	case "java.vm.specification.name":
		value = "Java Virtual Machine Specification"
	case "java.vm.specification.vendor":
		value = "Oracle and Jacobin"
	case "java.vm.specification.version":
		value = strconv.Itoa(g.MaxJavaVersion)
	case "java.vm.vendor":
		value = "Jacobin"
	case "java.vm.version":
		value = strconv.Itoa(g.MaxJavaVersion)
	case "line.separator":
		if operSys == "windows" {
			value = "\\r\\n"
		} else {
			value = "\\n"
		}
	case "native.encoding": // hard to find out what this is, so hard-coding to UTF8
		value = "UTF8"
	case "os.arch":
		value = runtime.GOARCH
	case "os.name":
		value = operSys
	case "os.version":
		value = "not yet available"
	case "path.separator":
		value = string(os.PathSeparator)
	case "user.dir": // present working directory
		value, _ = os.Getwd()
	case "user.home":
		currentUser, _ := user.Current()
		value = currentUser.HomeDir
	case "user.name":
		currentUser, _ := user.Current()
		value = currentUser.Name
	default:
		return object.Null
	}

	obj := object.StringObjectFromGoString(value)
	return obj
}
