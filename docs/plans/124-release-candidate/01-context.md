# Context

PR117 implements #77, PR120 implements #80, PR123 implements #81, and #112 has
merged active-session update protections. All retain separate delivery gates.
#54 already shipped and is closed. The owner's goal is to finish this batch,
then reassess machine readiness/Claude work, not implement it now.

Published1.1.0 remains immutable. Current engineering evidence covers individual
heads on macOS arm64 and cross-built targets, not a new accepted release.
