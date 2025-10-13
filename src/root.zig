const std = @import("std");
const c = @cImport({
    @cInclude("alpm.h");
    @cInclude("string.h");
});

// C-compatible struct for Go to consume
pub const CPackage = extern struct {
    name: [*:0]const u8,
    version: [*:0]const u8,
    desc: [*:0]const u8,
};

pub const CPackageList = extern struct {
    packages: [*]CPackage,
    count: usize,
    error_msg: [*:0]const u8, // null if no error
};

// Export for Go/C
pub fn pmListPackages() callconv(.C) *CPackageList {
    // We need a persistent allocator for data that Go will read
    // This is tricky - we'll use malloc for C compatibility

    var result = @as(*CPackageList, @ptrCast(@alignCast(c.malloc(@sizeOf(CPackageList)))));

    // Initialize alpm
    var err: c.alpm_errno_t = undefined;
    const handle = c.alpm_initialize("/", "/var/lib/pacman/", &err);

    if (handle == null) {
        result.packages = undefined;
        result.count = 0;
        result.error_msg = c.alpm_strerror(err);
        return result;
    }
    defer _ = c.alpm_release(handle);

    // Get local database
    const db = c.alpm_get_localdb(handle);
    if (db == null) {
        result.packages = undefined;
        result.count = 0;
        result.error_msg = "Failed to get local database";
        return result;
    }

    // Get package list
    const pkgs = c.alpm_db_get_pkgcache(db);

    // Count packages
    var count: usize = 0;
    var iter = pkgs;
    while (iter != null) : (iter = c.alpm_list_next(iter)) {
        count += 1;
    }

    // Allocate array
    const packages = @as([*]CPackage, @ptrCast(@alignCast(c.malloc(@sizeOf(CPackage) * count))));

    // Fill array
    var i: usize = 0;
    iter = pkgs;
    while (iter != null) : (iter = c.alpm_list_next(iter)) {
        const pkg = @as(?*c.alpm_pkg_t, @ptrCast(@alignCast(iter.*.data)));
        if (pkg) |p| {
            packages[i] = CPackage{
                .name = c.strdup(c.alpm_pkg_get_name(p)),
                .version = c.strdup(c.alpm_pkg_get_version(p)),
                .desc = c.strdup(c.alpm_pkg_get_desc(p) orelse "No description"),
            };
            i += 1;
        }
    }

    result.packages = packages;
    result.count = count;
    // result.error_msg = null;
    return result;
}

// Cleanup function Go must call
export fn pm_free_package_list(list: *CPackageList) callconv(.C) void {
    if (list.count > 0) {
        c.free(list.packages);
    }
    c.free(list);
}
