package backend

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestIdentityKeysRules(t *testing.T) {
	tests := []struct {
		name string
		pkg  Package
		want []string
	}{
		// pacman & pacman/AUR python prefixes
		{
			name: "pacman python prefix",
			pkg:  Package{DB: "pacman", Name: "python-requests"},
			want: []string{"py:requests"},
		},
		{
			name: "pacman AUR python prefix",
			pkg:  Package{DB: "pacman/AUR", Name: "python-black"},
			want: []string{"py:black"},
		},
		// pacman suffixes: -bin, -git, -appimage
		{
			name: "pacman -bin suffix",
			pkg:  Package{DB: "pacman", Name: "yay-bin"},
			want: []string{"app:yay"},
		},
		{
			name: "pacman -git suffix",
			pkg:  Package{DB: "pacman/AUR", Name: "example-git"},
			want: []string{"app:example"},
		},
		{
			name: "pacman -appimage suffix",
			pkg:  Package{DB: "pacman", Name: "obsidian-appimage"},
			want: []string{"app:obsidian"},
		},
		{
			name: "pacman plain app",
			pkg:  Package{DB: "pacman", Name: "vlc"},
			want: []string{"app:vlc"},
		},
		// apt/dpkg python prefix & regular
		{
			name: "dpkg python3 prefix",
			pkg:  Package{DB: "apt/dpkg", Name: "python3-requests"},
			want: []string{"py:requests"},
		},
		{
			name: "dpkg regular app",
			pkg:  Package{DB: "apt/dpkg", Name: "curl"},
			want: []string{"app:curl"},
		},
		// brew, brew/Cask, snap
		{
			name: "brew app",
			pkg:  Package{DB: "brew", Name: "wget"},
			want: []string{"app:wget"},
		},
		{
			name: "brew cask app",
			pkg:  Package{DB: "brew/Cask", Name: "visual-studio-code"},
			want: []string{"app:visual-studio-code"},
		},
		{
			name: "snap app",
			pkg:  Package{DB: "snap", Name: "firefox"},
			want: []string{"app:firefox"},
		},
		// pip & pipx
		{
			name: "pip library",
			pkg:  Package{DB: "pip", Name: "requests"},
			want: []string{"py:requests"},
		},
		{
			name: "pipx dual keys",
			pkg:  Package{DB: "pipx", Name: "black"},
			want: []string{"py:black", "app:black"},
		},
		// case and underscore normalisation
		{
			name: "case normalisation",
			pkg:  Package{DB: "brew", Name: "Upper_Case_Tool"},
			want: []string{"app:upper-case-tool"},
		},
		{
			name: "pip underscore normalisation",
			pkg:  Package{DB: "pip", Name: "Py_YAML"},
			want: []string{"py:py-yaml"},
		},
		// npm scope kept
		{
			name: "npm scoped package",
			pkg:  Package{DB: "npm", Name: "@anthropic-ai/claude-code"},
			want: []string{"app:@anthropic-ai/claude-code"},
		},
		{
			name: "npm scoped with underscore",
			pkg:  Package{DB: "npm", Name: "@Scope/My_Tool"},
			want: []string{"app:@scope/my-tool"},
		},
		{
			name: "npm unscoped package",
			pkg:  Package{DB: "npm", Name: "corepack"},
			want: []string{"app:corepack"},
		},
		// golang last element
		{
			name: "golang path last element",
			pkg:  Package{DB: "golang", Name: "golang.org/x/tools/gopls"},
			want: []string{"app:gopls"},
		},
		{
			name: "golang single element name",
			pkg:  Package{DB: "golang", Name: "mytool"},
			want: []string{"app:mytool"},
		},
		// flatpak last segment non-aliased
		{
			name: "flatpak dot segment",
			pkg:  Package{DB: "flatpak", ID: "org.mozilla.firefox", Name: "Firefox"},
			want: []string{"app:firefox"},
		},
		{
			name: "flatpak dot segment vlc",
			pkg:  Package{DB: "flatpak", ID: "org.videolan.VLC", Name: "VLC"},
			want: []string{"app:vlc"},
		},
		// edge cases: unknown db, empty name, flatpak empty id
		{
			name: "unknown DB",
			pkg:  Package{DB: "cargo", Name: "ripgrep"},
			want: nil,
		},
		{
			name: "empty name",
			pkg:  Package{DB: "pacman", Name: ""},
			want: nil,
		},
		{
			name: "flatpak empty ID",
			pkg:  Package{DB: "flatpak", ID: "", Name: "Firefox"},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IdentityKeys(tt.pkg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IdentityKeys(%+v) = %v, want %v", tt.pkg, got, tt.want)
			}
		})
	}
}

func TestIdentityKeysAliases(t *testing.T) {
	// Step 2(b): one case per alias entry, generated from the table itself so no entry escapes
	if len(flatpakAliases) < 15 || len(flatpakAliases) > 20 {
		t.Fatalf("flatpakAliases length = %d, expected 15-20", len(flatpakAliases))
	}

	// Assert spec and fixture requirements exist in table
	required := []string{
		"com.spotify.Client",
		"org.telegram.desktop",
		"com.google.Chrome",
		"com.visualstudio.code",
		"com.obsproject.Studio",
		"org.gnome.Calculator",
	}
	for _, req := range required {
		if _, ok := flatpakAliases[req]; !ok {
			t.Errorf("required alias %q missing from flatpakAliases", req)
		}
	}

	// Test each entry in the table
	for id, wantKey := range flatpakAliases {
		t.Run(id, func(t *testing.T) {
			pkg := Package{DB: "flatpak", ID: id, Name: "any"}
			got := IdentityKeys(pkg)
			want := []string{wantKey}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("IdentityKeys(%+v) = %v, want %v", pkg, got, want)
			}
		})
	}
}

func TestIdentityKeysGoVersionSkip(t *testing.T) {
	// Step 2(c): Go's /vN skip, plus a single-element path that is only v2 keeping v2
	tests := []struct {
		name string
		pkg  Package
		want []string
	}{
		{
			name: "module with /v2 suffix skips /v2",
			pkg:  Package{DB: "golang", Name: "github.com/charmbracelet/glow/v2"},
			want: []string{"app:glow"},
		},
		{
			name: "module with /v10 suffix skips /v10",
			pkg:  Package{DB: "golang", Name: "github.com/foo/bar/v10"},
			want: []string{"app:bar"},
		},
		{
			name: "single-element path only v2 keeps v2",
			pkg:  Package{DB: "golang", Name: "v2"},
			want: []string{"app:v2"},
		},
		{
			name: "single-element path only v1 keeps v1",
			pkg:  Package{DB: "golang", Name: "v1"},
			want: []string{"app:v1"},
		},
		{
			name: "path ending in non-digit v does not skip",
			pkg:  Package{DB: "golang", Name: "github.com/foo/vim"},
			want: []string{"app:vim"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IdentityKeys(tt.pkg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IdentityKeys(%+v) = %v, want %v", tt.pkg, got, tt.want)
			}
		})
	}
}

func TestIdentityKeysGenericSegments(t *testing.T) {
	// Step 2(d): the generic-segment skip, and a generic segment that is aliased
	genericUnaliased := []string{
		"org.example.client",
		"org.example.desktop",
		"org.example.app",
		"org.example.application",
		"org.example.Client",
		"org.example.Desktop",
		"org.example.App",
		"org.example.Application",
	}

	for _, id := range genericUnaliased {
		t.Run("unaliased generic/"+id, func(t *testing.T) {
			pkg := Package{DB: "flatpak", ID: id, Name: "generic"}
			got := IdentityKeys(pkg)
			if len(got) != 0 {
				t.Errorf("IdentityKeys(%+v) = %v, want nil", pkg, got)
			}
		})
	}

	genericAliased := []struct {
		id      string
		wantKey string
	}{
		{"com.spotify.Client", "app:spotify"},
		{"org.telegram.desktop", "app:telegram-desktop"},
		{"io.github.shiftey.Desktop", "app:github-desktop"},
		{"com.mattermost.Desktop", "app:mattermost-desktop"},
		{"com.bitwarden.desktop", "app:bitwarden"},
	}

	for _, tt := range genericAliased {
		t.Run("aliased generic/"+tt.id, func(t *testing.T) {
			pkg := Package{DB: "flatpak", ID: tt.id, Name: "App"}
			got := IdentityKeys(pkg)
			want := []string{tt.wantKey}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("IdentityKeys(%+v) = %v, want %v", pkg, got, want)
			}
		})
	}
}

func TestManagerFamily(t *testing.T) {
	tests := []struct {
		db   string
		want string
	}{
		{"pacman", "pacman"},
		{"pacman/AUR", "pacman"},
		{"brew", "brew"},
		{"brew/Cask", "brew"},
		{"apt/dpkg", "apt/dpkg"},
		{"snap", "snap"},
		{"pip", "pip"},
		{"pipx", "pipx"},
		{"npm", "npm"},
		{"golang", "golang"},
		{"flatpak", "flatpak"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.db, func(t *testing.T) {
			if got := ManagerFamily(tt.db); got != tt.want {
				t.Errorf("ManagerFamily(%q) = %q, want %q", tt.db, got, tt.want)
			}
		})
	}
}

func TestFindDuplicates(t *testing.T) {
	t.Run("union-find merge and representative key", func(t *testing.T) {
		// pipx black has py:black and app:black
		// pacman python-black has py:black
		// brew black has app:black
		// All 3 should be merged into 1 group.
		// Representative key: app:black beats py:black.
		pkgs := []Package{
			{DB: "pacman", Name: "python-black", Size: 100, IsDirect: false},
			{DB: "pipx", Name: "black", Size: 200, IsDirect: true},
			{DB: "brew", Name: "black", Size: 300, IsDirect: true},
		}

		got := FindDuplicates(pkgs)
		if len(got) != 1 {
			t.Fatalf("expected 1 group, got %d: %+v", len(got), got)
		}

		g := got[0]
		if g.Key != "app:black" {
			t.Errorf("Key = %q, want %q", g.Key, "app:black")
		}
		if g.Size != 600 {
			t.Errorf("Size = %d, want 600", g.Size)
		}
		if len(g.Packages) != 3 {
			t.Fatalf("Packages count = %d, want 3", len(g.Packages))
		}
		// Packages must be sorted by DB, then Name: brew, pacman, pipx
		wantDBs := []string{"brew", "pacman", "pipx"}
		for i, w := range wantDBs {
			if g.Packages[i].DB != w {
				t.Errorf("package %d DB = %q, want %q", i, g.Packages[i].DB, w)
			}
		}
	})

	t.Run("same family twice is never a group", func(t *testing.T) {
		tests := []struct {
			name string
			pkgs []Package
		}{
			{
				name: "pacman and pacman/AUR",
				pkgs: []Package{
					{DB: "pacman", Name: "foo", IsDirect: true},
					{DB: "pacman/AUR", Name: "foo-bin", IsDirect: true},
				},
			},
			{
				name: "brew and brew/Cask",
				pkgs: []Package{
					{DB: "brew", Name: "bar", IsDirect: true},
					{DB: "brew/Cask", Name: "bar", IsDirect: true},
				},
			},
			{
				name: "dpkg multiarch libc6",
				pkgs: []Package{
					{DB: "apt/dpkg", Name: "libc6", Size: 1000, IsDirect: true},
					{DB: "apt/dpkg", Name: "libc6", Size: 2000, IsDirect: true},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := FindDuplicates(tt.pkgs)
				if len(got) != 0 {
					t.Errorf("expected 0 groups, got %d: %+v", len(got), got)
				}
			})
		}
	})

	t.Run("direct rule", func(t *testing.T) {
		// All copies are dependencies -> dropped
		allDeps := []Package{
			{DB: "pacman", Name: "vlc", IsDirect: false},
			{DB: "snap", Name: "vlc", IsDirect: false},
		}
		if got := FindDuplicates(allDeps); len(got) != 0 {
			t.Errorf("all dependencies: got %d groups, want 0", len(got))
		}

		// At least one direct -> kept
		oneDirect := []Package{
			{DB: "pacman", Name: "vlc", IsDirect: false},
			{DB: "snap", Name: "vlc", IsDirect: true},
		}
		if got := FindDuplicates(oneDirect); len(got) != 1 {
			t.Errorf("one direct: got %d groups, want 1", len(got))
		}
	})

	t.Run("ordering across and within groups", func(t *testing.T) {
		// Groups sorted by Size descending, then Key ascending
		pkgs := []Package{
			// Group 1: key app:alpha, size 100
			{DB: "pacman", Name: "alpha", Size: 50, IsDirect: true},
			{DB: "snap", Name: "alpha", Size: 50, IsDirect: true},
			// Group 2: key app:beta, size 300
			{DB: "pacman", Name: "beta", Size: 150, IsDirect: true},
			{DB: "snap", Name: "beta", Size: 150, IsDirect: true},
			// Group 3: key app:gamma, size 300 (same size as beta, but gamma > beta)
			{DB: "pacman", Name: "gamma", Size: 150, IsDirect: true},
			{DB: "snap", Name: "gamma", Size: 150, IsDirect: true},
		}

		got := FindDuplicates(pkgs)
		if len(got) != 3 {
			t.Fatalf("got %d groups, want 3", len(got))
		}

		// Want order: app:beta (300), app:gamma (300), app:alpha (100)
		wantKeys := []string{"app:beta", "app:gamma", "app:alpha"}
		for i, w := range wantKeys {
			if got[i].Key != w {
				t.Errorf("group %d Key = %q, want %q", i, got[i].Key, w)
			}
		}

		// Within each group, packages sorted by DB then Name: apt/dpkg < brew < flatpak < golang < npm < pacman < pip < pipx < snap
		for i, g := range got {
			if !sort.SliceIsSorted(g.Packages, func(a, b int) bool {
				if g.Packages[a].DB != g.Packages[b].DB {
					return g.Packages[a].DB < g.Packages[b].DB
				}
				return g.Packages[a].Name < g.Packages[b].Name
			}) {
				t.Errorf("group %d packages not sorted by DB, Name: %+v", i, g.Packages)
			}
		}
	})

	t.Run("input order invariance", func(t *testing.T) {
		pkgs := []Package{
			{DB: "pacman", Name: "python-black", Size: 100, IsDirect: false},
			{DB: "pipx", Name: "black", Size: 200, IsDirect: true},
			{DB: "brew", Name: "black", Size: 300, IsDirect: true},
			{DB: "flatpak", ID: "org.mozilla.firefox", Name: "Firefox", Size: 500, IsDirect: true},
			{DB: "snap", Name: "firefox", Size: 500, IsDirect: true},
			{DB: "pacman", Name: "vlc", Size: 50, IsDirect: true},
			{DB: "flatpak", ID: "org.videolan.VLC", Name: "VLC", Size: 400, IsDirect: true},
			{DB: "npm", Name: "corepack", Size: 10, IsDirect: true},
			{DB: "apt/dpkg", Name: "libc6", Size: 1000, IsDirect: true},
		}

		baseResult := FindDuplicates(pkgs)

		// Reversed
		reversed := make([]Package, len(pkgs))
		copy(reversed, pkgs)
		for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
			reversed[i], reversed[j] = reversed[j], reversed[i]
		}
		if got := FindDuplicates(reversed); !reflect.DeepEqual(got, baseResult) {
			t.Errorf("reversed input gave different result:\n got  %+v\n want %+v", got, baseResult)
		}

		// Shuffled multiple times
		r := rand.New(rand.NewSource(12345))
		for attempt := 0; attempt < 10; attempt++ {
			shuffled := make([]Package, len(pkgs))
			copy(shuffled, pkgs)
			r.Shuffle(len(shuffled), func(i, j int) {
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			})

			if got := FindDuplicates(shuffled); !reflect.DeepEqual(got, baseResult) {
				t.Fatalf("attempt %d: shuffled input gave different result:\n got  %+v\n want %+v", attempt, got, baseResult)
			}
		}
	})
}
