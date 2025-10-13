const lib = @import("root");

// Export for Go/C
export fn pm_list_packages_b() callconv(.C) *lib.CPackageList {
    return lib.pm_list_packages();
}
