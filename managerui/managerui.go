package managerui

import "embed"

//go:embed assets/*
//go:embed index.html
//go:embed favicon.ico
//go:embed manifest.json
var Assets embed.FS
