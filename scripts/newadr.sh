#!/bin/bash

NUM=$(ls docs/adr/ADR-* 2>/dev/null | wc -l)
NEXT=$(printf "%04d" $((NUM+1)))

FILE="docs/adr/ADR-$NEXT-$1.md"

cp docs/adr/template.md $FILE

echo "Created $FILE"
