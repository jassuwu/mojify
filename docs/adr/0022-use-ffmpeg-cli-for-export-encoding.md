# Use the FFmpeg CLI for export encoding

Mojify export will borrow the reader/renderer/rasterizer/writer pipeline shape from media-to-ascii, but it will keep encoding and muxing behind FFmpeg CLI processes instead of adopting Rust, OpenCV, or FFmpeg bindings. This preserves the existing source-build constraint that avoids native codec linking while still making MP4 export a first-class product capability.
