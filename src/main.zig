const std = @import("std");
const lib = @import("pkgmate_lib");

pub fn main() !void {
    const result = lib.pmListPackages();
    // if (result.error_msg != null) {
    //     std.debug.print("Error: {s}\n", .{result.error_msg});
    //     return;
    // }
    std.debug.print("Found {} packages\n", .{result.count});
    for (0..5) |index| {
        const package = result.packages[@intCast(index)];
        std.debug.print("name: {s}\n", .{package.name});
    }
}
