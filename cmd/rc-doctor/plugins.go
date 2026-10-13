package main

// Every plug-in registers itself from init(). Importing it here is what puts it
// in the binary, so adding a platform, command or migration source is one new
// package plus one line below.
//
// Rules (internal/rules) and output formats (internal/report) are linked in
// through the engine and the commands, so they need no line here unless they
// live in a package of their own.
import (
	// Platforms.
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/platform/android"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/platform/ios"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/platform/kmp"

	// Commands.
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/commands/check"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/commands/explain"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/commands/facts"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/commands/list"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/commands/migrate"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/commands/review"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/commands/snapshot"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/commands/version"
	// Migration sources land here as they are built, for example:
	// _ "github.com/aggarwalpulkit596/rc-doctor/internal/migrate/superwall"
)
