require("lspconfig")["gopls"].setup({
	cmd = { "gopls" },
	settings = {
		gopls = {
			buildFlags = { "-tags=arch" },
			env = { CGO_ENABLED = "1", GOOS = "linux" },
		},
	},
})
