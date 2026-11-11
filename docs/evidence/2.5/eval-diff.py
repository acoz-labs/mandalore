#!/usr/bin/env python3
"""Compare two MANDALORE_EVAL_REPORT files prompt by prompt (#232).

Usage: eval-diff.py BEFORE.json AFTER.json
Prints, for each list in the report, how many entries differ, and each
difference; exits 1 when any entry differs or the lists' lengths differ.
"""

import json
import sys

before, after = (json.load(open(p)) for p in sys.argv[1:3])
total = 0
for key in sorted(set(before) | set(after)):
    a, b = before.get(key) or [], after.get(key) or []
    if not isinstance(a, list):
        continue
    diff = [(x, y) for x, y in zip(a, b) if x != y]
    if len(a) != len(b):
        print("%s: %d entries before, %d after" % (key, len(a), len(b)))
        total += 1
    for x, y in diff:
        print("%s: %s -> %s" % (key, json.dumps(x), json.dumps(y)))
    print("%-10s %4d entries, %d different" % (key, len(a), len(diff)))
    total += len(diff)
print("differences:", total)
sys.exit(1 if total else 0)
