package campaignfiles

import "embed"

// FS contains all authored campaign assets needed by the server at runtime.
// This allows Lambda deployments to load campaign JSON and placeholder dialogue
// assets without relying on loose files being shipped alongside the binary.
//
//go:embed */campaign.json */dialogue/*
var FS embed.FS
