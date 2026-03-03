#!/bin/bash
# Quick validation script for MS-CLI MVP

echo "=== MS-CLI MVP Validation ==="
echo ""

cd /root/zhy/ms-cli

echo "1. Checking Go file count..."
GO_FILES=$(find . -name "*.go" -not -path "./vendor/*" | wc -l)
echo "   Go files: $GO_FILES"

echo ""
echo "2. Checking package structure..."
echo "   Packages found:"
find . -name "*.go" -not -path "./vendor/*" -exec dirname {} \; | sort | uniq | while read dir; do
    pkg=$(head -1 "$dir"/*.go 2>/dev/null | grep "^package" | head -1)
    if [ -n "$pkg" ]; then
        echo "   - $dir: $pkg"
    fi
done

echo ""
echo "3. Key files check..."
KEY_FILES=(
    "app/main.go"
    "app/bootstrap.go"
    "app/run.go"
    "agent/smart_agent.go"
    "agent/loop/agent.go"
    "agent/context/manager.go"
    "agent/context/budget.go"
    "agent/memory/store.go"
    "executor/runner.go"
    "integrations/llm/provider.go"
    "tools/registry.go"
    "tools/fs/fs.go"
    "tools/shell/shell.go"
    "internal/config/config.go"
    "ui/app.go"
)

for file in "${KEY_FILES[@]}"; do
    if [ -f "$file" ]; then
        lines=$(wc -l < "$file")
        echo "   ✅ $file ($lines lines)"
    else
        echo "   ❌ $file MISSING"
    fi
done

echo ""
echo "4. Documentation check..."
DOC_FILES=(
    "docs/ANALYSIS_REPORT.md"
    "docs/DEVELOPMENT_PLAN.md"
    "docs/MVP_GUIDE.md"
    "README.md"
)

for file in "${DOC_FILES[@]}"; do
    if [ -f "$file" ]; then
        lines=$(wc -l < "$file")
        echo "   ✅ $file ($lines lines)"
    else
        echo "   ❌ $file MISSING"
    fi
done

echo ""
echo "5. Configuration files..."
if [ -f "configs/mscli.yaml" ]; then
    echo "   ✅ configs/mscli.yaml"
else
    echo "   ⚠️  configs/mscli.yaml (optional)"
fi

echo ""
echo "=== Validation Complete ==="
