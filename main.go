package main

// #cgo LDFLAGS: -lalpm
// #include <alpm.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"unsafe"
)

func main() {
	// Initialize alpm
	var err C.alpm_errno_t
	handle := C.alpm_initialize(C.CString("/"), C.CString("/var/lib/pacman/"), &err)
	
	if handle == nil {
		errStr := C.GoString(C.alpm_strerror(err))
		fmt.Printf("Failed to initialize alpm: %s\n", errStr)
		return
	}
	defer C.alpm_release(handle)
	
	// Get local database
	db := C.alpm_get_localdb(handle)
	if db == nil {
		fmt.Println("Failed to get local database")
		return
	}
	
	// Get package list
	pkgList := C.alpm_db_get_pkgcache(db)
	
	// Iterate through packages
	count := 0
	for iter := pkgList; iter != nil; iter = C.alpm_list_next(iter) {
		pkg := (*C.alpm_pkg_t)(unsafe.Pointer(iter.data))
		name := C.GoString(C.alpm_pkg_get_name(pkg))
		version := C.GoString(C.alpm_pkg_get_version(pkg))
		fmt.Printf("%s %s\n", name, version)
		count++
	}
	
	fmt.Printf("\nTotal packages: %d\n", count)
}
