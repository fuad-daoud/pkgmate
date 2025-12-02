require("lspconfig")["gopls"].setup({
	cmd = { "gopls" },
	settings = {
		gopls = {
			buildFlags = { "-tags=all-backends" },
			env = { CGO_ENABLED = "1", GOOS = "linux" },
		},
	},
})
