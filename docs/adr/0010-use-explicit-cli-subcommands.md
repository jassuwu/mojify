# Use explicit CLI subcommands

Mojify v1 uses explicit subcommands rather than treating a bare argument as playback. Running `mojify` alone should show help, while playback is invoked through a command such as `mojify play video.mp4`; this keeps the surface extensible for future image, export, and recipe workflows without making the default command ambiguous.
