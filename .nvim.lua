require("lspconfig")["gopls"].setup({
	cmd = { "gopls" },
	settings = {
		gopls = {
			buildFlags = { "-tags=dummy,all_backends" },
			env = { CGO_ENABLED = "1", GOOS = "linux" },
		},
	},
})
